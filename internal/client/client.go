package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"specter/internal/protocol"
	"specter/internal/server"
	"time"
)

func sendRequest(req protocol.Request) (protocol.Response, error) {
	conn, err := net.Dial("unix", server.SocketName)
	if err != nil {
		return protocol.Response{}, err
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return protocol.Response{}, err
	}

	decoder := json.NewDecoder(conn)
	var resp protocol.Response
	if err := decoder.Decode(&resp); err != nil {
		return protocol.Response{}, err
	}

	return resp, nil
}

func sendRequestOrExit(req protocol.Request) {
	resp, err := sendRequest(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\nIs specter spawned?\n", err)
		os.Exit(1)
	}

	if resp.Status != "ok" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
		os.Exit(1)
	}

	if resp.Data != "" {
		fmt.Print(resp.Data)
	} else if resp.Message != "" {
		fmt.Println(resp.Message)
	}
}

func Spawn(args []string) {
	var cmd []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			cmd = args[i+1:]
			break
		}
	}

	if len(cmd) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = []string{shell}
	}

	if _, err := os.Stat(server.SocketName); err == nil {
		fmt.Fprintf(os.Stderr, "Specter already running (socket exists: %s)\n", server.SocketName)
		os.Exit(1)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	serverArgs := append([]string{"_server", "--"}, cmd...)
	serverCmd := exec.Command(exe, serverArgs...)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(server.SocketName); err == nil {
			fmt.Printf("Spawned: %v\n", cmd)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "Timeout waiting for server to start\n")
	os.Exit(1)
}

func Type(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: specter type <text>\n")
		os.Exit(1)
	}

	text := unescape(args[0])

	req := protocol.Request{
		Op:      protocol.OpType,
		Payload: []string{text},
	}
	sendRequestOrExit(req)
}

func unescape(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				out = append(out, '\n')
				i++
			case 'r':
				out = append(out, '\r')
				i++
			case 't':
				out = append(out, '\t')
				i++
			case '\\':
				out = append(out, '\\')
				i++
			case 'x':
				if i+3 < len(s) {
					hex := s[i+2 : i+4]
					if val, err := parseHexByte(hex); err == nil {
						out = append(out, val)
						i += 3
						continue
					}
				}
				out = append(out, '\\')
			default:
				out = append(out, '\\')
				out = append(out, s[i+1])
				i++
			}
		} else if s[i] == '\\' {
			out = append(out, '\\')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func parseHexByte(hex string) (byte, error) {
	if len(hex) != 2 {
		return 0, fmt.Errorf("invalid hex length")
	}
	var val byte
	for _, c := range hex {
		val <<= 4
		switch {
		case c >= '0' && c <= '9':
			val |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			val |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			val |= byte(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex char")
		}
	}
	return val, nil
}

func Capture(args []string) {
	format := "text"
	outputFile := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--format" && i+1 < len(args) {
			format = args[i+1]
			i++
		} else if args[i] == "--out" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		}
	}

	req := protocol.Request{
		Op:      protocol.OpCapture,
		Options: map[string]string{"format": format},
	}

	resp, err := sendRequest(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}

	if resp.Status != "ok" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
		os.Exit(1)
	}

	if format == "png" {
		data, err := base64.StdEncoding.DecodeString(resp.Data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding PNG data: %v\n", err)
			os.Exit(1)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Screenshot saved to %s\n", outputFile)
		} else {
			fi, _ := os.Stdout.Stat()
			if (fi.Mode() & os.ModeCharDevice) != 0 {
				displayImageKitty(resp.Data)
			} else {
				os.Stdout.Write(data)
			}
		}
	} else {
		if resp.Data != "" {
			if outputFile != "" {
				os.WriteFile(outputFile, []byte(resp.Data), 0644)
			} else {
				fmt.Print(resp.Data)
			}
		}
	}
}

func History() {
	req := protocol.Request{
		Op: protocol.OpHistory,
	}

	resp, err := sendRequest(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}

	if resp.Status != "ok" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
		os.Exit(1)
	}

	var history []string
	if err := json.Unmarshal([]byte(resp.Data), &history); err != nil {
		fmt.Println(resp.Data)
		return
	}

	for i, entry := range history {
		fmt.Printf("%d: %q\n", i, entry)
	}
}

func displayImageKitty(b64Data string) {
	const chunkSize = 4096

	for i := 0; i < len(b64Data); i += chunkSize {
		end := i + chunkSize
		if end > len(b64Data) {
			end = len(b64Data)
		}
		chunk := b64Data[i:end]

		more := 1
		if end >= len(b64Data) {
			more = 0
		}

		if i == 0 {
			fmt.Printf("\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, chunk)
		} else {
			fmt.Printf("\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	fmt.Println()
}

func Wait() {
	req := protocol.Request{
		Op: protocol.OpWait,
	}
	sendRequestOrExit(req)
}

func Kill() {
	req := protocol.Request{
		Op: protocol.OpKill,
	}

	resp, err := sendRequest(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\nIs specter running?\n", err)
		os.Exit(1)
	}

	if resp.Status != "ok" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
		os.Exit(1)
	}

	fmt.Println("Specter terminated")
}
