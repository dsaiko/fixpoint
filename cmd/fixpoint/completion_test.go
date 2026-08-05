package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteCompletionSupportedShells(t *testing.T) {
	// Each script must register itself with the shell's own mechanism, or it is a
	// file that does nothing.
	wantRegistration := map[string]string{
		"bash": "complete -F _fixpoint fixpoint",
		"zsh":  "compdef _fixpoint fixpoint",
		"fish": "complete -c fixpoint",
	}
	for shell, registration := range wantRegistration {
		t.Run(shell, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := writeCompletion([]string{shell}, &out, &errOut); code != 0 {
				t.Fatalf("writeCompletion(%s) = %d, stderr: %s", shell, code, errOut.String())
			}
			script := out.String()
			if !strings.Contains(script, registration) {
				t.Errorf("%s script never registers itself (want %q):\n%s", shell, registration, script)
			}
			// The contract between the generated script and the binary: candidates come
			// from querying at completion time, so a bundle change takes effect without
			// regenerating. If this flag is ever renamed, every installed script breaks
			// silently -- completion just stops offering anything.
			if !strings.Contains(script, "--list --porcelain") {
				t.Errorf("%s script must query `fixpoint --list --porcelain`:\n%s", shell, script)
			}
			// Base configs must not be offered: running one fails validation.
			if !strings.Contains(script, "runnable") {
				t.Errorf("%s script must filter on the runnable column:\n%s", shell, script)
			}
			if !strings.Contains(script, "Install with:") && shell != "zsh" {
				t.Errorf("%s script should say how to install it:\n%s", shell, script)
			}
		})
	}
}

// completionFlags is written by hand on purpose (see its comment: a generated
// script is sourced for months, so deriving the list would only be right until the
// next upgrade). The cost of that choice is that it silently falls behind, which is
// exactly what happened when -base-ref was added: the flag worked, and completion
// never offered it. Deriving it AT TEST TIME keeps the deliberate hand-written list
// and still fails the build when someone adds a flag without it.
//
// The source of truth is --help, because that is what the FlagSet itself prints:
// a flag missing from both is a flag nobody can discover.
func TestCompletionFlagsCoverEveryRealFlag(t *testing.T) {
	var out bytes.Buffer
	run([]string{"-h"}, &out, &out) // usage goes to the writer, exit code is not the subject

	declared := map[string]bool{}
	for _, f := range strings.Fields(completionFlags) {
		declared[strings.TrimLeft(f, "-")] = true
	}

	var missing, stale []string
	actual := map[string]bool{}
	for _, line := range strings.Split(out.String(), "\n") {
		// PrintDefaults indents each flag as "  -name" with its description below.
		name, ok := strings.CutPrefix(line, "  -")
		if !ok {
			continue
		}
		name, _, _ = strings.Cut(name, " ") // drop the value placeholder ("-config string")
		if name == "" {
			continue
		}
		actual[name] = true
		if !declared[name] {
			missing = append(missing, name)
		}
	}
	if len(actual) == 0 {
		t.Fatalf("parsed no flags out of --help; the test cannot detect drift:\n%s", out.String())
	}
	for name := range declared {
		if !actual[name] {
			stale = append(stale, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("flags exist but completion never offers them: %v\nadd them to completionFlags in completion.go", missing)
	}
	if len(stale) > 0 {
		t.Errorf("completion offers flags that no longer exist: %v\nremove them from completionFlags in completion.go", stale)
	}
}

// An unknown or missing shell must fail with usage, not emit a script for some
// default shell the caller did not ask for.
func TestWriteCompletionRejectsBadArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no shell", nil},
		{"two shells", []string{"bash", "zsh"}},
		{"unknown shell", []string{"powershell"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := writeCompletion(tc.args, &out, &errOut); code != 2 {
				t.Errorf("exit = %d, want 2 (usage error)", code)
			}
			if out.Len() != 0 {
				t.Errorf("a rejected request must emit no script, got:\n%s", out.String())
			}
			if errOut.Len() == 0 {
				t.Error("a rejected request must explain itself on stderr")
			}
		})
	}
}

// The zsh script is the one that shows descriptions, so it must pass name:desc
// pairs -- that is what produces the two-column layout, rather than a bare list.
func TestZshCompletionOffersDescriptions(t *testing.T) {
	var out, errOut bytes.Buffer
	writeCompletion([]string{"zsh"}, &out, &errOut)
	script := out.String()
	if !strings.Contains(script, `candidates+=("${name}:${desc}")`) {
		t.Error("zsh script must build name:description pairs for _describe")
	}
	// A pipeline would put the loop in a subshell and discard every append, so
	// completion would silently offer nothing. This caught a real bug. Asserted
	// against CODE only: both scripts carry comments naming the construct they
	// avoid, and matching those would fail on the explanation rather than the bug.
	if strings.Contains(code(script), "| while") {
		t.Error("zsh script must not read the listing through a pipeline: the loop would run in a subshell and lose its appends")
	}
	// macOS still ships bash 3.2, which has no mapfile.
	var bashOut bytes.Buffer
	writeCompletion([]string{"bash"}, &bashOut, &errOut)
	if strings.Contains(code(bashOut.String()), "mapfile") {
		t.Error("bash script must avoid mapfile: it is bash 4+, and macOS ships 3.2")
	}
}

