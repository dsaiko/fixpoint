package logstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

func newStore(t *testing.T, formats ...string) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "logs")
	s, err := New(config.Logs{
		Dir:             filepath.Join(dir, "{timestamp}", "round-{round}"),
		Formats:         formats,
		Pattern:         "{role}-{agent}-{prompt}.{ext}",
		SummaryPattern:  "summary.{ext}",
		TimestampFormat: "20060102-150405",
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

// singleRunDir returns the store's one run directory under the logs parent.
func singleRunDir(t *testing.T, dir string) string {
	t.Helper()
	runs, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run dir, got %d", len(runs))
	}
	return filepath.Join(dir, runs[0].Name())
}

// filesIn lists the files (ignoring subdirectories) directly inside dir.
func filesIn(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	return out
}

// roundFiles lists the step files written for one round of the single run.
func roundFiles(t *testing.T, dir string, round int) map[string]string {
	t.Helper()
	return filesIn(t, filepath.Join(singleRunDir(t, dir), fmt.Sprintf("round-%d", round)))
}

func TestStepWritesAllFormats(t *testing.T) {
	s, dir := newStore(t, "md", "json", "raw")
	parsed := model.ReviewOutput{Findings: []model.ReviewFinding{{Title: "bug"}}}
	if err := s.Step("review", "claude", "review-bugs", 1, parsed, "# md content", "$ raw content"); err != nil {
		t.Fatal(err)
	}
	files := roundFiles(t, dir, 1)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %v", files)
	}
	if got := files["review-claude-review-bugs.md"]; got != "# md content" {
		t.Errorf("md content = %q", got)
	}
	if got := files["review-claude-review-bugs.raw"]; got != "$ raw content" {
		t.Errorf("raw content = %q", got)
	}
	var out model.ReviewOutput
	if err := json.Unmarshal([]byte(files["review-claude-review-bugs.json"]), &out); err != nil {
		t.Fatalf("json log: %v", err)
	}
	if len(out.Findings) != 1 || out.Findings[0].Title != "bug" {
		t.Errorf("json roundtrip = %+v", out)
	}
}

