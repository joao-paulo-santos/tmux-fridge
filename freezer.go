package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var filteredCommands = map[string]bool{
	"bash": true, "-bash": true,
	"zsh": true, "-zsh": true,
	"sh": true, "-sh": true,
	"fish": true, "-fish": true,
	"python": true, "python3": true,
	"ruby": true, "node": true,
}

var interpreters = map[string]bool{
	"node": true, "python": true, "python3": true, "ruby": true,
}

func isFilteredCommand(cmd string) bool {
	if cmd == "" {
		return true
	}
	return filteredCommands[cmd]
}

func resolveCommand(panePid int, currentCmd string) string {
	if !interpreters[currentCmd] {
		return currentCmd
	}
	children, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", panePid, panePid))
	if err != nil {
		return currentCmd
	}
	for _, pidStr := range strings.Fields(string(children)) {
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%s/cmdline", pidStr))
		if err != nil {
			continue
		}
		parts := strings.Split(string(cmdline), "\x00")
		if len(parts) >= 2 {
			return filepath.Base(parts[1])
		}
	}
	return currentCmd
}

func captureSession(sessionName string) (*Session, error) {
	windows, err := queryWindows(sessionName)
	if err != nil {
		return nil, err
	}

	cfg := LoadConfig()
	session := &Session{SessionName: sessionName}

	for _, w := range windows {
		panes, err := queryPanesForWindow(sessionName, w.index)
		if err != nil {
			return nil, err
		}

		var sessionPanes []Pane
		for _, p := range panes {
			pane := Pane{
				StartDirectory: p.path,
				Focus:          p.active,
			}
			cmd := p.command
			if cfg.ResolveInterpreters {
				cmd = resolveCommand(p.pid, cmd)
			}
			if !isFilteredCommand(cmd) {
				if mapped, ok := cfg.CommandMap[cmd]; ok {
					pane.ShellCommand = mapped
				} else {
					pane.ShellCommand = cmd
				}
			}
			sessionPanes = append(sessionPanes, pane)
		}

		startDir := ""
		if len(sessionPanes) > 0 {
			startDir = sessionPanes[0].StartDirectory
		}

		session.Windows = append(session.Windows, Window{
			WindowName:     w.name,
			Layout:         w.layout,
			StartDirectory: startDir,
			Panes:          sessionPanes,
		})
	}

	return session, nil
}

type tmuxWindow struct {
	name   string
	layout string
	active bool
	index  int
}

type tmuxPane struct {
	path    string
	command string
	active  bool
	pid     int
}

func queryPanesForWindow(session string, winIdx int) ([]tmuxPane, error) {
	target := fmt.Sprintf("%s:%d", session, winIdx)
	cmd := exec.Command("tmux", "list-panes", "-t", target,
		"-F", "#{pane_current_path}\t#{pane_current_command}\t#{pane_active}\t#{pane_pid}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying panes for %s: %w", target, err)
	}

	var panes []tmuxPane
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(parts[3])
		panes = append(panes, tmuxPane{
			path:    parts[0],
			command: parts[1],
			active:  parts[2] == "1",
			pid:     pid,
		})
	}
	return panes, nil
}

func queryWindows(session string) ([]tmuxWindow, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session,
		"-F", "#{window_name}\t#{window_layout}\t#{window_active}\t#{window_index}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying windows: %w", err)
	}

	var windows []tmuxWindow
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		winIdx, _ := strconv.Atoi(parts[3])
		windows = append(windows, tmuxWindow{
			name:   parts[0],
			layout: parts[1],
			active: parts[2] == "1",
			index:  winIdx,
		})
	}
	return windows, nil
}

func Freeze(sessionName string) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	session, err := captureSession(sessionName)
	if err != nil {
		return err
	}

	frozenPath := FrozenPath(sessionName)
	coldPath := ColdPath(sessionName)

	if err := WriteSession(frozenPath, session); err != nil {
		return fmt.Errorf("writing frozen session: %w", err)
	}

	if err := CopyFile(frozenPath, coldPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to copy to cold storage: %v\n", err)
	}

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("killing session: %w", err)
	}

	return nil
}

func Snapshot(sessionName string) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	session, err := captureSession(sessionName)
	if err != nil {
		return err
	}

	return WriteSession(ColdPath(sessionName), session)
}
