# Specter

Specter is a test harness for running and interacting with terminal applications (TUIs). It is designed to allow AI agents or automated scripts to test terminal interfaces by spawning processes, simulating input, and inspecting the terminal state.

> **Quick Start**: Run `specter quickstart` for a concise guide optimized for LLM coding agents.

## Architecture

Specter operates as a client-server architecture:

1.  **Server**: A background process that manages PTYs (Pseudo-Terminals) and tracks the state of running terminal applications. It uses a virtual terminal emulator (binding to `libvterm`) to maintain an in-memory representation of the screen.
2.  **Client**: A CLI tool that connects to the server to perform actions.
3.  **Communication**: The client and server communicate via a Unix domain socket (`.specter.sock`) located in the current working directory.

## Tech Stack

*   **Language**: Go
*   **PTY Management**: [creack/pty](https://github.com/creack/pty)
*   **Terminal Emulation**: Bindings to `libvterm` for screen state tracking.

## Usage

### 1. Start the Server

Start the specter server in the directory where you want to manage sessions.

```bash
specter server &
```

This creates a hidden socket (`.specter.sock`) in the current directory.

### 2. Spawn a Terminal Session

Start a new terminal session.

```bash
specter spawn -- bash
# Or with a specific ID: specter spawn --id myapp -- vim file.txt
```

All commands default to `--id "default"` for simple single-session usage.

### 3. Send Input

Send key presses or text to the session.

```bash
specter type "ls -la\n"          # Type command and press Enter
specter type "Hello World"       # Type text without Enter
specter type "\t"                # Press Tab (for autocomplete)
specter type "\x03"              # Send Ctrl+C (interrupt)
```

#### Common Escape Sequences

| Sequence | Description |
|----------|-------------|
| `\n` | Enter/newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Literal backslash |
| `\x03` | Ctrl+C (interrupt) |
| `\x04` | Ctrl+D (EOF) |
| `\x1b` | Escape key |

### 4. Capture Screen

Capture the current state of the terminal screen.

```bash
specter capture                  # Get text content
specter capture --format png     # Get screenshot image
```

Use `--out <file>` to specify a filename for PNG output.

### 5. Wait for Exit

Wait for a process to exit.

```bash
specter wait                     # Blocks until process exits
```

### 6. View History

View the input history sent to a session.

```bash
specter history
```

## Tips for Testing TUIs

- After sending input, wait briefly then capture to see the result
- Use capture frequently to verify the application state
- For interactive programs (vim, htop), use escape sequences for navigation
- The `--id` flag lets you manage multiple sessions simultaneously

## Example: Testing a CLI Tool

```bash
specter spawn -- ./my-cli-tool
specter type "help\n"
specter capture                   # Verify help output appeared
specter type "quit\n"
specter wait                      # Wait for clean exit
```

## Development

### Prerequisites

*   Go 1.23+
*   `libvterm`

### Build

```bash
go build -o specter cmd/specter/main.go
```
