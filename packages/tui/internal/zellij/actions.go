package zellij

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func IsInsideZellij() bool {
	return os.Getenv("ZELLIJ") != ""
}

func NewPane(command string, args ...string) error {
	if !IsInsideZellij() {
		return fmt.Errorf("not inside Zellij")
	}
	cmdArgs := []string{"action", "new-pane", "--direction", "right", "--"}
	cmdArgs = append(cmdArgs, command)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("zellij", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// AttachToTerminal opens a Zellij pane running `codev-tui attach <terminalId>`
// which bridges Tower's WebSocket binary protocol to stdin/stdout.
func AttachToTerminal(towerURL, terminalID, workspace string) error {
	if terminalID == "" {
		return fmt.Errorf("no terminal ID")
	}

	if !IsInsideZellij() {
		fmt.Printf("Terminal ID: %s\n", terminalID)
		fmt.Printf("Run: codev-tui attach --tower-url %s --workspace %q %s\n",
			towerURL, workspace, terminalID)
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "codev-tui"
	}

	args := []string{
		"action", "new-pane", "--direction", "right", "--",
		exePath, "attach",
		"--tower-url", towerURL,
	}
	if workspace != "" {
		args = append(args, "--workspace", workspace)
	}
	args = append(args, terminalID)

	cmd := exec.Command("zellij", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// OpenWorkspaceTab creates a named Zellij tab for a workspace with a split
// layout: TUI on the left scoped to the workspace, architect terminal on
// the right attached to the live PTY session.
func OpenWorkspaceTab(name, towerURL, workspacePath, architectTerminalID string) error {
	if !IsInsideZellij() {
		return fmt.Errorf("not inside Zellij")
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "codev-tui"
	}

	layoutPath, err := writeWorkspaceLayout(exePath, towerURL, workspacePath, architectTerminalID)
	if err != nil {
		return fmt.Errorf("failed to write layout: %w", err)
	}

	args := []string{"action", "new-tab", "--layout", layoutPath, "--name", name}
	cmd := exec.Command("zellij", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeWorkspaceLayout(exePath, towerURL, workspacePath, architectTerminalID string) (string, error) {
	var rightPane string
	if architectTerminalID != "" {
		rightPane = fmt.Sprintf(`        pane command=%q size="60%%" {
            args "attach" "--tower-url" %q "--workspace" %q %q
            name "Architect"
        }`, exePath, towerURL, workspacePath, architectTerminalID)
	} else {
		rightPane = `        pane size="60%" {
            name "Terminal"
        }`
	}

	layout := fmt.Sprintf(`layout {
    pane split_direction="vertical" {
        pane command=%q size="40%%" {
            args "--no-zellij" "--tower-url" %q "--workspace" %q
            name "Codev"
        }
%s
    }
}
`, exePath, towerURL, workspacePath, rightPane)

	tmpDir := os.TempDir()
	f, err := os.CreateTemp(tmpDir, "codev-ws-*.kdl")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(layout); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

// OpenAllWorkspaceTabs opens a named Zellij tab for each workspace.
// Takes a slice of workspace info with name, path, and optional architect terminal ID.
func OpenAllWorkspaceTabs(towerURL string, workspaces []WorkspaceTabInfo) error {
	for _, ws := range workspaces {
		if err := OpenWorkspaceTab(ws.Name, towerURL, ws.Path, ws.ArchitectTerminalID); err != nil {
			return fmt.Errorf("failed to open tab for %s: %w", ws.Name, err)
		}
	}
	return nil
}

type WorkspaceTabInfo struct {
	Name                string
	Path                string
	ArchitectTerminalID string
}

func OpenBuilderTerminal(worktreePath string) error {
	if !IsInsideZellij() {
		fmt.Printf("Builder worktree: %s\n", worktreePath)
		return nil
	}
	shellCmd := fmt.Sprintf("cd %q && exec ${SHELL:-bash}", worktreePath)
	return NewPane("bash", "-c", shellCmd)
}

func ClosePane() error {
	if !IsInsideZellij() {
		return nil
	}
	return exec.Command("zellij", "action", "close-pane").Run()
}

func FocusNextPane() error {
	if !IsInsideZellij() {
		return nil
	}
	return exec.Command("zellij", "action", "focus-next-pane").Run()
}

// ExePath returns the path to the current executable, falling back to
// searching PATH for "codev-tui".
func ExePath() string {
	if p, err := os.Executable(); err == nil {
		if abs, err := filepath.EvalSymlinks(p); err == nil {
			return abs
		}
		return p
	}
	return "codev-tui"
}
