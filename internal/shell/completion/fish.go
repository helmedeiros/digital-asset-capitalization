package completion

// GetFishCompletion returns the fish completion script
func GetFishCompletion() string {
	return `function __fish_assetcap_no_subcommand
    set cmd (commandline -opc)
    if [ (count $cmd) -eq 1 ]
        return 0
    end
    return 1
end

# Main commands
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a version -d 'Show version information'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a completion -d 'Generate shell completion scripts'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a config -d 'Manage configuration settings'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a assets -d 'Manage digital assets'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a tasks -d 'Manage tasks from various platforms'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a sprint -d 'Manage sprint-related operations'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a help -d 'Shows a list of commands or help for one command'

# Completion subcommands
complete -c assetcap -n '__fish_seen_subcommand_from completion' -a bash -d 'Generate bash completion script'
complete -c assetcap -n '__fish_seen_subcommand_from completion' -a zsh -d 'Generate zsh completion script'
complete -c assetcap -n '__fish_seen_subcommand_from completion' -a fish -d 'Generate fish completion script'

# Config subcommands
complete -c assetcap -n '__fish_seen_subcommand_from config' -a init -d 'Initialize configuration'
complete -c assetcap -n '__fish_seen_subcommand_from config' -a show -d 'Show current configuration'
complete -c assetcap -n '__fish_seen_subcommand_from config' -a validate -d 'Validate current configuration'

# Asset subcommands
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a create -d 'Create a new asset'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a list -d 'List all assets'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a show -d 'Show detailed information about an asset'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a update -d 'Update an asset description and metadata'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a sync -d 'Sync assets from Confluence'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a enrich -d 'Enrich asset fields using LLaMA 3'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a keywords -d 'Generate keywords for an asset using LLaMA 3'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a documentation -d 'Manage asset documentation'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a tasks -d 'Manage asset tasks'

# Asset documentation subcommands
complete -c assetcap -n '__fish_seen_subcommand_from documentation' -a update -d 'Mark asset documentation as updated'

# Asset tasks subcommands
function __fish_assetcap_assets_tasks
    set cmd (commandline -opc)
    if contains assets $cmd; and contains tasks $cmd
        return 0
    end
    return 1
end

complete -c assetcap -n '__fish_assetcap_assets_tasks' -a increment -d 'Increment task count for an asset'
complete -c assetcap -n '__fish_assetcap_assets_tasks' -a decrement -d 'Decrement task count for an asset'

# Task subcommands
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a fetch -d 'Fetch tasks from a platform (e.g., Jira)'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a show -d 'Show tasks for a project and sprint'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a classify -d 'Classify tasks for a specific project and sprint'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a inspect -d 'Inspect a specific task by its key'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a migrate -d 'Migrate sprint data from comma-separated strings to arrays'

# Sprint subcommands
complete -c assetcap -n '__fish_seen_subcommand_from sprint' -a list -d 'List sprints for a project and time period'
complete -c assetcap -n '__fish_seen_subcommand_from sprint' -a allocate -d 'Calculate time allocation for JIRA issues in a sprint'`
}
