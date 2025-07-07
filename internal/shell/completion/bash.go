package completion

// GetBashCompletion returns the bash completion script
func GetBashCompletion() string {
	return `#! /bin/bash

_assetcap_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="version completion config assets tasks sprint help"

    case "${prev}" in
        "completion")
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        "config")
            COMPREPLY=( $(compgen -W "init show validate" -- ${cur}) )
            return 0
            ;;
        "assets")
            COMPREPLY=( $(compgen -W "create list show update sync enrich keywords documentation tasks" -- ${cur}) )
            return 0
            ;;
        "documentation")
            COMPREPLY=( $(compgen -W "update" -- ${cur}) )
            return 0
            ;;
        "tasks")
            case "${COMP_WORDS[COMP_CWORD-2]}" in
                "assets")
                    COMPREPLY=( $(compgen -W "increment decrement" -- ${cur}) )
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "fetch show classify inspect migrate" -- ${cur}) )
                    ;;
            esac
            return 0
            ;;
        "sprint")
            COMPREPLY=( $(compgen -W "list allocate" -- ${cur}) )
            return 0
            ;;
        *)
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _assetcap_completion assetcap`
}
