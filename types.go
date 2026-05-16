package main

type Pane struct {
	StartDirectory string `yaml:"start_directory"`
	ShellCommand   string `yaml:"shell_command,omitempty"`
	Focus          bool   `yaml:"focus,omitempty"`
}

type Window struct {
	WindowName     string `yaml:"window_name"`
	Layout         string `yaml:"layout"`
	StartDirectory string `yaml:"start_directory"`
	Panes          []Pane `yaml:"panes"`
}

type Session struct {
	SessionName string   `yaml:"session_name"`
	Windows     []Window `yaml:"windows"`
}
