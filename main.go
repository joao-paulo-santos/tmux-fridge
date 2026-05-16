package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: tmux-fridge <command> [args]

Commands:
  freeze <session>     Freeze a session (snapshot + kill)
  unfreeze <session>   Unfreeze a session (restore + attach)
  attach <session>     Snapshot to cold storage + attach
  recover <session>    Recover from cold storage
  snapshot <session>   Snapshot to cold storage only
  clean <session>      Remove cold storage snapshot
  clean-all            Remove all cold storage snapshots
  list-frozen          List frozen sessions
  list-cold            List recoverable sessions
  list-cold-all        List all cold storage snapshots`)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func requireSession(args []string, cmd string) string {
	if len(args) < 3 {
		fatal("Usage: tmux-fridge %s <session>", cmd)
	}
	return args[2]
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "freeze":
		session := requireSession(os.Args, "freeze")
		if err := Freeze(session); err != nil {
			fatal("freeze: %v", err)
		}

	case "unfreeze":
		session := requireSession(os.Args, "unfreeze")
		if err := Unfreeze(session); err != nil {
			fatal("unfreeze: %v", err)
		}

	case "attach":
		session := requireSession(os.Args, "attach")
		if err := Attach(session); err != nil {
			fatal("attach: %v", err)
		}

	case "recover":
		session := requireSession(os.Args, "recover")
		if err := Recover(session); err != nil {
			fatal("recover: %v", err)
		}

	case "snapshot":
		session := requireSession(os.Args, "snapshot")
		if err := Snapshot(session); err != nil {
			fatal("snapshot: %v", err)
		}

	case "clean":
		session := requireSession(os.Args, "clean")
		if err := Clean(session); err != nil {
			fatal("clean: %v", err)
		}

	case "clean-all":
		if err := CleanAll(); err != nil {
			fatal("clean-all: %v", err)
		}

	case "list-frozen":
		if err := EnsureDirs(); err != nil {
			fatal("list-frozen: %v", err)
		}
		sessions, err := ListFrozen()
		if err != nil {
			fatal("list-frozen: %v", err)
		}
		for _, s := range sessions {
			fmt.Println(s)
		}

	case "list-cold":
		if err := EnsureDirs(); err != nil {
			fatal("list-cold: %v", err)
		}
		sessions, err := ListCold()
		if err != nil {
			fatal("list-cold: %v", err)
		}
		for _, s := range sessions {
			fmt.Println(s)
		}

	case "list-cold-all":
		if err := EnsureDirs(); err != nil {
			fatal("list-cold-all: %v", err)
		}
		sessions, err := ListColdAll()
		if err != nil {
			fatal("list-cold-all: %v", err)
		}
		for _, s := range sessions {
			fmt.Println(s)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		usage()
		os.Exit(1)
	}
}