// Config validation accepts a logs.pattern with directory components (e.g.
// "{role}/{agent}/..."), so the rendered path's parent directory may not exist
// yet. Step, Prompt, and Summary must create it before writing rather than
// failing with ENOENT and silently losing the artifact.
func TestNestedPatternCreatesParentDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	s, err := New(config.Logs{
		Dir:             filepath.Join(dir, "{timestamp}", "round-{round}"),
		Formats:         []string{"md"},
		Pattern:         "{role}/{agent}/{prompt}-{round}.{ext}",
		SummaryPattern:  "reports/summary.{ext}",
		TimestampFormat: "20060102-150405",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Step("review", "claude", "review-bugs", 1, nil, "# md content", ""); err != nil {
		t.Fatalf("Step into a nested pattern: %v", err)
	}
	if err := s.Prompt("fix", "claude-coder", "fix", 1, "prompt text"); err != nil {
		t.Fatalf("Prompt into a nested pattern: %v", err)
	}
	run := singleRunDir(t, dir)
	step := filepath.Join(run, "round-1", "review", "claude", "review-bugs-1.md")
	if got, err := os.ReadFile(step); err != nil || string(got) != "# md content" {
		t.Errorf("nested step file: got %q err %v", got, err)
	}
	if _, err := s.Summary(&model.RunSummary{Termination: model.TermConverged}); err != nil {
		t.Fatalf("Summary into a nested pattern: %v", err)
	}
	if _, err := os.Stat(filepath.Join(run, "reports", "summary.md")); err != nil {
		t.Errorf("nested summary file not written: %v", err)
	}
}

func TestPromptFile(t *testing.T) {
	s, dir := newStore(t, "md")
	if err := s.Prompt("fix", "claude-coder", "fix", 2, "the exact prompt text"); err != nil {
		t.Fatal(err)
	}
	files := roundFiles(t, dir, 2)
	got, ok := files["fix-claude-coder-fix.prompt"]
	if !ok {
		t.Fatalf("prompt file not written; files: %v", files)
	}
	if got != "the exact prompt text" {
		t.Errorf("prompt content = %q", got)
	}
}

// Secrets must be masked in every persisted format, not just raw: dropping the
// raw format must not leave a reviewed credential sitting in the md, json, or
// .prompt files. A regression removing the RedactSecrets calls in Step/Prompt
// would fail here.
func TestStepAndPromptRedactSecrets(t *testing.T) {
	s, dir := newStore(t, "md", "json")
	const secret = "sk-ant-api03-ABCDEFGHIJKLMNOPqrstuv"
	// An adversarial generic-assignment value that ENDS on a quote: once
	// marshaled, that trailing quote becomes the JSON escape \" -- redaction run
	// over the serialized bytes must mask the value without eating the escape
	// backslash and unbalancing the surrounding JSON string quoting.
	const quotedSecret = `password=supersecretvalue"`
	const title = "leak"
	// The secret rides in both the parsed JSON payload (a finding description a
	// reviewer quoted) and the human md rendering.
	parsed := model.ReviewOutput{Findings: []model.ReviewFinding{{
		Title:       title,
		Description: "found " + secret + " and " + quotedSecret,
	}}}
	md := "# Review\n\nfound " + secret + "\n"
	if err := s.Step("review", "claude", "review-bugs", 1, parsed, md, "$ raw"); err != nil {
		t.Fatal(err)
	}
	if err := s.Prompt("fix", "claude-coder", "fix", 1, "the prompt embeds "+secret); err != nil {
		t.Fatal(err)
	}
	files := roundFiles(t, dir, 1)
	for _, name := range []string{
		"review-claude-review-bugs.md",
		"review-claude-review-bugs.json",
		"fix-claude-coder-fix.prompt",
	} {
		content, ok := files[name]
		if !ok {
			t.Fatalf("%s not written; files: %v", name, files)
		}
		if strings.Contains(content, secret) {
			t.Errorf("%s still contains the secret:\n%s", name, content)
		}
		if strings.Contains(content, "supersecretvalue") {
			t.Errorf("%s still contains the quoted assignment secret:\n%s", name, content)
		}
		if !strings.Contains(content, "[REDACTED]") {
			t.Errorf("%s missing the [REDACTED] mask (elided instead of masked?):\n%s", name, content)
		}
	}
	// The redacted JSON must stay well-formed and round-trip: a regex that ate
	// the trailing \" would corrupt the escaping and fail this unmarshal, and a
	// pass that dropped unrelated fields would lose the title.
	var back model.ReviewOutput
	if err := json.Unmarshal([]byte(files["review-claude-review-bugs.json"]), &back); err != nil {
		t.Fatalf("redacted json no longer parses (escaping corrupted?): %v\n%s", err, files["review-claude-review-bugs.json"])
	}
	if len(back.Findings) != 1 || back.Findings[0].Title != title {
		t.Fatalf("redaction altered the non-secret payload: %+v", back)
	}
	if !strings.HasPrefix(back.Findings[0].Description, "found [REDACTED] and password=[REDACTED]") {
		t.Errorf("description not masked as expected: %q", back.Findings[0].Description)
	}
}

// The durable artifacts are read back in a terminal, so agent-authored text in
// them gets the same escaping the live output gets: a prompt-injected reviewer
// (or reviewed content quoted into the prompt) must not be able to drive the
// terminal of whoever `cat`s the log. Raw is exempt -- it is the fidelity record.
func TestStepAndPromptEscapeTerminalControls(t *testing.T) {
	s, dir := newStore(t, "md", "raw")
	// OSC 52 (clipboard write), a screen clear, and a right-to-left override.
	const rlo = "\u202e"
	attack := "\x1b]52;c;ZXZpbA==\x07\x1b[2Jinnocent" + rlo + "txt.eb"
	md := "# Review\n\n## [r1.1] " + attack + "\n\n\tindented snippet\n"
	if err := s.Step("review", "claude", "review-bugs", 1, nil, md, "$ raw "+attack); err != nil {
		t.Fatal(err)
	}
	if err := s.Prompt("fix", "claude-coder", "fix", 1, "the diff carries "+attack+"\nsecond line\n"); err != nil {
		t.Fatal(err)
	}
	files := roundFiles(t, dir, 1)
	for _, name := range []string{"review-claude-review-bugs.md", "fix-claude-coder-fix.prompt"} {
		content, ok := files[name]
		if !ok {
			t.Fatalf("%s not written; files: %v", name, files)
		}
		if strings.ContainsAny(content, "\x1b\x07") || strings.Contains(content, rlo) {
			t.Errorf("%s still carries terminal control sequences:\n%q", name, content)
		}
		// Escaped, not dropped: the text stays visible and greppable.
		if !strings.Contains(content, "\\x1b") || !strings.Contains(content, "\\u202e") {
			t.Errorf("%s dropped the offending text instead of escaping it:\n%q", name, content)
		}
		// Escaping must not flatten the document: line breaks are what keeps a
		// durable artifact readable.
		if !strings.Contains(content, "\n") {
			t.Errorf("%s was flattened onto one line:\n%q", name, content)
		}
	}
	if got := files["review-claude-review-bugs.raw"]; got != "$ raw "+attack {
		t.Errorf("raw log is the fidelity record and must keep its bytes: %q", got)
	}
	if !strings.Contains(files["review-claude-review-bugs.md"], "\n\tindented snippet\n") {
		t.Errorf("md lost the tab indentation:\n%q", files["review-claude-review-bugs.md"])
	}
}

func TestStepSkipsJSONWhenParseFailed(t *testing.T) {
	s, dir := newStore(t, "md", "json")
	if err := s.Step("review", "claude", "review-bugs", 1, nil, "md", "raw"); err != nil {
		t.Fatal(err)
	}
	files := roundFiles(t, dir, 1)
	if len(files) != 1 {
		t.Fatalf("expected only md file, got %v", files)
	}
}

// Two runs starting within the same timestamp interval must not share a run
// directory: the second claims a suffixed one instead of overwriting.
func TestRunDirCollision(t *testing.T) {
	s, dir := newStore(t, "md")
	want := s.runDir
	if err := os.MkdirAll(s.runDir, 0o700); err != nil { // an earlier run's dir
		t.Fatal(err)
	}
	if err := s.Step("review", "claude", "review-bugs", 1, nil, "md content", ""); err != nil {
		t.Fatal(err)
	}
	if s.runDir == want {
		t.Fatalf("run dir %s was not disambiguated from the existing one", s.runDir)
	}
	if _, err := os.Stat(filepath.Join(s.runDir, "round-1", "review-claude-review-bugs.md")); err != nil {
		t.Errorf("step file not in the disambiguated dir: %v", err)
	}
	runs, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 run dirs, got %d", len(runs))
	}
}

