package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func configBase() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "tmux", "tmux-workspaces")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tmux", "tmux-workspaces")
}

type Config struct {
	ResolveInterpreters bool              `yaml:"resolve_interpreters"`
	CommandMap          map[string]string `yaml:"command_map"`
}

func ConfigPath() string {
	return filepath.Join(configBase(), "config.yaml")
}

func LoadConfig() *Config {
	cfg := &Config{CommandMap: make(map[string]string)}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	yaml.Unmarshal(data, cfg)
	if cfg.CommandMap == nil {
		cfg.CommandMap = make(map[string]string)
	}
	return cfg
}

func FrozenDir() string    { return filepath.Join(configBase(), "frozen") }
func ColdDir() string      { return filepath.Join(configBase(), "cold-storage") }
func FrozenPath(s string) string { return filepath.Join(FrozenDir(), s+".yaml") }
func ColdPath(s string) string   { return filepath.Join(ColdDir(), s+".yaml") }

func EnsureDirs() error {
	if err := os.MkdirAll(FrozenDir(), 0755); err != nil {
		return fmt.Errorf("creating frozen dir: %w", err)
	}
	if err := os.MkdirAll(ColdDir(), 0755); err != nil {
		return fmt.Errorf("creating cold-storage dir: %w", err)
	}
	return nil
}

func WriteSession(path string, s *Session) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func ReadSession(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	var s Session
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	return &s, nil
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func listYamlNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

func activeTmuxSessions() []string {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions
}

func ListFrozen() ([]string, error) {
	return listYamlNames(FrozenDir())
}

func ListColdAll() ([]string, error) {
	return listYamlNames(ColdDir())
}

func ListCold() ([]string, error) {
	cold, err := listYamlNames(ColdDir())
	if err != nil {
		return nil, err
	}

	exclude := make(map[string]bool)
	for _, s := range activeTmuxSessions() {
		exclude[s] = true
	}
	frozen, _ := listYamlNames(FrozenDir())
	for _, s := range frozen {
		exclude[s] = true
	}

	var result []string
	for _, s := range cold {
		if !exclude[s] {
			result = append(result, s)
		}
	}
	return result, nil
}

func Clean(session string) error {
	path := ColdPath(session)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing cold storage for %s: %w", session, err)
	}
	return nil
}

func CleanAll() error {
	entries, err := os.ReadDir(ColdDir())
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			os.Remove(filepath.Join(ColdDir(), e.Name()))
		}
	}
	return nil
}
