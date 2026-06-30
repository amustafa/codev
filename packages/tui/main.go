package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cluesmith-ai/codev-tui/internal/api"
	"github.com/cluesmith-ai/codev-tui/internal/attach"
	"github.com/cluesmith-ai/codev-tui/internal/tui"
	"github.com/cluesmith-ai/codev-tui/internal/zellij"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "attach" {
		runAttach()
		return
	}

	towerURL := flag.String("tower-url", "http://localhost:4100", "Tower server URL")
	workspace := flag.String("workspace", "", "Workspace path (skips selector if set)")
	noZellij := flag.Bool("no-zellij", false, "Run without Zellij even if available")
	flag.Parse()

	if !*noZellij && !zellij.IsInsideZellij() {
		launchWithZellij()
		return
	}

	client := api.NewClient(*towerURL)
	app := tui.NewApp(client, *workspace)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAttach() {
	attachFlags := flag.NewFlagSet("attach", flag.ExitOnError)
	towerURL := attachFlags.String("tower-url", "http://localhost:4100", "Tower server URL")
	workspace := attachFlags.String("workspace", "", "Workspace path for scoped WebSocket URL")
	attachFlags.Parse(os.Args[2:])

	args := attachFlags.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: codev-tui attach [--tower-url URL] [--workspace PATH] <terminalId>")
		os.Exit(1)
	}

	terminalID := args[0]

	if !zellij.IsInsideZellij() {
		launchAttachInZellij(os.Args[1:]...)
		return
	}

	if err := attach.Run(*towerURL, terminalID, *workspace); err != nil {
		fmt.Fprintf(os.Stderr, "attach error: %v\n", err)
		os.Exit(1)
	}
}

func launchAttachInZellij(attachArgs ...string) {
	if _, err := exec.LookPath("zellij"); err != nil {
		fmt.Fprintln(os.Stderr, "Zellij not found. Install it or run inside an existing Zellij session.")
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "codev-tui"
	}

	zellijArgs := []string{"run", "--"}
	zellijArgs = append(zellijArgs, exePath)
	zellijArgs = append(zellijArgs, attachArgs...)

	cmd := exec.Command("zellij", zellijArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Zellij exited: %v\n", err)
		os.Exit(1)
	}
}

func launchWithZellij() {
	_, err := exec.LookPath("zellij")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Zellij not found. Install it or run with --no-zellij.")
		fmt.Fprintln(os.Stderr, "  Install: cargo install --locked zellij")
		fmt.Fprintln(os.Stderr, "  Or:      brew install zellij")
		os.Exit(1)
	}

	layoutPath := findLayout()

	args := []string{"--layout", layoutPath, "--session", "codev"}
	cmd := exec.Command("zellij", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Zellij exited: %v\n", err)
		os.Exit(1)
	}
}

func findLayout() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "layouts", "codev.kdl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, "layouts", "codev.kdl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	candidate := filepath.Join(cwd, "packages", "tui", "layouts", "codev.kdl")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	fmt.Fprintln(os.Stderr, "Could not find layouts/codev.kdl. Run from the packages/tui/ directory.")
	os.Exit(1)
	return ""
}
