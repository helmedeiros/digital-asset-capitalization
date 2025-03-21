package completion

// GetZshCompletion returns the zsh completion script
func GetZshCompletion() string {
	return `#compdef assetcap

_assetcap() {
    local -a commands
    commands=(
        'timeallocation-calc:Calculate time allocation for JIRA issues'
        'assets:Manage digital assets'
        'completion:Generate shell completion scripts'
        'help:Shows a list of commands or help for one command'
    )

    local -a asset_commands
    asset_commands=(
        'create:Create a new asset'
        'list:List all assets'
        'contribution-type:Manage contribution types'
        'documentation:Manage asset documentation'
        'tasks:Manage asset tasks'
    )

    local -a contribution_commands
    contribution_commands=(
        'add:Add a contribution type to an asset'
    )

    local -a documentation_commands
    documentation_commands=(
        'update:Mark asset documentation as updated'
    )

    local -a task_commands
    task_commands=(
        'increment:Increment task count for an asset'
        'decrement:Decrement task count for an asset'
    )

    _arguments -C \
        "1: :{_describe 'command' commands}" \
        "*::arg:->args"

    case $line[1] in
        assets)
            _arguments "1: :{_describe 'asset command' asset_commands}"
            ;;
        contribution-type)
            _arguments "1: :{_describe 'contribution command' contribution_commands}"
            ;;
        documentation)
            _arguments "1: :{_describe 'documentation command' documentation_commands}"
            ;;
        tasks)
            _arguments "1: :{_describe 'task command' task_commands}"
            ;;
    esac
}

compdef _assetcap assetcap`
}
