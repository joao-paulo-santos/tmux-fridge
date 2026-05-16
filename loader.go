package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func detectTerminal() string {
	if t := os.Getenv("TERMINAL"); t != "" {
		return t
	}
	for _, candidate := range []string{"kitty", "alacritty", "gnome-terminal", "foot"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return "xterm"
}

func buildAttachArgs(terminal, sessionName string) []string {
	cmd := []string{"tmux", "attach-session", "-t", sessionName}
	switch terminal {
	case "alacritty", "xterm", "xterm-256color":
		return append([]string{"-e"}, cmd...)
	case "gnome-terminal":
		return append([]string{"--"}, cmd...)
	default:
		return cmd
	}
}

func attachInTerminal(sessionName string) error {
	terminal := detectTerminal()
	args := buildAttachArgs(terminal, sessionName)
	cmd := exec.Command(terminal, args...)
	devnull, _ := os.Open(os.DevNull)
	cmd.Stdin = nil
	cmd.Stdout = devnull
	cmd.Stderr = devnull
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching terminal: %w", err)
	}
	return nil
}

func loadSession(s *Session) error {
	if len(s.Windows) == 0 {
		return fmt.Errorf("no windows in session")
	}

	firstWin := s.Windows[0]

	cmd := exec.Command("tmux", "new-session", "-d", "-s", s.SessionName,
		"-n", firstWin.WindowName, "-c", firstWin.StartDirectory)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	for i, w := range s.Windows {
		if i == 0 {
			continue
		}
		cmd := exec.Command("tmux", "new-window", "-t", s.SessionName,
			"-n", w.WindowName, "-c", w.StartDirectory)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("creating window %s: %w", w.WindowName, err)
		}
	}

	winIndices, err := getWindowIndices(s.SessionName)
	if err != nil {
		return fmt.Errorf("querying window indices: %w", err)
	}

	for i, w := range s.Windows {
		if i >= len(winIndices) {
			break
		}
		target := fmt.Sprintf("%s:%d", s.SessionName, winIndices[i])

		for paneIdx := 1; paneIdx < len(w.Panes); paneIdx++ {
			p := w.Panes[paneIdx]
			splitCmd := exec.Command("tmux", "split-window", "-t", target, "-c", p.StartDirectory)
			if err := splitCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to split pane in window %s: %v\n", w.WindowName, err)
			}
		}

		if len(w.Panes) > 1 {
			exec.Command("tmux", "select-layout", "-t", target, w.Layout).Run()
		}
	}

	time.Sleep(100 * time.Millisecond)

	for i, w := range s.Windows {
		if i >= len(winIndices) {
			break
		}

		paneIndices, err := getPaneIndices(s.SessionName, winIndices[i])
		if err != nil {
			continue
		}

		for j, p := range w.Panes {
			if p.ShellCommand == "" || j >= len(paneIndices) {
				continue
			}
			paneTarget := fmt.Sprintf("%s:%d.%d", s.SessionName, winIndices[i], paneIndices[j])
			sendCmd := fmt.Sprintf("sleep 0.3 && cd '%s' && %s", p.StartDirectory, p.ShellCommand)
			exec.Command("tmux", "send-keys", "-t", paneTarget, sendCmd, "Enter").Run()
		}
	}

	return nil
}

func getWindowIndices(sessionName string) ([]int, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		idx, _ := strconv.Atoi(line)
		indices = append(indices, idx)
	}
	return indices, nil
}

func getPaneIndices(sessionName string, winIdx int) ([]int, error) {
	target := fmt.Sprintf("%s:%d", sessionName, winIdx)
	cmd := exec.Command("tmux", "list-panes", "-t", target, "-F", "#{pane_index}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		idx, _ := strconv.Atoi(line)
		indices = append(indices, idx)
	}
	return indices, nil
}

func Unfreeze(sessionName string) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	frozenPath := FrozenPath(sessionName)
	coldPath := ColdPath(sessionName)

	session, err := ReadSession(frozenPath)
	if err != nil {
		return err
	}

	if err := loadSession(session); err != nil {
		return err
	}

	CopyFile(frozenPath, coldPath)
	os.Remove(frozenPath)

	return attachInTerminal(sessionName)
}

func Recover(sessionName string) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	session, err := ReadSession(ColdPath(sessionName))
	if err != nil {
		return err
	}

	if err := loadSession(session); err != nil {
		return err
	}

	return attachInTerminal(sessionName)
}

func Attach(sessionName string) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	if err := Snapshot(sessionName); err != nil {
		fmt.Fprintf(os.Stderr, "warning: snapshot failed: %v\n", err)
	}

	return attachInTerminal(sessionName)
}
