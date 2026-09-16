package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/replay"
)

// runDirOf returns the single run directory the fixture's logs template produced.
// The template is "<tmp>/logs/{timestamp}/round-{round}", so the run directory is
// the one level holding the summary, the journal and the recording.
func runDirOf(t *testing.T, f *fixture) string {
	t.Helper()
	root := filepath.Dir(filepath.Dir(f.logsDir)) // strip round-{round} and {timestamp}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	if len(dirs) != 1 {
		t.Fatalf("want exactly one run directory under %s, got %v", root, dirs)
	}
	return dirs[0]
}

// The end-to-end property the whole feature exists for: a run recorded once can
// be re-run with no agent at all. The mock agent counts its invocations, so
// "no agent ran" is asserted rather than assumed -- which is the only way to
// prove a replay is not quietly falling through to the live path.
func TestReplayRerunsAFinishedRunWithoutInvokingAnAgent(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("a real bug")))
	cfg := f.configFileWithLogs("directory", "", "  review_only: true", "  replay: true\n")

	var rec bytes.Buffer
	if got := run([]string{"-config", cfg}, &rec, &rec); got != model.ExitChangesRequested {
		t.Fatalf("recording run = %d, want %d; stderr:\n%s", got, model.ExitChangesRequested, rec.String())
	}
	if got := f.invocations(); got != 1 {
		t.Fatalf("recording run made %d invocation(s), want 1", got)
	}
	dir := runDirOf(t, f)
	if _, err := os.Stat(filepath.Join(dir, model.ReplayName)); err != nil {
		t.Fatalf("the run wrote no recording: %v", err)
	}

	var rep bytes.Buffer
	code := run([]string{"-config", cfg, "-review-only", "-replay", dir}, &rep, &rep)
	if code != model.ExitChangesRequested {
		t.Fatalf("replayed run = %d, want %d (the same verdict the recording produced); stderr:\n%s", code, model.ExitChangesRequested, rep.String())
	}
	// The count is still 1: the replay served the reply from the recording and
	// started no process.
	if got := f.invocations(); got != 1 {
		t.Errorf("the replay invoked the agent %d time(s); it must invoke none", got-1)
	}
	if !strings.Contains(rep.String(), "replay: matched the recording exactly") {
		t.Errorf("the replay did not report a clean match:\n%s", rep.String())
	}
}

// A replay whose config no longer asks what the recording answers must say so.
// Without this the run looks ordinary and the divergence is invisible in the
// summary -- see reportDivergence.
func TestReplayReportsAnInvocationTheRecordingLacks(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	cfg := f.configFileWithLogs("directory", "", "  review_only: true", "  replay: true\n")
	var rec bytes.Buffer
	if got := run([]string{"-config", cfg}, &rec, &rec); got != 0 {
		t.Fatalf("recording run = %d; stderr:\n%s", got, rec.String())
	}
	dir := runDirOf(t, f)

	// Replay it under a config whose reviewer is a DIFFERENT agent, so the key the
	// replayed run asks for is not in the recording. Renamed rather than swapped
	// for the coder: a reviewer must be read-only, so pointing the lens at the
	// write-capable mock would fail validation instead of reaching the replay.
	other := strings.ReplaceAll(readFile(t, cfg), "mock-rev", "mock-rev-renamed")
	otherPath := filepath.Join(t.TempDir(), "other.yaml")
	if err := os.WriteFile(otherPath, []byte(other), 0o600); err != nil {
		t.Fatal(err)
	}
	var rep bytes.Buffer
	if got := run([]string{"-config", otherPath, "-review-only", "-replay", dir}, &rep, &rep); got == 0 {
		t.Fatalf("a replay that diverged from its recording exited 0; stderr:\n%s", rep.String())
	}
	if !strings.Contains(rep.String(), "no reply for") {
		t.Errorf("the run does not say which invocation the recording lacks:\n%s", rep.String())
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A fix round's edits are in no recording, so replaying one would pair a coder
// claiming edits with a tree that has none. Refused at startup, before anything
// runs.
func TestReplayRefusesAFixRun(t *testing.T) {
	f := newFixture(t)
	cfg := f.configFileWithLogs("directory", "", "  max_iterations: 3", "  replay: true\n")
	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-trusted-target", "-replay", t.TempDir()}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 (refusal); stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "-review-only") {
		t.Errorf("the refusal does not name the flag that fixes it:\n%s", buf.String())
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("the refused run still invoked %d agent(s)", got)
	}
}

// The refusal for a recording that does not exist has to name the switch that
// would have produced one, because the cause is a setting on the run that is
// already over.
func TestReplayExplainsAMissingRecording(t *testing.T) {
	f := newFixture(t)
	cfg := f.configFile("directory", "", "  review_only: true")
	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-review-only", "-replay", t.TempDir()}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "no recording at") {
		t.Errorf("unhelpful refusal:\n%s", buf.String())
	}
}

