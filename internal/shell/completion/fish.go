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

complete -c assetcap -n '__fish_assetcap_no_subcommand' -a assets -d 'Manage digital assets'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a completion -d 'Generate shell completion scripts'
complete -c assetcap -n '__fish_assetcap_no_subcommand' -a help -d 'Shows a list of commands or help for one command'

complete -c assetcap -n '__fish_seen_subcommand_from assets' -a create -d 'Create a new asset'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a list -d 'List all assets'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a contribution-type -d 'Manage contribution types'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a documentation -d 'Manage asset documentation'
complete -c assetcap -n '__fish_seen_subcommand_from assets' -a tasks -d 'Manage asset tasks'

complete -c assetcap -n '__fish_seen_subcommand_from contribution-type' -a add -d 'Add a contribution type to an asset'
complete -c assetcap -n '__fish_seen_subcommand_from documentation' -a update -d 'Mark asset documentation as updated'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a increment -d 'Increment task count for an asset'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a decrement -d 'Decrement task count for an asset'
complete -c assetcap -n '__fish_seen_subcommand_from tasks' -a classify -d 'Classify tasks for a project and sprint'`
}
