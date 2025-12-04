package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"specter/internal/protocol"
	"sync"

	"github.com/creack/pty"
	"github.com/mattn/go-libvterm"
)

const SocketName = ".specter.sock"

type Server struct {
	session  *Session
	listener net.Listener
}

type Session struct {
	Cmd          *exec.Cmd
	Pty          *os.File
	VTerm        *vterm.VTerm
	Screen       *vterm.Screen
	Mu           sync.Mutex
	InputHistory []string
	Exited       bool
	ExitCode     int
	ExitChan     chan struct{}
}

func Start(cmd []string) error {
	if _, err := os.Stat(SocketName); err == nil {
		os.Remove(SocketName)
	}

	listener, err := net.Listen("unix", SocketName)
	if err != nil {
		return err
	}

	s := &Server{
		listener: listener,
	}

	if err := s.spawnSession(cmd); err != nil {
		listener.Close()
		os.Remove(SocketName)
		return err
	}

	fmt.Printf("Specter running with: %v\n", cmd)

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go s.handleConnection(conn)
	}

	return nil
}

func (s *Server) spawnSession(cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	rows, cols := 30, 100
	vt := vterm.New(rows, cols)
	vt.SetUTF8(true)
	screen := vt.ObtainScreen()
	screen.Reset(true)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		vt.Close()
		return fmt.Errorf("failed to start pty: %v", err)
	}

	sess := &Session{
		Cmd:      cmd,
		Pty:      ptmx,
		VTerm:    vt,
		Screen:   screen,
		ExitChan: make(chan struct{}),
	}

	s.session = sess

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			sess.Mu.Lock()
			sess.VTerm.Write(buf[:n])
			sess.Mu.Unlock()
		}
		exitCode := 0
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		sess.Mu.Lock()
		sess.Exited = true
		sess.ExitCode = exitCode
		sess.Mu.Unlock()
		close(sess.ExitChan)
	}()

	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req protocol.Request
	if err := decoder.Decode(&req); err != nil {
		return
	}

	resp := s.processRequest(req)
	encoder.Encode(resp)
}

func (s *Server) processRequest(req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpType:
		return s.handleType(req)
	case protocol.OpCapture:
		return s.handleCapture(req)
	case protocol.OpHistory:
		return s.handleHistory(req)
	case protocol.OpWait:
		return s.handleWait(req)
	case protocol.OpKill:
		return s.handleKill(req)
	default:
		return protocol.Response{Status: "error", Message: "Unknown operation"}
	}
}

func (s *Server) handleType(req protocol.Request) protocol.Response {
	sess := s.session
	if sess == nil {
		return protocol.Response{Status: "error", Message: "No session"}
	}

	sess.Mu.Lock()
	exited := sess.Exited
	sess.Mu.Unlock()
	if exited {
		return protocol.Response{Status: "error", Message: "Process has exited"}
	}

	if len(req.Payload) == 0 {
		return protocol.Response{Status: "ok"}
	}

	text := req.Payload[0]
	_, err := sess.Pty.Write([]byte(text))
	if err != nil {
		return protocol.Response{Status: "error", Message: fmt.Sprintf("Failed to write: %v", err)}
	}

	sess.Mu.Lock()
	sess.InputHistory = append(sess.InputHistory, text)
	sess.Mu.Unlock()

	return protocol.Response{Status: "ok"}
}

func (s *Server) handleCapture(req protocol.Request) protocol.Response {
	sess := s.session
	if sess == nil {
		return protocol.Response{Status: "error", Message: "No session"}
	}

	sess.Mu.Lock()
	defer sess.Mu.Unlock()

	format := "text"
	if req.Options != nil {
		if f, ok := req.Options["format"]; ok {
			format = f
		}
	}

	if format == "png" {
		pngBytes, err := s.renderPNG(sess)
		if err != nil {
			return protocol.Response{Status: "error", Message: fmt.Sprintf("Failed to render PNG: %v", err)}
		}
		encoded := base64.StdEncoding.EncodeToString(pngBytes)
		return protocol.Response{Status: "ok", Data: encoded}
	}

	rows, cols := sess.VTerm.Size()
	var output string

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell, err := sess.Screen.GetCellAt(r, c)
			if err == nil {
				chars := cell.Chars()
				if len(chars) > 0 {
					output += string(chars)
				} else {
					output += " "
				}
			} else {
				output += " "
			}
		}
		output += "\n"
	}

	return protocol.Response{Status: "ok", Data: output}
}

func (s *Server) handleHistory(req protocol.Request) protocol.Response {
	sess := s.session
	if sess == nil {
		return protocol.Response{Status: "error", Message: "No session"}
	}

	sess.Mu.Lock()
	defer sess.Mu.Unlock()

	bytes, err := json.Marshal(sess.InputHistory)
	if err != nil {
		return protocol.Response{Status: "error", Message: fmt.Sprintf("Failed to marshal history: %v", err)}
	}

	return protocol.Response{Status: "ok", Data: string(bytes)}
}

func (s *Server) handleWait(req protocol.Request) protocol.Response {
	sess := s.session
	if sess == nil {
		return protocol.Response{Status: "error", Message: "No session"}
	}

	<-sess.ExitChan

	sess.Mu.Lock()
	exitCode := sess.ExitCode
	sess.Mu.Unlock()

	return protocol.Response{Status: "ok", Data: fmt.Sprintf("%d", exitCode)}
}

func (s *Server) handleKill(req protocol.Request) protocol.Response {
	sess := s.session
	if sess != nil {
		sess.Mu.Lock()
		if !sess.Exited {
			sess.Cmd.Process.Kill()
		}
		sess.Pty.Close()
		sess.VTerm.Close()
		sess.Mu.Unlock()
	}

	go func() {
		s.listener.Close()
		os.Remove(SocketName)
	}()

	return protocol.Response{Status: "ok", Message: "Server shutting down"}
}
