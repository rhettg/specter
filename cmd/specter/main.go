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
	case "_server":
		var cmd []string
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--" {
				cmd = os.Args[i+1:]
				break
			}
		}
		if err := server.Start(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	case "spawn":
		client.Spawn(os.Args[2:])
	case "type":
		client.Type(os.Args[2:])
	case "capture":
		client.Capture(os.Args[2:])
	case "history":
		client.History()
	case "wait":
		client.Wait()
	case "kill":
		client.Kill()
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
	fmt.Println("  spawn       Start a new session (usage: specter spawn [-- <cmd>])")
	fmt.Println("  type        Send input to session (usage: specter type <text>)")
	fmt.Println("  capture     Capture screen content (usage: specter capture [--format png] [--out file])")
	fmt.Println("  history     Show input history")
	fmt.Println("  wait        Wait for process to exit and return exit code")
	fmt.Println("  kill        Terminate the specter session")
	fmt.Println("  quickstart  Show quickstart guide for LLM coding agents")
}

func printQuickstart() {
	fmt.Print(`Specter Quickstart Guide
========================

Specter is a terminal test harness that lets you spawn, interact with, and
inspect terminal applications (TUIs) programmatically.

## Basic Workflow

1. Spawn a shell session:
   specter spawn                    # Starts $SHELL (or /bin/sh)
   specter spawn -- vim file.txt    # Or run a specific command

2. Send input (text and keypresses):
   specter type "ls -la\n"          # Type command and press Enter
   specter type "Hello World"       # Type text without Enter
   specter type "\t"                # Press Tab (for autocomplete)
   specter type "\x03"              # Send Ctrl+C (ASCII 3)

3. Capture the screen to see what's displayed:
   specter capture                  # Get text content
   specter capture --format png     # Get screenshot image

4. When done, terminate the session:
   specter kill

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

## Example: Running commands in a shell

  specter spawn
  specter type "echo hello\n"
  specter capture                   # Verify output
  specter type "./my-cli-tool\n"    # Run a program
  specter capture
  specter kill                      # Done
`)
}