// Step is called from parallel reviewer goroutines; the run directory must be
// claimed exactly once, with every writer's files landing in it. Run under
// -race this also guards the sync.Once protecting runDir.
func TestStepConcurrent(t *testing.T) {
	s, dir := newStore(t, "md")
	// Pre-create the timestamped dir so the goroutines also race on picking
	// the suffixed replacement, not just on MkdirAll.
	if err := os.MkdirAll(s.runDir, 0o700); err != nil {
		t.Fatal(err)
	}

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = s.Step("review", fmt.Sprintf("agent%d", i), "lens", 1, nil, "md content", "")
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("Step %d: %v", i, err)
		}
	}

	runs, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 { // the seeded dir + exactly one claimed by the store
		t.Fatalf("expected 2 run dirs (seed + one claimed), got %d", len(runs))
	}
	entries, err := os.ReadDir(filepath.Join(s.runDir, "round-1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != n {
		t.Errorf("claimed round dir has %d files, want %d (all writers in one dir)", len(entries), n)
	}
}

func TestFilesAreOwnerOnly(t *testing.T) {
	s, dir := newStore(t, "raw")
	if err := s.Step("review", "claude", "review-bugs", 1, nil, "", "secret material"); err != nil {
		t.Fatal(err)
	}
	runDir := singleRunDir(t, dir)
	roundDir := filepath.Join(runDir, "round-1")
	for _, d := range []string{runDir, roundDir} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("dir %s mode = %o, want 700", d, perm)
		}
	}
	entries, err := os.ReadDir(roundDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no step files written")
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode = %o, want 600", e.Name(), perm)
		}
	}
}

