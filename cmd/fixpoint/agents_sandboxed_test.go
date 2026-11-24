package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// -agents-sandboxed end to end at the CLI: the flag reaches the Overrides literal
// in run (a field written at the call site and never read back is a silently
// false bool, not a compile error), it is named in the applied overrides, and the
// one refusal it lifts is lifted only beside a sandbox.command.
func TestRunAgentsSandboxedFlag(t *testing.T) {
	// a config whose read-only reviewer disables its CLI's own sandbox, optionally
	// with a wrapper in front of every agent.
	build := func(t *testing.T, wrapper bool) string {
		t.Helper()
		f := newFixture(t)
		p := f.configFile("directory", "", "")
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		cmd := fmt.Sprintf("%q", f.script)
		from := "mock-rev:\n    command: [" + cmd + "]"
		if !strings.Contains(string(body), from) {
			t.Fatalf("fixture config changed shape; %q not found:\n%s", from, body)
		}
		out := strings.Replace(string(body), from, "mock-rev:\n    command: ["+cmd+", --yolo]", 1)
		if wrapper {
			out += "sandbox:\n  command: [env, \"{{target}}\", \"{{target_mode}}\", --]\n"
		}
		if err := os.WriteFile(p, []byte(out), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("refused without the flag, wrapper or not", func(t *testing.T) {
		for _, wrapper := range []bool{false, true} {
			var buf bytes.Buffer
			if got := run([]string{"-config", build(t, wrapper), "-check", "-trusted-target"}, &buf, &buf); got == 0 {
				t.Errorf("wrapper=%v: a read-only reviewer passing --yolo was accepted without -agents-sandboxed:\n%s", wrapper, buf.String())
			}
			if !strings.Contains(buf.String(), "lets the agent write files") {
				t.Errorf("wrapper=%v: refused, but not for the bypass flag:\n%s", wrapper, buf.String())
			}
		}
	})
	t.Run("accepted with the flag and a wrapper, and recorded", func(t *testing.T) {
		var buf bytes.Buffer
		if got := run([]string{"-config", build(t, true), "-check", "-trusted-target", "-agents-sandboxed"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-agents-sandboxed) = %d, want 0:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "agents_sandboxed=true") {
			t.Errorf("the applied overrides do not record the flag:\n%s", buf.String())
		}
	})
	t.Run("an error with the flag and no wrapper", func(t *testing.T) {
		var buf bytes.Buffer
		if got := run([]string{"-config", build(t, false), "-check", "-trusted-target", "-agents-sandboxed"}, &buf, &buf); got == 0 {
			t.Fatalf("-agents-sandboxed without sandbox.command was accepted:\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "sets no sandbox.command") {
			t.Errorf("the error does not say the flag claims a sandbox that is not there:\n%s", buf.String())
		}
	})
}
