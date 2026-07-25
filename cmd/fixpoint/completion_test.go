package main

import (
	"bytes"
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