// Config names are FILENAMES from a bundle directory, and the first one searched is
// <project>/config -- inside the repository under review. `compgen -W` splits its
// wordlist on IFS and EXPANDS each word, so a candidate reaching -W means a clone
// shipping config/'$(cmd)'.yaml executes cmd when the operator presses TAB, before
// any run and so before any trust gate. Only the constant flag list may go through
// -W; names must be read into COMPREPLY.
//
// This RUNS the generated script against a fixpoint that emits hostile candidate
// names, because the property is about what the script does, not how it is
// spelled: an unquoted wordlist variable, an eval, or an unquoted expansion while
// appending to COMPREPLY all restore command execution on TAB while leaving any
// source-text assertion green.
func TestBashCompletionNeverExpandsConfigNames(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not available: %v", err)
	}

	var out, errOut bytes.Buffer
	if code := writeCompletion([]string{"bash"}, &out, &errOut); code != 0 {
		t.Fatalf("writeCompletion(bash) = %d, stderr: %s", code, errOut.String())
	}

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fixpoint.bash")
	if err := os.WriteFile(scriptPath, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	// The sentinel is created ONLY by executing a candidate name. Its absence
	// afterwards is the security property: no candidate ran as a command.
	sentinel := filepath.Join(dir, "pwned")
	// A separate working directory holding one file, so a candidate glob that got
	// expanded shows up as this filename instead of staying literal.
	work := filepath.Join(dir, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "glob-bait.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	// Every hostile name a bundle directory could really carry: command
	// substitution in both spellings, a glob, and embedded whitespace. Plus a
	// benign name to prove completion still works, and a base config to prove the
	// runnable filter still holds.
	hostile := []string{
		`$(touch ` + sentinel + `)`,
		"`touch " + sentinel + "`",
		"*",
		"two words",
		"safe-name",
	}
	fake := filepath.Join(dir, "bin", "fixpoint")
	if err := os.Mkdir(filepath.Dir(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	var lister strings.Builder
	lister.WriteString("#!/bin/sh\n")
	for _, name := range hostile {
		// Single-quoted in the emitting script so the fake lists these names
		// verbatim; ' cannot appear in them, so the quoting is unambiguous.
		fmt.Fprintf(&lister, "printf '%%s\\t%%s\\t%%s\\n' '%s' runnable 'a description'\n", name)
	}
	lister.WriteString("printf '%s\\t%s\\t%s\\n' 'base-only' base 'not runnable'\n")
	if err := os.WriteFile(fake, []byte(lister.String()), 0o755); err != nil {
		t.Fatal(err)
	}

	// complete() the empty word: every runnable candidate is offered. One
	// COMPREPLY element per line, bracketed so embedded whitespace is visible as
	// one element rather than two.
	driver := `set -u
source "$1"
COMP_WORDS=(fixpoint "$2")
COMP_CWORD=1
_fixpoint
for r in ${COMPREPLY[@]+"${COMPREPLY[@]}"}; do printf '[%s]\n' "$r"; done
`
	run := func(cur string) []string {
		t.Helper()
		cmd := exec.Command(bash, "-c", driver, "bash", scriptPath, cur)
		cmd.Dir = work
		cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(fake)+string(os.PathListSeparator)+os.Getenv("PATH"))
		got, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("running the generated completion (cur=%q) failed: %v\n%s", cur, err, got)
		}
		var reply []string
		for _, line := range strings.Split(strings.TrimSpace(string(got)), "\n") {
			if line != "" {
				reply = append(reply, strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			}
		}
		return reply
	}

	all := run("")
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatalf("pressing TAB executed a config name: the sentinel %s was created", sentinel)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	want := append([]string(nil), hostile...)
	if !reflect.DeepEqual(all, want) {
		t.Errorf("COMPREPLY = %q, want the candidate names literally and unsplit: %q", all, want)
	}

	// Prefix filtering still works, and still filters on the literal name.
	if got := run("safe"); !reflect.DeepEqual(got, []string{"safe-name"}) {
		t.Errorf("COMPREPLY for prefix \"safe\" = %q, want [safe-name]", got)
	}
	// A base config is never offered: running one fails validation.
	for _, r := range all {
		if r == "base-only" {
			t.Error("a base (non-runnable) config was offered as a completion")
		}
	}

	// Source-level backstop for the two spellings that caused this: they would
	// also fail above, but naming them keeps the reason visible at the call site.
	script := code(out.String())
	if strings.Contains(script, `compgen -W "$`) {
		t.Errorf("bash script passes a variable wordlist to compgen -W, which expands it:\n%s", script)
	}
	if strings.Contains(script, "compgen -W \"$(") {
		t.Errorf("bash script passes command output straight to compgen -W:\n%s", script)
	}
}

// code strips comment lines from a shell script, so an assertion about what the
// script DOES is not satisfied (or broken) by a comment about what it avoids.
func code(script string) string {
	var kept []string
	for _, line := range strings.Split(script, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
