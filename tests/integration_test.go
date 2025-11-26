package tests

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"specter/internal/protocol"
	"specter/internal/server"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Wait for socket
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
	
	// Helper to send request
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

	// Spawn
	// We use 'sh' to sleep a bit so we can capture something, or echo.
	// 'echo hello' finishes quickly, so the pty might close before we capture?
	// But the server keeps the session object.
	// However, if the process exits, the pty read loop exits.
	// But the vterm state should remain in the session object.
	spawnResp := send(protocol.Request{
		Op: protocol.OpSpawn,
		ID: "test1",
		Payload: []string{"/bin/sh", "-c", "echo hello world; sleep 1"},
	})
	
	if spawnResp.Status != "ok" {
		t.Fatalf("Spawn failed: %s", spawnResp.Message)
	}

	// Wait a moment for output to propagate to vterm
	time.Sleep(500 * time.Millisecond)

	// Capture
	capResp := send(protocol.Request{
		Op: protocol.OpCapture,
		ID: "test1",
	})

	if capResp.Status != "ok" {
		t.Fatalf("Capture failed: %s", capResp.Message)
	}

	t.Logf("Capture output:\n%s", capResp.Data)

	// Check for "hello world"
	found := false
	if len(capResp.Data) > 0 {
		// Simple check
		for i := 0; i < len(capResp.Data)-10; i++ {
			if capResp.Data[i:i+11] == "hello world" {
				found = true
				break
			}
		}
	}
	
	if !found {
		t.Error("Did not find 'hello world' in capture output")
	}

	// Test Interactive (Type)
	spawnResp2 := send(protocol.Request{
		Op: protocol.OpSpawn,
		ID: "test2",
		Payload: []string{"/bin/cat"}, // cat echoes input
	})

	if spawnResp2.Status != "ok" {
		t.Fatalf("Spawn cat failed: %s", spawnResp2.Message)
	}

	time.Sleep(100 * time.Millisecond)

	// Send input
	typeResp := send(protocol.Request{
		Op: protocol.OpType,
		ID: "test2",
		Payload: []string{"foo bar\n"},
	})

	if typeResp.Status != "ok" {
		t.Fatalf("Type failed: %s", typeResp.Message)
	}

	time.Sleep(100 * time.Millisecond)

	// Capture
	capResp2 := send(protocol.Request{
		Op: protocol.OpCapture,
		ID: "test2",
	})

	if capResp2.Status != "ok" {
		t.Fatalf("Capture 2 failed: %s", capResp2.Message)
	}

	t.Logf("Capture 2 output:\n%s", capResp2.Data)

	// Check for "foo bar"
	// Note: cat echoes the input, so we should see it.
	// Depending on terminal mode, it might echo the input chars as they are typed, 
	// and then the output from cat.
	found2 := false
	// Simple search
	for i := 0; i < len(capResp2.Data)-6; i++ {
		if capResp2.Data[i:i+7] == "foo bar" {
			found2 = true
			break
		}
	}
	
	if !found2 {
		t.Error("Did not find 'foo bar' in capture output for cat session")
	}

	// Test Capture PNG
	pngResp := send(protocol.Request{
		Op: protocol.OpCapture,
		ID: "test2",
		Options: map[string]string{"format": "png"},
	})

	if pngResp.Status != "ok" {
		t.Fatalf("Capture PNG failed: %s", pngResp.Message)
	}

	// Decode base64
	pngData, err := base64.StdEncoding.DecodeString(pngResp.Data)
	if err != nil {
		t.Fatalf("Failed to decode PNG data: %v", err)
	}

	// Check Magic Bytes (0x89 0x50 0x4E 0x47 0x0D 0x0A 0x1A 0x0A)
	expectedMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(pngData) < 8 {
		t.Fatalf("PNG data too short")
	}
	
	for i := range expectedMagic {
		if pngData[i] != expectedMagic[i] {
			t.Errorf("Invalid PNG magic byte at %d: got %x, want %x", i, pngData[i], expectedMagic[i])
		}
	}

	// Save it for inspection (optional)
	os.WriteFile("test_output.png", pngData, 0644)
}
