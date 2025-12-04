package tests

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"specter/internal/protocol"
	"specter/internal/server"
	"strings"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	os.Remove(server.SocketName)

	go func() {
		if err := server.Start([]string{"/bin/sh", "-c", "echo hello world; sleep 2"}); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	connected := false
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for server socket")
		case <-ticker.C:
			if _, err := os.Stat(server.SocketName); err == nil {
				connected = true
			}
		}
		if connected {
			break
		}
	}

	send := func(req protocol.Request) protocol.Response {
		conn, err := net.Dial("unix", server.SocketName)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		json.NewEncoder(conn).Encode(req)
		var resp protocol.Response
		json.NewDecoder(conn).Decode(&resp)
		return resp
	}

	time.Sleep(500 * time.Millisecond)

	capResp := send(protocol.Request{
		Op: protocol.OpCapture,
	})

	if capResp.Status != "ok" {
		t.Fatalf("Capture failed: %s", capResp.Message)
	}

	t.Logf("Capture output:\n%s", capResp.Data)

	if !strings.Contains(capResp.Data, "hello world") {
		t.Error("Did not find 'hello world' in capture output")
	}

	killResp := send(protocol.Request{
		Op: protocol.OpKill,
	})

	if killResp.Status != "ok" {
		t.Fatalf("Kill failed: %s", killResp.Message)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestInteractive(t *testing.T) {
	os.Remove(server.SocketName)

	go func() {
		if err := server.Start([]string{"/bin/cat"}); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	connected := false
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for server socket")
		case <-ticker.C:
			if _, err := os.Stat(server.SocketName); err == nil {
				connected = true
			}
		}
		if connected {
			break
		}
	}

	send := func(req protocol.Request) protocol.Response {
		conn, err := net.Dial("unix", server.SocketName)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		json.NewEncoder(conn).Encode(req)
		var resp protocol.Response
		json.NewDecoder(conn).Decode(&resp)
		return resp
	}

	time.Sleep(100 * time.Millisecond)

	typeResp := send(protocol.Request{
		Op:      protocol.OpType,
		Payload: []string{"foo bar\n"},
	})

	if typeResp.Status != "ok" {
		t.Fatalf("Type failed: %s", typeResp.Message)
	}

	time.Sleep(100 * time.Millisecond)

	capResp := send(protocol.Request{
		Op: protocol.OpCapture,
	})

	if capResp.Status != "ok" {
		t.Fatalf("Capture failed: %s", capResp.Message)
	}

	t.Logf("Capture output:\n%s", capResp.Data)

	if !strings.Contains(capResp.Data, "foo bar") {
		t.Error("Did not find 'foo bar' in capture output for cat session")
	}

	pngResp := send(protocol.Request{
		Op:      protocol.OpCapture,
		Options: map[string]string{"format": "png"},
	})

	if pngResp.Status != "ok" {
		t.Fatalf("Capture PNG failed: %s", pngResp.Message)
	}

	pngData, err := base64.StdEncoding.DecodeString(pngResp.Data)
	if err != nil {
		t.Fatalf("Failed to decode PNG data: %v", err)
	}

	expectedMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(pngData) < 8 {
		t.Fatalf("PNG data too short")
	}

	for i := range expectedMagic {
		if pngData[i] != expectedMagic[i] {
			t.Errorf("Invalid PNG magic byte at %d: got %x, want %x", i, pngData[i], expectedMagic[i])
		}
	}

	os.WriteFile("test_output.png", pngData, 0644)

	send(protocol.Request{Op: protocol.OpKill})
	time.Sleep(100 * time.Millisecond)
}
