package main

import (
	"fmt"
	"os"
	"specter/internal/client"
	"specter/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	case "spawn":
		// specter spawn --id <id> -- <cmd>
		client.Spawn(os.Args[2:])
	case "type":
		// specter type --id <id> "text"
		client.Type(os.Args[2:])
	case "capture":
		// specter capture --id <id>
		client.Capture(os.Args[2:])
	case "history":
		// specter history --id <id>
		client.History(os.Args[2:])
	case "wait":
		// specter wait --id <id>
		client.Wait(os.Args[2:])
	case "quickstart":
		printQuickstart()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: specter <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  server      Start the specter server")
	fmt.Println("  spawn       Start a new process (usage: specter spawn [--id <id>] -- <cmd>)")
	fmt.Println("  type        Send input to a session (usage: specter type [--id <id>] <text>)")
	fmt.Println("  capture     Capture screen content (usage: specter capture [--id <id>])")
	fmt.Println("  history     Show input history (usage: specter history [--id <id>])")
	fmt.Println("  wait        Wait for process to exit and return exit code (usage: specter wait [--id <id>])")
	fmt.Println("  quickstart  Show quickstart guide for LLM coding agents")
}

func printQuickstart() {
	fmt.Print(`Specter Quickstart Guide for LLM Coding Agents
===============================================

Specter is a terminal test harness that lets you spawn, interact with, and
inspect terminal applications (TUIs) programmatically.

## Basic Workflow

1. Start the server (run once, keeps running in background):
   specter server &

2. Spawn a terminal session:
   specter spawn -- bash
   # Or with a specific ID: specter spawn --id myapp -- vim file.txt

3. Send input (text and keypresses):
   specter type "ls -la\n"          # Type command and press Enter
   specter type "Hello World"       # Type text without Enter
   specter type "\t"                # Press Tab (for autocomplete)
   specter type "\x03"              # Send Ctrl+C (ASCII 3)

4. Capture the screen to see what's displayed:
   specter capture                  # Get text content
   specter capture --format png     # Get screenshot image

5. Wait for a process to exit:
   specter wait                     # Blocks until process exits

## Common Escape Sequences for type

  \n    - Enter/newline
  \t    - Tab
  \r    - Carriage return
  \\    - Literal backslash
  \x03  - Ctrl+C (interrupt)
  \x04  - Ctrl+D (EOF)
  \x1b  - Escape key

## Tips for Testing TUIs

- After sending input, wait briefly then capture to see the result
- Use capture frequently to verify the application state
- For interactive programs (vim, htop), use escape sequences for navigation
- The --id flag lets you manage multiple sessions simultaneously
- All commands default to --id "default" for simple single-session usage

## Example: Testing a CLI Tool

  specter spawn -- ./my-cli-tool
  specter type "help\n"
  specter capture                   # Verify help output appeared
  specter type "quit\n"
  specter wait                      # Wait for clean exit
`)
}