func TestSummary(t *testing.T) {
	s, dir := newStore(t, "md", "json", "raw")
	sum := &model.RunSummary{
		Mode: "directory", Path: ".", Strategy: "rotate",
		Termination: model.TermConverged,
		Rounds: []model.RoundRecord{{
			Round:       1,
			Assignments: []model.Assignment{{Lens: "prompts/review-bugs.md", Agent: "claude"}},
			Findings: []model.Finding{{
				ID: "r1.1", Agent: "claude", Lens: "review-bugs",
				Category: "correctness", Severity: "high", File: "a.go", Line: 3,
				Title: "bug", Verdict: "fixed", VerdictDetail: "patched",
			}},
			Fixed:     1,
			CommitSHA: "0123456789abcdef",
			Steps: []model.StepStat{
				{Role: "review", Agent: "claude", Lens: "review-bugs", PromptBytes: 4096, OutputBytes: 2048, DurationMS: 1500},
				{Role: "fix", Agent: "coder", Lens: "fix", PromptBytes: 1024, OutputBytes: 0, DurationMS: 500, Failed: true},
			},
		}},
	}
	mdPath, err := s.Summary(sum)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(mdPath) != "summary.md" {
		t.Errorf("summary md path = %s", mdPath)
	}
	files := filesIn(t, singleRunDir(t, dir))
	md, ok := files["summary.md"]
	if !ok {
		t.Fatalf("summary.md not written; files: %v", files)
	}
	for _, want := range []string{
		"termination: **converged**",
		"review-bugs→claude",
		"**FIXED** [r1.1] (correctness, high) a.go:3 — bug (by claude/review-bugs)",
		"Committed: `0123456789ab`",
		// ioTotals summary line and the per-step section (r1.9).
		"agent I/O: 2 invocation(s)",
		"Steps:",
		"review claude/review-bugs:",
		"fix coder/fix:",
		"**FAILED**",
		// SizeDesc token approximation: 4096 bytes -> 4.0 KB, ~1.0k tokens.
		"4.0 KB ≈ 1.0k tok",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("summary md missing %q:\n%s", want, md)
		}
	}
	var back model.RunSummary
	if err := json.Unmarshal([]byte(files["summary.json"]), &back); err != nil {
		t.Fatalf("summary json: %v", err)
	}
	if back.Termination != model.TermConverged || len(back.Rounds) != 1 {
		t.Errorf("summary json roundtrip = %+v", back)
	}
}

// renderSummaryMD has branches that TestSummary's converged-with-one-fix round
// never exercises: the salvaged-coder-error banner, the reviewer-errors list,
// the "no non-advisory findings" line, and the advisory-findings section. A
// round that sets all four at once asserts each header renders, so a regression
// dropping or mislabeling one is caught.
func TestSummaryRendersAllRoundSections(t *testing.T) {
	s, dir := newStore(t, "md")
	sum := &model.RunSummary{
		Mode: "directory", Path: ".", Strategy: "rotate",
		Termination: model.TermError,
		Rounds: []model.RoundRecord{{
			Round:        1,
			Assignments:  []model.Assignment{{Lens: "prompts/review-bugs.md", Agent: "claude"}},
			CoderError:   "timed out after 10m",
			ReviewErrors: []string{"codex via review-security: no <review> block"},
			Advisory:     []model.Finding{{Severity: "low", Title: "consider a doc comment"}},
			// No non-advisory Findings -- exercises the empty-findings branch.
		}},
	}
	if _, err := s.Summary(sum); err != nil {
		t.Fatal(err)
	}
	md := filesIn(t, singleRunDir(t, dir))["summary.md"]
	for _, want := range []string{
		"**Coder failed (partial work salvaged):** timed out after 10m",
		"**Review errors:**",
		"- codex via review-security: no <review> block",
		"No non-advisory findings.",
		"Advisory findings (1, report only):",
		"- (low) consider a doc comment — see review log for detail",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("summary md missing %q:\n%s", want, md)
		}
	}
}

// When the closing round fails, the run's termination becomes "error" and the
// loop's own outcome is preserved separately -- the difference between a run that
// never converged and one that converged and then tripped on its last step. The
// summary markdown is the durable artifact a human reads to learn that, so the
// line has to be rendered, and it must be absent when there is nothing to
// distinguish (LoopTermination empty) rather than printing an empty outcome.
func TestSummaryRendersTheLoopTerminationWhenTheClosingRoundFailed(t *testing.T) {
	render := func(t *testing.T, loopTerm string) string {
		t.Helper()
		s, dir := newStore(t, "md")
		sum := &model.RunSummary{
			Termination:     model.TermError,
			LoopTermination: loopTerm,
			Error:           "closing round 3: coder failed",
			Rounds:          []model.RoundRecord{{Round: 1}},
		}
		if _, err := s.Summary(sum); err != nil {
			t.Fatal(err)
		}
		return filesIn(t, singleRunDir(t, dir))["summary.md"]
	}

	md := render(t, model.TermConverged)
	if !strings.Contains(md, "termination: **error**") {
		t.Errorf("summary md missing the run's termination:\n%s", md)
	}
	if !strings.Contains(md, "loop termination (before the closing round failed): converged") {
		t.Errorf("summary md missing the loop's own outcome:\n%s", md)
	}

	// A run whose loop outcome IS the run outcome has nothing extra to say.
	if md := render(t, ""); strings.Contains(md, "loop termination") {
		t.Errorf("summary md reports a loop termination that was never recorded:\n%s", md)
	}
}

