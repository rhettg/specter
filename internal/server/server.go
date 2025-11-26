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
	sessions map[string]*Session
	mu       sync.RWMutex
}

type Session struct {
	ID           string
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

func Start() error {
	// Remove existing socket if it exists
	if _, err := os.Stat(SocketName); err == nil {
		os.Remove(SocketName)
	}

	listener, err := net.Listen("unix", SocketName)
	if err != nil {
		return err
	}
	defer listener.Close()

	s := &Server{
		sessions: make(map[string]*Session),
	}

	fmt.Printf("Server listening on %s\n", SocketName)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		go s.handleConnection(conn)
	}
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
	case protocol.OpSpawn:
		return s.handleSpawn(req)
	case protocol.OpType:
		return s.handleType(req)
	case protocol.OpCapture:
		return s.handleCapture(req)
	case protocol.OpHistory:
		return s.handleHistory(req)
	case protocol.OpWait:
		return s.handleWait(req)
	default:
		return protocol.Response{Status: "error", Message: "Unknown operation"}
	}
}

func (s *Server) handleSpawn(req protocol.Request) protocol.Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing session with this ID
	if oldSess, exists := s.sessions[req.ID]; exists {
		oldSess.Mu.Lock()
		exited := oldSess.Exited
		oldSess.Mu.Unlock()
		
		if !exited {
			return protocol.Response{Status: "error", Message: "Session ID already exists and process is still running"}
		}
		// Clean up the old exited session
		oldSess.Pty.Close()
		oldSess.VTerm.Close()
		delete(s.sessions, req.ID)
	}

	if len(req.Payload) == 0 {
		return protocol.Response{Status: "error", Message: "No command specified"}
	}

	cmd := exec.Command(req.Payload[0], req.Payload[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Initial size
	rows, cols := 30, 100
	vt := vterm.New(rows, cols)
	vt.SetUTF8(true)
	screen := vt.ObtainScreen()
	screen.Reset(true)

	// Start PTY
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		vt.Close()
		return protocol.Response{Status: "error", Message: fmt.Sprintf("Failed to start pty: %v", err)}
	}

	sess := &Session{
		ID:       req.ID,
		Cmd:      cmd,
		Pty:      ptmx,
		VTerm:    vt,
		Screen:   screen,
		ExitChan: make(chan struct{}),
	}

	s.sessions[req.ID] = sess

	// Start background reader to feed vterm
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				// Process likely exited
				break
			}
			sess.Mu.Lock()
			sess.VTerm.Write(buf[:n])
			sess.Mu.Unlock()
		}
		// Wait for process and capture exit code
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

	return protocol.Response{Status: "ok", Message: fmt.Sprintf("Started %s", req.Payload[0])}
}

func (s *Server) handleType(req protocol.Request) protocol.Response {
	s.mu.RLock()
	sess, ok := s.sessions[req.ID]
	s.mu.RUnlock()

	if !ok {
		return protocol.Response{Status: "error", Message: "Session not found"}
	}

	sess.Mu.Lock()
	exited := sess.Exited
	sess.Mu.Unlock()
	if exited {
		return protocol.Response{Status: "error", Message: "Process has exited"}
	}

	if len(req.Payload) == 0 {
		return protocol.Response{Status: "ok"} // Nothing to type
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
	s.mu.RLock()
	sess, ok := s.sessions[req.ID]
	s.mu.RUnlock()

	if !ok {
		return protocol.Response{Status: "error", Message: "Session not found"}
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

	// Scrape the screen (Text)
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
	s.mu.RLock()
	sess, ok := s.sessions[req.ID]
	s.mu.RUnlock()

	if !ok {
		return protocol.Response{Status: "error", Message: "Session not found"}
	}

	sess.Mu.Lock()
	defer sess.Mu.Unlock()

	// Return history as a JSON string or just newline separated?
	// Protocol Data is string. Let's use JSON for structured data, 
	// but for simplicity in CLI, maybe just lines?
	// Protocol says Data is string.
	// Let's return it as a JSON array string.
	
	bytes, err := json.Marshal(sess.InputHistory)
	if err != nil {
		return protocol.Response{Status: "error", Message: fmt.Sprintf("Failed to marshal history: %v", err)}
	}
	
	return protocol.Response{Status: "ok", Data: string(bytes)}
}

func (s *Server) handleWait(req protocol.Request) protocol.Response {
	s.mu.RLock()
	sess, ok := s.sessions[req.ID]
	s.mu.RUnlock()

	if !ok {
		return protocol.Response{Status: "error", Message: "Session not found"}
	}

	// Wait for process to exit
	<-sess.ExitChan

	sess.Mu.Lock()
	exitCode := sess.ExitCode
	sess.Mu.Unlock()

	return protocol.Response{Status: "ok", Data: fmt.Sprintf("%d", exitCode)}
}
