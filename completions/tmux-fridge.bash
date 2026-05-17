_tmux-fridge_completions() {
	local cur prev commands
	COMPREPLY=()
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[COMP_CWORD-1]}"
	commands="freeze unfreeze attach recover snapshot clean clean-all list-frozen list-cold list-cold-all"

	if [ "$COMP_CWORD" -eq 1 ]; then
		COMPREPLY=($(compgen -W "$commands" -- "$cur"))
		return 0
	fi

	case "$prev" in
		freeze|attach|snapshot)
			COMPREPLY=($(compgen -W "$(tmux list-sessions -F '#{session_name}' 2>/dev/null)" -- "$cur"))
			;;
		unfreeze)
			COMPREPLY=($(compgen -W "$(tmux-fridge list-frozen 2>/dev/null)" -- "$cur"))
			;;
		recover|clean)
			COMPREPLY=($(compgen -W "$(tmux-fridge list-cold-all 2>/dev/null)" -- "$cur"))
			;;
	esac
}

complete -F _tmux-fridge_completions tmux-fridge