// A run made WITHOUT logs.replay must write no recording: the switch is the
// operator's, and a default-on artifact nobody asked for is a second on-disk copy
// of every agent reply.
func TestNoRecordingWhenReplayIsOff(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	cfg := f.configFile("directory", "", "  review_only: true")
	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d; stderr:\n%s", got, buf.String())
	}
	if _, err := os.Stat(filepath.Join(runDirOf(t, f), model.ReplayName)); !os.IsNotExist(err) {
		t.Errorf("a run with logs.replay off wrote a recording (err = %v)", err)
	}
}

// A replay must invoke nothing, and the agent PING is the path that would
// otherwise slip past that: it runs before the first round, launches every
// configured CLI for real, and is on by default. A replay on a machine without
// the agent CLI installed -- which is the machine a replay is most useful on --
// would fail there before serving a single recorded reply.
func TestReplaySkipsTheAgentPing(t *testing.T) {
	f := newFixture(t)
	// The ping takes the first invocation ordinal and the review the second, so
	// both have to be scripted -- which is itself the point: a ping IS an agent
	// launch.
	f.respond(1, "OK")
	f.respond(2, reviewResponse(t))
	// ping_agents on, unlike the fixture's default: this is the setting under test.
	cfg := f.configFileWithLogs("directory", "", "  review_only: true", "  replay: true\n")
	withPing := strings.ReplaceAll(readFile(t, cfg), "ping_agents: false", "ping_agents: true")
	pingPath := filepath.Join(t.TempDir(), "ping.yaml")
	if err := os.WriteFile(pingPath, []byte(withPing), 0o600); err != nil {
		t.Fatal(err)
	}

	var rec bytes.Buffer
	if got := run([]string{"-config", pingPath}, &rec, &rec); got != 0 {
		t.Fatalf("recording run = %d; stderr:\n%s", got, rec.String())
	}
	// The recording run pinged (1) and reviewed (1).
	before := f.invocations()
	dir := runDirOf(t, f)

	var rep bytes.Buffer
	if got := run([]string{"-config", pingPath, "-review-only", "-replay", dir}, &rep, &rep); got != 0 {
		t.Fatalf("replayed run = %d; stderr:\n%s", got, rep.String())
	}
	if got := f.invocations(); got != before {
		t.Errorf("the replay launched %d agent process(es); it must launch none", got-before)
	}
	if !strings.Contains(rep.String(), "skipping the agent ping") {
		t.Errorf("the run does not say the ping was skipped:\n%s", rep.String())
	}
}

