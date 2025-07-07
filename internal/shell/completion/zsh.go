package completion

// GetZshCompletion returns the zsh completion script
func GetZshCompletion() string {
	return `#compdef assetcap

_assetcap() {
    local -a commands
    commands=(
        'version:Show version information'
        'completion:Generate shell completion scripts'
        'config:Manage configuration settings'
        'assets:Manage digital assets'
        'tasks:Manage tasks from various platforms'
        'sprint:Manage sprint-related operations'
        'help:Shows a list of commands or help for one command'
    )

    local -a completion_commands
    completion_commands=(
        'bash:Generate bash completion script'
        'zsh:Generate zsh completion script'
        'fish:Generate fish completion script'
    )

    local -a config_commands
    config_commands=(
        'init:Initialize configuration'
        'show:Show current configuration'
        'validate:Validate current configuration'
    )

    local -a asset_commands
    asset_commands=(
        'create:Create a new asset'
        'list:List all assets'
        'show:Show detailed information about an asset'
        'update:Update an asset description and metadata'
        'sync:Sync assets from Confluence'
        'enrich:Enrich asset fields using LLaMA 3'
        'keywords:Generate keywords for an asset using LLaMA 3'
        'documentation:Manage asset documentation'
        'tasks:Manage asset tasks'
    )

    local -a documentation_commands
    documentation_commands=(
        'update:Mark asset documentation as updated'
    )

    local -a asset_task_commands
    asset_task_commands=(
        'increment:Increment task count for an asset'
        'decrement:Decrement task count for an asset'
    )

    local -a task_commands
    task_commands=(
        'fetch:Fetch tasks from a platform (e.g., Jira)'
        'show:Show tasks for a project and sprint'
        'classify:Classify tasks for a specific project and sprint'
        'inspect:Inspect a specific task by its key'
        'migrate:Migrate sprint data from comma-separated strings to arrays'
    )

    local -a sprint_commands
    sprint_commands=(
        'list:List sprints for a project and time period'
        'allocate:Calculate time allocation for JIRA issues in a sprint'
    )

    _arguments -C \
        "1: :{_describe 'command' commands}" \
        "*::arg:->args"

    case $line[1] in
        completion)
            _arguments "1: :{_describe 'completion command' completion_commands}"
            ;;
        config)
            _arguments "1: :{_describe 'config command' config_commands}"
            ;;
        assets)
            _arguments "1: :{_describe 'asset command' asset_commands}"
            case $line[2] in
                documentation)
                    _arguments "1: :{_describe 'documentation command' documentation_commands}"
                    ;;
                tasks)
                    _arguments "1: :{_describe 'asset task command' asset_task_commands}"
                    ;;
            esac
            ;;
        tasks)
            _arguments "1: :{_describe 'task command' task_commands}"
            ;;
        sprint)
            _arguments "1: :{_describe 'sprint command' sprint_commands}"
            ;;
    esac
}

compdef _assetcap assetcap`
}