// The closing round can also be INTERRUPTED rather than fail, which preserves the
// loop's outcome the same way. The line has to say which of the two happened: a
// summary telling the operator their closing round "failed" when they stopped it
// themselves sends them looking for a failure that never occurred.
func TestSummaryNamesAnInterruptedClosingRound(t *testing.T) {
	s, dir := newStore(t, "md")
	sum := &model.RunSummary{
		Termination:     model.TermInterrupted,
		LoopTermination: model.TermConverged,
		Rounds:          []model.RoundRecord{{Round: 1}},
	}
	if _, err := s.Summary(sum); err != nil {
		t.Fatal(err)
	}
	md := filesIn(t, singleRunDir(t, dir))["summary.md"]
	if !strings.Contains(md, "loop termination (before the closing round was interrupted): converged") {
		t.Errorf("summary md does not report the closing round as interrupted:\n%s", md)
	}
}

// The run summary is a durable artifact that embeds finding descriptions and
// coder verdict details -- both derived from agent output that may quote a
// discovered secret. It must be redacted like the per-step logs, in BOTH the
// md and json renderings, and the redacted json must stay parseable.
func TestSummaryRedactsSecrets(t *testing.T) {
	s, dir := newStore(t, "md", "json")
	const secret = "sk-ant-api03-ABCDEFGHIJKLMNOPqrstuv"
	// A verdict detail that ends on a quote, so the serialized JSON escapes it as
	// \": redaction over the marshaled bytes must not corrupt that escaping.
	const quotedSecret = `password=supersecretvalue"`
	sum := &model.RunSummary{
		Termination: model.TermConverged,
		Rounds: []model.RoundRecord{{
			Round: 1,
			Findings: []model.Finding{{
				ID: "r1.1", Title: "leak", Severity: "high", File: "a.go", Line: 3,
				Description:   "reviewer quoted " + secret,
				Verdict:       "fixed",
				VerdictDetail: "coder noted " + quotedSecret,
			}},
			Fixed: 1,
		}},
	}
	if _, err := s.Summary(sum); err != nil {
		t.Fatal(err)
	}
	files := filesIn(t, singleRunDir(t, dir))
	for _, name := range []string{"summary.md", "summary.json"} {
		content, ok := files[name]
		if !ok {
			t.Fatalf("%s not written; files: %v", name, files)
		}
		if strings.Contains(content, secret) || strings.Contains(content, "supersecretvalue") {
			t.Errorf("%s still contains a secret:\n%s", name, content)
		}
		if !strings.Contains(content, "[REDACTED]") {
			t.Errorf("%s missing the [REDACTED] mask:\n%s", name, content)
		}
	}
	// The redacted json summary must remain valid and round-trip.
	var back model.RunSummary
	if err := json.Unmarshal([]byte(files["summary.json"]), &back); err != nil {
		t.Fatalf("redacted summary.json no longer parses: %v\n%s", err, files["summary.json"])
	}
	if len(back.Rounds) != 1 || len(back.Rounds[0].Findings) != 1 || back.Rounds[0].Findings[0].Title != "leak" {
		t.Fatalf("redaction altered the non-secret summary payload: %+v", back)
	}
}

// The summary is the one artifact an operator is certain to open, and it carries
// agent-authored titles and verdict details forward, so it is escaped like the
// per-step md.
func TestSummaryEscapesTerminalControls(t *testing.T) {
	s, dir := newStore(t, "md")
	const attack = "\x1b[2Jcleared"
	sum := &model.RunSummary{
		Termination: model.TermConverged,
		Rounds: []model.RoundRecord{{
			Round: 1,
			Findings: []model.Finding{{
				ID: "r1.1", Title: attack, Severity: "high", File: "a.go", Line: 3,
				Verdict: "fixed", VerdictDetail: attack,
			}},
			Fixed: 1,
		}},
	}
	if _, err := s.Summary(sum); err != nil {
		t.Fatal(err)
	}
	md := filesIn(t, singleRunDir(t, dir))["summary.md"]
	if strings.Contains(md, "\x1b") {
		t.Errorf("summary.md still carries an escape sequence:\n%q", md)
	}
	if !strings.Contains(md, "\\x1b[2Jcleared") {
		t.Errorf("summary.md dropped the offending title instead of escaping it:\n%s", md)
	}
	// The scoreboard's box drawing and the markdown structure must survive.
	if !strings.Contains(md, "# fixpoint run summary\n") || !strings.Contains(md, "\u2500") {
		t.Errorf("summary.md lost its structure:\n%s", md)
	}
}

