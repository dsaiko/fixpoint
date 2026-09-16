package main

import (
	"bytes"
	"strings"
	"testing"
)

// stats reads a finished run back and needs no config at all, which is the
// property this covers end to end: the fixture runs once, then the subcommand
// prices it from the artifacts alone.
func TestStatsReportsAFinishedRun(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("a real bug")))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	if got := run([]string{"-config", cfg}, &buf, &buf); got == 1 {
		t.Fatalf("setup run failed; stderr:\n%s", buf.String())
	}

	// The fixture's logs live outside the project root, so the path is passed
	// explicitly -- which is also the shape an operator uses after changing
	// logs.dir.
	root := runDirOf(t, f)
	var out, errOut bytes.Buffer
	if got := run([]string{"stats", root}, &out, &errOut); got != 0 {
		t.Fatalf("stats = %d; stderr:\n%s", got, errOut.String())
	}
	s := out.String()
	if !strings.Contains(s, "mock-rev") {
		t.Errorf("the reviewer is missing from the table:\n%s", s)
	}
	if !strings.Contains(s, "1 run(s)") {
		t.Errorf("the run count is wrong:\n%s", s)
	}
	// stdout carries the report and stderr the log, so `fixpoint stats | ...` works.
	if errOut.Len() != 0 {
		t.Errorf("stats wrote to stderr: %s", errOut.String())
	}
}

func TestStatsOnAnEmptyRootExplainsItself(t *testing.T) {
	newFixture(t)
	var out, errOut bytes.Buffer
	if got := run([]string{"stats", t.TempDir()}, &out, &errOut); got != 0 {
		t.Fatalf("stats = %d; stderr:\n%s", got, errOut.String())
	}
	if !strings.Contains(out.String(), "no runs found") {
		t.Errorf("unhelpful empty report:\n%s", out.String())
	}
}

func TestStatsRejectsMoreThanOnePath(t *testing.T) {
	newFixture(t)
	var out, errOut bytes.Buffer
	if got := run([]string{"stats", t.TempDir(), t.TempDir()}, &out, &errOut); got != 2 {
		t.Fatalf("stats with two paths = %d, want 2 (usage error)", got)
	}
}