// -check-live is the ping; -replay suppresses pinging. Reporting "all agents
// responding" without having asked any is the one answer that command must never
// give, so the combination is refused.
func TestReplayAndCheckLiveAreRefusedTogether(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	cfg := f.configFileWithLogs("directory", "", "  review_only: true", "  replay: true\n")
	var rec bytes.Buffer
	if got := run([]string{"-config", cfg}, &rec, &rec); got != 0 {
		t.Fatalf("recording run = %d; stderr:\n%s", got, rec.String())
	}
	dir := runDirOf(t, f)

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-review-only", "-replay", dir, "-check-live"}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 (refusal); stderr:\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), "all agents responding") {
		t.Errorf("check-live claimed a green it never measured:\n%s", buf.String())
	}
}

// REGRESSION (review run 20260916-085129, finding i9). A recording cannot be
// authenticated -- it is only files -- and a pull request that commits
// .fixpoint/<ts>/replay.jsonl has it written into the worktree by
// `gh pr checkout` whatever .gitignore says. Replaying it would let the PR author
// write its own findings, verdict and review body; and because the replayed run
// writes a FRESH untracked directory, a later -post-run over that one passes the
// publishing provenance gate and posts the author's verdict under the operator's
// identity. The same rule -post-run applies therefore has to apply here.
func TestReplayRefusesARecordingCommittedIntoTheRepository(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	cfg := f.configFileWithLogs("directory", "", "  review_only: true", "  replay: true\n")
	var rec bytes.Buffer
	if got := run([]string{"-config", cfg}, &rec, &rec); got != 0 {
		t.Fatalf("recording run = %d; stderr:\n%s", got, rec.String())
	}
	ownRun := runDirOf(t, f)

	// Plant that same recording inside the repository under review and COMMIT it,
	// which is the one property a directory fixpoint wrote never has.
	planted := filepath.Join(f.repo, "planted-run")
	if err := os.MkdirAll(planted, 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(ownRun, model.ReplayName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planted, model.ReplayName), body, 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, f.repo, "add", "planted-run")
	gitIn(t, f.repo, "commit", "-m", "a pull request ships its own recording")

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-review-only", "-replay", planted}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 (refusal); stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "tracked by git") {
		t.Errorf("the refusal does not say why the recording is untrusted:\n%s", buf.String())
	}
	// And the operator's own, untracked, recording still replays.
	var ok bytes.Buffer
	if got := run([]string{"-config", cfg, "-review-only", "-replay", ownRun}, &ok, &ok); got != 0 {
		t.Fatalf("the operator's own recording was refused: %d\n%s", got, ok.String())
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// The shipped default is a behavior: every run made from the shipped bundle
// records one, and nothing else in the suite would notice it being flipped off
// (review run 20260916-085129, finding i22).
func TestShippedBundleRecordsAReplay(t *testing.T) {
	l, err := config.LoadBundle(&config.Resolver{Bundles: []string{"../../config"}}, "review-code", "", config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if !l.Config.Logs.Replay {
		t.Error("the shipped bundle does not record a replay; a finished run cannot then be re-run without quota")
	}
}

// The implement refusal and the side-effect roles the recording cannot restore
// (review run 20260916-085129, finding i23). attachReplay refuses the pipeline;
// internal/replay refuses the roles for a caller that does not come through it.
func TestReplayRefusesEverySideEffectRole(t *testing.T) {
	for _, role := range []string{"fix", "task", "plan"} {
		t.Run(role, func(t *testing.T) {
			dir := t.TempDir()
			line := fmt.Sprintf(`{"v":%d,"seq":1,"at":"2026-09-16T08:00:00Z","role":%q,"agent":"a","prompt":"p","round":1,"prompt_sha256":"x","stdout":"y"}`+"\n",
				model.ReplayVersion, role)
			if err := os.WriteFile(filepath.Join(dir, model.ReplayName), []byte(line), 0o600); err != nil {
				t.Fatal(err)
			}
			src, err := replay.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := src.Serve(role, "a", "p", 1, "prompt"); !errors.Is(err, replay.ErrNotReviewOnly) {
				t.Fatalf("role %q was served from a recording that cannot hold its side effects: %v", role, err)
			}
		})
	}
}
