package main

import (
	"fmt"
	"io"
)

// writeCompletion prints a shell completion script.
//
// The scripts query `fixpoint --list --porcelain` at completion time rather than
// baking config names into the generated script. That matters because bundles are
// per-project and shadowable: the right candidates depend on which directory you
// are standing in, so a list captured at generation time would be wrong as soon as
// you changed projects.
func writeCompletion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: fixpoint completion bash|zsh|fish\n")
		return 2
	}
	script, ok := completionScripts[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "unsupported shell %q; want bash, zsh, or fish\n", args[0])
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}

var completionScripts = map[string]string{"bash": bashCompletion, "zsh": zshCompletion, "fish": fishCompletion}

// Flag names are listed for completion. They are spelled out rather than derived
// from the FlagSet because the script is generated once and sourced thereafter:
// deriving them would only be accurate until the next upgrade, which is a worse
// kind of wrong than a list a reader can see and check.
const completionFlags = "--list --porcelain --config --review-only --max-iterations --base-ref --pr --target --out " +
	"--trusted-target --trusted-bundle --allow-untrusted-fix --post --post-verdict --post-run --check --check-live " +
	"--no-coverage-check --plan-only --version"

const bashCompletion = `# fixpoint completion for bash. Install with:
#   fixpoint completion bash > /etc/bash_completion.d/fixpoint
# or append to ~/.bashrc:
#   source <(fixpoint completion bash)
# The flag list is a constant baked into this script, so word splitting on
# ` + "`compgen -W`" + ` output is intentional there and is how COMPREPLY is built.
# Config names are NOT passed through -W: see the comment on the read loop below.
# ` + "`mapfile`" + ` would read better than that loop but is bash 4+, and macOS still ships
# bash 3.2 -- this form works from 3.2 through 5.x.
_fixpoint() {
    local cur=${COMP_WORDS[COMP_CWORD]}
    if [[ $cur == -* ]]; then
        COMPREPLY=( $(compgen -W "` + completionFlags + `" -- "$cur") )
        return
    fi
    # Only runnable configs: a base config exists to be inherited via ` + "`extends`" + `,
    # and offering it would invite a run that fails validation.
    #
    # Read into COMPREPLY and prefix-filter here rather than calling
    # ` + "`compgen -W \"$names\"`" + `: -W splits its wordlist on IFS and EXPANDS every word,
    # and these candidates are FILENAMES from a bundle directory -- the first one
    # searched is <project>/config, inside the repository under review. A clone
    # shipping config/'$(cmd)'.yaml would otherwise run cmd on TAB, before any
    # fixpoint run and so before any trust gate.
    local line
    COMPREPLY=()
    while IFS= read -r line; do
        [[ -n $line ]] || continue
        [[ $line == "$cur"* ]] && COMPREPLY+=( "$line" )
    done < <(fixpoint --list --porcelain 2>/dev/null | awk -F'\t' '$2 == "runnable" { print $1 }')
}
complete -F _fixpoint fixpoint
`

const zshCompletion = `#compdef fixpoint
# fixpoint completion for zsh. Install with:
#   fixpoint completion zsh > "${fpath[1]}/_fixpoint"
# or append to ~/.zshrc:
#   source <(fixpoint completion zsh)
_fixpoint() {
    local -a candidates
    if [[ ${words[CURRENT]} == -* ]]; then
        candidates=(` + zshFlagPairs + `)
        _describe -t flags 'flag' candidates
        return
    fi
    # name:description pairs, which is what gives the two-column listing zsh shows
    # for git. Descriptions come from each config's own ` + "`description:`" + ` field.
    #
    # Read via command substitution, NOT ` + "`… | while read`" + `: a pipeline puts the loop
    # in a subshell, so appends to ` + "`candidates`" + ` are discarded and completion silently
    # offers nothing.
    local -a lines
    lines=("${(@f)$(fixpoint --list --porcelain 2>/dev/null)}")
    local line name state desc rest
    for line in $lines; do
        [[ -n $line ]] || continue
        name=${line%%$'\t'*}
        rest=${line#*$'\t'}
        state=${rest%%$'\t'*}
        desc=${rest#*$'\t'}
        [[ $state == runnable ]] || continue
        candidates+=("${name}:${desc}")
    done
    _describe -t configs 'config' candidates
}
# Guarded: sourcing this from ~/.zshrc ABOVE ` + "`compinit`" + ` would otherwise fail with
# "command not found: compdef" on every new shell. When installed into fpath the
# #compdef line at the top does the registration instead.
(( $+functions[compdef] )) && compdef _fixpoint fixpoint
`

