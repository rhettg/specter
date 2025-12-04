# Specter

Specter is a test harness for running and interacting with terminal applications (TUIs). It is designed to allow AI agents or automated scripts to test terminal interfaces by spawning processes, simulating input, and inspecting the terminal state.

> **Quick Start**: Run `specter quickstart` for a concise guide optimized for LLM coding agents.

![Demo](https://raw.githubusercontent.com/rhettg/specter/master/media/demo.gif)

## Architecture

Specter operates as a self-contained process:

1. **Spawn**: The `spawn` command starts a background process that manages a PTY (Pseudo-Terminal) and tracks the state of the running terminal application using a virtual terminal emulator (binding to `libvterm`).
2. **Commands**: CLI commands connect to the running process to perform actions.
3. **Communication**: Commands communicate via a Unix domain socket (`.specter.sock`) located in the current working directory.

## Tech Stack

* **Language**: Go
* **PTY Management**: [creack/pty](https://github.com/creack/pty)
* **Terminal Emulation**: Bindings to `libvterm` for screen state tracking.

## Usage

### 1. Spawn a Shell Session

Start a new shell session:

```bash
specter spawn                    # Starts $SHELL (or /bin/sh)
specter spawn -- vim file.txt    # Or run a specific command directly
```

This starts the process and creates a hidden socket (`.specter.sock`) in the current directory.

### 2. Send Input

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

### 3. Capture Screen

Capture the current state of the terminal screen.

```bash
specter capture                  # Get text content
specter capture --format png     # Get screenshot image
```

Use `--out <file>` to specify a filename for PNG output.

### 4. Wait for Exit

Wait for a process to exit.

```bash
specter wait                     # Blocks until process exits
```

### 5. View History

View the input history sent to the session.

```bash
specter history
```

### 6. Terminate Session

Kill the specter session and clean up.

```bash
specter kill
```

## Tips for Testing TUIs

* After sending input, wait briefly then capture to see the result
* Use capture frequently to verify the application state
* For interactive programs (vim, htop), use escape sequences for navigation

## Example: Running Commands in a Shell

```bash
specter spawn
specter type "echo hello\n"
specter capture                   # Verify output
specter type "./my-cli-tool\n"    # Run a program
specter capture
specter kill                      # Done
```

## Development

### Prerequisites

* Go 1.23+
* `libvterm`

### Build

```bash
go build -o specter cmd/specter/main.go
```
