package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"specter/internal/protocol"
	"specter/internal/server"
)

const DefaultID = "default"

func sendRequest(req protocol.Request) {
	conn, err := net.Dial("unix", server.SocketName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\nIs the server running?\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(conn)
	var resp protocol.Response
	if err := decoder.Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
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
	// Expected args: --id <id> -- <cmd> ...
	// For simplicity in this prototype, let's assume order: --id <id> <cmd> ...
	
	id := ""
	var cmd []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--id" && i+1 < len(args) {
			id = args[i+1]
			i++ // skip value
		} else if args[i] == "--" {
			cmd = args[i+1:]
			break
		} else {
			// If no -- separator, assume everything after ID is command?
			// Or just strict parsing. Let's handle the case where there is no --
			// If we already have ID, the rest is command
			if id != "" && len(cmd) == 0 {
				cmd = args[i:]
				break
			}
		}
	}

	if id == "" {
		id = DefaultID
	}

	req := protocol.Request{
		Op:      protocol.OpSpawn,
		ID:      id,
		Payload: cmd,
	}
	sendRequest(req)
}

func Type(args []string) {
	id := ""
	text := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--id" && i+1 < len(args) {
			id = args[i+1]
			i++
		} else {
			text = args[i]
		}
	}

	if id == "" {
		id = DefaultID
	}
	
	// Unescape newlines and tabs for convenience
	text = unescape(text)

	req := protocol.Request{
		Op:      protocol.OpType,
		ID:      id,
		Payload: []string{text},
	}
	sendRequest(req)
}

func unescape(s string) string {
	// Unescape common control chars and hex sequences
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
				// Handle \xNN hex escape sequences
				if i+3 < len(s) {
					hex := s[i+2 : i+4]
					if val, err := parseHexByte(hex); err == nil {
						out = append(out, val)
						i += 3
						continue
					}
				}
				// Invalid hex sequence, output literally
				out = append(out, '\\')
			default:
				out = append(out, '\\')
				out = append(out, s[i+1])
				i++
			}
		} else if s[i] == '\\' {
			// Trailing backslash
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
	id := ""
	format := "text"
	outputFile := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--id" && i+1 < len(args) {
			id = args[i+1]
			i++
		} else if args[i] == "--format" && i+1 < len(args) {
			format = args[i+1]
			i++
		} else if args[i] == "--out" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		}
	}

	if id == "" {
		id = DefaultID
	}

	req := protocol.Request{
		Op:      protocol.OpCapture,
		ID:      id,
		Options: map[string]string{"format": format},
	}
	
	conn, err := net.Dial("unix", server.SocketName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(conn)
	var resp protocol.Response
	if err := decoder.Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
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
		
		if outputFile == "" {
			// Check if stdout is a terminal
			fi, _ := os.Stdout.Stat()
			if (fi.Mode() & os.ModeCharDevice) != 0 {
				// Output is a TTY, use default filename
				outputFile = id + ".png"
			}
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Screenshot saved to %s\n", outputFile)
		} else {
			// Piped to another command, write raw binary
			os.Stdout.Write(data)
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

func History(args []string) {
	id := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--id" && i+1 < len(args) {
			id = args[i+1]
			i++
		}
	}

	if id == "" {
		id = DefaultID
	}

	req := protocol.Request{
		Op: protocol.OpHistory,
		ID: id,
	}
	
	conn, err := net.Dial("unix", server.SocketName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(conn)
	var resp protocol.Response
	if err := decoder.Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.Status != "ok" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
		os.Exit(1)
	}

	// Output is JSON array string. Let's pretty print it or just print it.
	// For CLI usability, maybe just print each item on a new line?
	// Or raw JSON.
	// Let's try to decode and print lines.
	var history []string
	if err := json.Unmarshal([]byte(resp.Data), &history); err != nil {
		// Fallback to raw
		fmt.Println(resp.Data)
		return
	}

	for i, entry := range history {
		fmt.Printf("%d: %q\n", i, entry)
	}
}

func Wait(args []string) {
	id := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--id" && i+1 < len(args) {
			id = args[i+1]
			i++
		}
	}

	if id == "" {
		id = DefaultID
	}

	req := protocol.Request{
		Op: protocol.OpWait,
		ID: id,
	}
	sendRequest(req)
}