// zshFlagPairs are flag:description pairs for zsh's two-column display -- the
// layout that makes `git c<TAB>` readable rather than a bare list of words.
const zshFlagPairs = `'--list:list the configs available here' ` +
	`'--porcelain:with --list, emit a tab-separated form for scripts' ` +
	`'--config:path to a config file instead of a bundle name' ` +
	`'--review-only:one review round, never invoke the coder' ` +
	`'--max-iterations:override loop.max_iterations' ` +
	`'--base-ref:override target.base_ref in git-diff mode' ` +
	`'--pr:override target.pr in pr mode' ` +
	`'--target:point a directory-mode run at a file or a directory' ` +
	`'--out:where a create run writes its deliverable' ` +
	`'--trusted-target:assert the target holds trusted code, permitting fixes' ` +
	`'--trusted-bundle:assert only the bundle inside the target may be run' ` +
	`'--allow-untrusted-fix:permit fix rounds in pr mode' ` +
	`'--post:publish the review on the pull request as a comment' ` +
	`'--post-verdict:with --post, let the review approve or request changes' ` +
	`'--post-run:publish the review a finished run already produced' ` +
	`'--check:validate the configuration and exit' ` +
	`'--check-live:validate, ping every agent, and exit' ` +
	`'--no-coverage-check:implement: run without the coverage rule; reports say unchecked' ` +
	`'--plan-only:implement: plan and validate, then stop without building' ` +
	`'--version:print the version, commit and build date, and exit'`

const fishCompletion = `# fixpoint completion for fish. Install with:
#   fixpoint completion fish > ~/.config/fish/completions/fixpoint.fish
function __fixpoint_configs
    # Only runnable configs; fish takes value<TAB>description pairs.
    fixpoint --list --porcelain 2>/dev/null | while read -l line
        set -l parts (string split \t -- $line)
        if test "$parts[2]" = runnable
            printf '%s\t%s\n' $parts[1] $parts[3]
        end
    end
end

complete -c fixpoint -f
complete -c fixpoint -a '(__fixpoint_configs)'
complete -c fixpoint -l list -d 'list the configs available here'
complete -c fixpoint -l porcelain -d 'with --list, emit a tab-separated form for scripts'
complete -c fixpoint -l config -r -d 'path to a config file instead of a bundle name'
complete -c fixpoint -l review-only -d 'one review round, never invoke the coder'
complete -c fixpoint -l max-iterations -r -d 'override loop.max_iterations'
complete -c fixpoint -l base-ref -r -d 'override target.base_ref in git-diff mode'
complete -c fixpoint -l target -r -d 'point a directory-mode run at a file or a directory'
complete -c fixpoint -l out -r -d 'where a create run writes its deliverable'
complete -c fixpoint -l pr -r -d 'override target.pr in pr mode'
complete -c fixpoint -l trusted-target -d 'assert the target holds trusted code, permitting fixes'
complete -c fixpoint -l trusted-bundle -d 'assert only the bundle inside the target may be run'
complete -c fixpoint -l allow-untrusted-fix -d 'permit fix rounds in pr mode'
complete -c fixpoint -l post -d 'publish the review on the pull request as a comment'
complete -c fixpoint -l post-verdict -d 'with --post, let the review approve or request changes'
complete -c fixpoint -l post-run -r -d 'publish the review a finished run already produced'
complete -c fixpoint -l check -d 'validate the configuration and exit'
complete -c fixpoint -l check-live -d 'validate, ping every agent, and exit'
complete -c fixpoint -l no-coverage-check -d 'implement: run without the coverage rule; reports say unchecked'
complete -c fixpoint -l plan-only -d 'implement: plan and validate, then stop without building'
complete -c fixpoint -l version -d 'print the version, commit and build date, and exit'
`