// renderSummaryMD abbreviates the commit SHA to 12 chars. A non-empty SHA
// shorter than 12 chars (a partial/abbreviated value from a failing or partial
// git call) must render verbatim rather than panic the summary write on an
// out-of-range slice.
func TestSummaryShortCommitSHA(t *testing.T) {
	s, dir := newStore(t, "md")
	sum := &model.RunSummary{
		Termination: model.TermConverged,
		Rounds: []model.RoundRecord{{
			Round:     1,
			CommitSHA: "abc",
			Fixed:     1,
		}},
	}
	if _, err := s.Summary(sum); err != nil {
		t.Fatalf("Summary with a short SHA: %v", err)
	}
	md := filesIn(t, singleRunDir(t, dir))["summary.md"]
	if !strings.Contains(md, "Committed: `abc`") {
		t.Errorf("short SHA not rendered verbatim:\n%s", md)
	}
}

func TestSizeDesc(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0.0 KB ≈ 0.0k tok"},
		{4096, "4.0 KB ≈ 1.0k tok"},
		{1024, "1.0 KB ≈ 0.3k tok"},
	}
	for _, tc := range cases {
		if got := SizeDesc(tc.n); got != tc.want {
			t.Errorf("SizeDesc(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestRenderReviewMD(t *testing.T) {
	findings := []model.Finding{{
		ID: "r1.1", Category: "bugs", Severity: "high", File: "a.go", Line: 7,
		Title: "leak", Description: "goroutine leaks", Suggestion: "close it",
	}}
	md := RenderReviewMD("claude", "review-bugs", 2, findings, nil)
	for _, want := range []string{
		"# Review — claude via review-bugs (round 2)",
		"## [r1.1] (bugs, high) a.go:7 — leak",
		"goroutine leaks",
		"**Suggested:** close it",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("review md missing %q:\n%s", want, md)
		}
	}
	empty := RenderReviewMD("claude", "review-bugs", 1, nil, nil)
	if !strings.Contains(empty, "No findings.") {
		t.Errorf("empty review md = %q", empty)
	}
	withErr := RenderReviewMD("claude", "review-bugs", 1, nil, os.ErrDeadlineExceeded)
	if !strings.Contains(withErr, "**ERROR:**") {
		t.Errorf("error review md = %q", withErr)
	}
}

func TestRenderFixMD(t *testing.T) {
	findings := []model.Finding{
		{ID: "r1.1", Title: "bug", Verdict: "fixed", VerdictDetail: "patched"},
		{ID: "r1.2", Title: "nit"},
	}
	md := RenderFixMD("coder", 1, findings, "note text", nil)
	for _, want := range []string{
		"# Fix — coder (round 1)",
		"- **FIXED** [r1.1] bug — patched",
		"- **UNRESOLVED** [r1.2] nit —",
		"**Notes:** note text",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("fix md missing %q:\n%s", want, md)
		}
	}
}

// A non-default logs.dir template must be honored end to end: the round
// artifacts land where the template says, and the summary lands in the run dir
// (the part before the first {round} segment) rather than in a round directory.
func TestCustomDirTemplateLayout(t *testing.T) {
	base := filepath.Join(t.TempDir(), "artifacts")
	s, err := New(config.Logs{
		Dir:             filepath.Join(base, "run-{timestamp}", "rounds", "{round}"),
		Formats:         []string{"md"},
		Pattern:         "{role}-{agent}-{prompt}.{ext}",
		SummaryPattern:  "summary.{ext}",
		TimestampFormat: "20060102-150405",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, round := range []int{1, 2} {
		if err := s.Step("review", "claude", "review-bugs", round, nil, "# md", ""); err != nil {
			t.Fatalf("Step round %d: %v", round, err)
		}
	}
	if _, err := s.Summary(&model.RunSummary{Termination: model.TermConverged}); err != nil {
		t.Fatalf("Summary: %v", err)
	}
	run := singleRunDir(t, base)
	for _, round := range []string{"1", "2"} {
		p := filepath.Join(run, "rounds", round, "review-claude-review-bugs.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("round %s artifact missing at %s: %v", round, p, err)
		}
	}
	// The summary spans all rounds, so it belongs to the run dir, not a round dir.
	if _, err := os.Stat(filepath.Join(run, "summary.md")); err != nil {
		t.Errorf("summary not at the run root: %v", err)
	}
}
