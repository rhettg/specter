# Specter

Specter is a test harness for running and interacting with terminal applications (TUIs). It is designed to allow AI agents or automated scripts to test terminal interfaces by spawning processes, simulating input, and inspecting the terminal state.

## Architecture

Specter operates as a client-server architecture:

1.  **Server**: A background process that manages PTYs (Pseudo-Terminals) and tracks the state of running terminal applications. It uses a virtual terminal emulator (likely binding to `libvterm`) to maintain an in-memory representation of the screen.
2.  **Client**: A CLI tool that connects to the server to perform actions.
3.  **Communication**: The client and server communicate via a Unix domain socket (pipe) located in the current working directory.

## Tech Stack

*   **Language**: Go
*   **PTY Management**: [creack/pty](https://github.com/creack/pty)
*   **Terminal Emulation**: Bindings to `libvterm` (or similar) for screen state tracking.

## Usage

### 1. Start the Server

Start the specter server in the directory where you want to manage sessions.

```bash
specter server
```

This creates a hidden socket (`.specter.sock`) in the current directory.

### 2. Run a Command

Start a new terminal session.

```bash
specter spawn --id "mysession" -- bash
```

### 3. Interact

Send key presses or text to the session.

```bash
specter type --id "mysession" "ls -la\n"
```

### 4. Inspect

Capture the current state of the terminal screen.

```bash
specter capture --id "mysession"
```

This will output the text content of the terminal as it appears to a user.

You can also capture a screenshot in PNG format:

```bash
specter capture --id "mysession" --format png
```

This saves to `mysession.png` by default. Use `--out <file>` to specify a different filename, or pipe the output to another command (binary data is written to stdout when not a TTY).

### 5. History

View the input history sent to a session.

```bash
specter history --id "mysession"
```

## Development

### Prerequisites

*   Go 1.23+
*   `libvterm` (if using C bindings)

### Build

```bash
go build -o specter cmd/specter/main.go
```