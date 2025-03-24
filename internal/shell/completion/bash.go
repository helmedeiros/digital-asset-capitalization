package completion

// GetBashCompletion returns the bash completion script
func GetBashCompletion() string {
	return `#! /bin/bash

_assetcap_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="assets completion help"

    case "${prev}" in
        "assets")
            COMPREPLY=( $(compgen -W "create list contribution-type documentation tasks" -- ${cur}) )
            return 0
            ;;
        "contribution-type")
            COMPREPLY=( $(compgen -W "add" -- ${cur}) )
            return 0
            ;;
        "documentation")
            COMPREPLY=( $(compgen -W "update" -- ${cur}) )
            return 0
            ;;
        "tasks")
            COMPREPLY=( $(compgen -W "increment decrement classify show" -- ${cur}) )
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
