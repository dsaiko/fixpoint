// Package logstore writes the per-step review/fix logs and the run summary,
// naming files according to the configured pattern.
package logstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// Store writes one run's artifacts: per-step logs grouped by round, and the
// run summary at the run root.
type Store struct {
	cfg config.Logs
	// runTS is the run's start time. It is captured once so the {timestamp} in
	// the logs.dir template has one value for the whole run, keeping every
	// round's artifacts under a single run directory. The {timestamp} in
	// logs.pattern is per-invocation and is not this value.
	runTS  time.Time
	runDir string // logs.dir rendered up to its {round} segment; claimed lazily

	mkdir    sync.Once // Step runs from parallel reviewer goroutines
	mkdirErr error

	// The journal is append-only and its sequence numbers must be gap-free, so
	// concurrent appends serialize here rather than relying on O_APPEND ordering.
	journalMu  sync.Mutex
	journalSeq int
}

// New prepares a store for one run by rendering the run-level part of the
// logs.dir template, so consecutive runs never mix.
func New(cfg config.Logs) (*Store, error) {
	ts := time.Now()
	return &Store{
		cfg:    cfg,
		runTS:  ts,
		runDir: cfg.RunPath(ts.Format(cfg.TimestampFormat)),
	}, nil
}

// Logs carry the reviewed source, raw agent output, and unfixed findings, so
// the run directory and every file in it are owner-only.
//
// The directory is claimed atomically (os.Mkdir) with a numeric suffix on
// collision, so two runs started within the same timestamp interval never
// share a directory and overwrite each other's artifacts.
func (s *Store) ensureDir() error {
	s.mkdir.Do(func() {
		if s.mkdirErr = os.MkdirAll(filepath.Dir(s.runDir), 0o700); s.mkdirErr != nil {
			return
		}
		base := s.runDir
		for i := 1; ; i++ {
			err := os.Mkdir(s.runDir, 0o700)
			if err == nil {
				return
			}
			if !os.IsExist(err) || i >= 1000 {
				s.mkdirErr = err
				return
			}
			s.runDir = fmt.Sprintf("%s-%d", base, i+1)
		}
	})
	return s.mkdirErr
}

// roundDir returns (creating it, owner-only) the round's artifact directory as
// the logs.dir template renders it beneath the claimed run dir, so every
// artifact of a round is grouped together. A template without {round} puts every
// round in the run dir itself (logs.pattern then carries {round} -- config
// validation enforces one or the other). Safe to call concurrently from the
// parallel reviewer goroutines of a round.
func (s *Store) roundDir(round int) (string, error) {
	if err := s.ensureDir(); err != nil {
		return "", err
	}
	dir := s.cfg.RoundPath(s.runDir, round, s.runTS.Format(s.cfg.TimestampFormat))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// writeArtifact writes one log file, first creating its parent directory
// (owner-only) so a logs.pattern carrying directory components (config
// validation accepts e.g. "{role}/{agent}/{prompt}-{round}.{ext}") does not
// fail every write with ENOENT on the not-yet-existing subdirectory. Every
// artifact write (Step, Prompt, Summary) goes through here so the mkdir and the
// 0600 permission stay in one place.
func writeArtifact(name string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return err
	}
	return os.WriteFile(name, content, 0o600)
}

// stepFilename renders a per-step log path from logs.pattern, filling the
// role/agent/prompt/round of one agent invocation. The substitution lives on
// config.Logs (StepPath) so this and the config-side collision validator share
// one renderer and cannot disagree about the path a step actually lands on.
func (s *Store) stepFilename(dir, role, agentName, promptName string, round int, ext string, ts time.Time) string {
	return filepath.Join(dir, s.cfg.StepPath(role, agentName, promptName, round, ext, ts.Format(s.cfg.TimestampFormat)))
}

// summaryFilename renders the per-run summary path from logs.summary_pattern.
// The summary spans all rounds and has no single role/agent/prompt/round, so it
// fills only the placeholders that apply to a whole run.
func (s *Store) summaryFilename(ext string, ts time.Time) string {
	r := strings.NewReplacer(
		"{timestamp}", ts.Format(s.cfg.TimestampFormat),
		"{ext}", ext,
	)
	return filepath.Join(s.runDir, r.Replace(s.cfg.SummaryPattern))
}

// Step writes one agent invocation's logs in every configured format.
// parsed is the extracted JSON payload (or nil if extraction failed), md the
// human rendering, raw the full process output.
func (s *Store) Step(role, agentName, promptName string, round int, parsed any, md, raw string) error {
	dir, err := s.roundDir(round)
	if err != nil {
		return err
	}
	ts := time.Now()
	for _, ext := range s.cfg.Formats {
		var content []byte
		switch ext {
		case "json":
			if parsed == nil {
				continue
			}
			b, err := json.MarshalIndent(parsed, "", "  ")
			if err != nil {
				return err
			}
			// A reviewer may quote a discovered secret into a finding, so redact
			// the parsed payload too. Masking keeps the JSON well-formed (a value
			// becomes "[REDACTED]").
			content = []byte(agent.RedactSecrets(string(b)))
		case "md":
			content = []byte(agent.RedactSecrets(md))
		case "raw":
			content = []byte(raw) // Result.Raw already redacted this
		}
		name := s.stepFilename(dir, role, agentName, promptName, round, ext, ts)
		if err := writeArtifact(name, content); err != nil {
			return err
		}
	}
	return nil
}

// Prompt persists the exact prompt text sent to an agent, written at
// invocation START (not completion) so a long-running or killed agent's input
// is inspectable mid-run and survives crashes. Extension: .prompt.
func (s *Store) Prompt(role, agentName, promptName string, round int, text string) error {
	dir, err := s.roundDir(round)
	if err != nil {
		return err
	}
	name := s.stepFilename(dir, role, agentName, promptName, round, "prompt", time.Now())
	// The prompt embeds the collected material (in git-diff/pr mode, the diff),
	// which very commonly contains the exact secret under review (a commit
	// adding an API key or .env). Redact before this always-written file lands
	// on disk -- dropping `raw` must not leave secrets persisted elsewhere.
	return writeArtifact(name, []byte(agent.RedactSecrets(text)))
}

// Summary writes the run summary as md + json (raw does not apply).
func (s *Store) Summary(sum *model.RunSummary) (string, error) {
	if err := s.ensureDir(); err != nil {
		return "", err
	}
	ts := time.Now()
	var mdPath string
	for _, ext := range []string{"md", "json"} {
		// The summary spans all rounds, so it lives at the run root, not in
		// any round-N subdirectory.
		name := s.summaryFilename(ext, ts)
		var content []byte
		if ext == "json" {
			b, err := json.MarshalIndent(sum, "", "  ")
			if err != nil {
				return "", err
			}
			// The summary embeds finding titles, descriptions, suggestions, and
			// verdict details -- all derived from agent output, which may quote a
			// discovered secret. Redact like Step/Prompt so the durable summary
			// files never persist a credential the per-step logs already masked.
			// (The generic rule excludes the JSON escape backslash, so masking a
			// value here cannot break the serialized JSON.)
			content = []byte(agent.RedactSecrets(string(b)))
		} else {
			content = []byte(agent.RedactSecrets(renderSummaryMD(sum)))
			mdPath = name
		}
		if err := writeArtifact(name, content); err != nil {
			return "", err
		}
	}
	return mdPath, nil
}

// RenderReviewMD renders one reviewer invocation for the .md log.
func RenderReviewMD(agent, lens string, round int, findings []model.Finding, runErr error) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Review — %s via %s (round %d)\n\n", agent, lens, round)
	if runErr != nil {
		fmt.Fprintf(&sb, "**ERROR:** %v\n\n", runErr)
	}
	if len(findings) == 0 {
		sb.WriteString("No findings.\n")
		return sb.String()
	}
	for _, f := range findings {
		fmt.Fprintf(&sb, "## [%s] (%s, %s) %s — %s\n\n", f.ID, f.Category, f.Severity, f.Loc(), f.Title)
		if f.Description != "" {
			sb.WriteString(f.Description + "\n\n")
		}
		if f.Suggestion != "" {
			sb.WriteString("**Suggested:** " + f.Suggestion + "\n\n")
		}
	}
	return sb.String()
}

// RenderFixMD renders the coder invocation for the .md log.
func RenderFixMD(agent string, round int, findings []model.Finding, notes string, runErr error) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Fix — %s (round %d)\n\n", agent, round)
	if runErr != nil {
		fmt.Fprintf(&sb, "**ERROR:** %v\n\n", runErr)
	}
	for _, f := range findings {
		fmt.Fprintf(&sb, "- **%s** [%s] %s — %s\n", strings.ToUpper(f.VerdictOrDefault()), f.ID, f.Title, f.VerdictDetail)
	}
	if notes != "" {
		sb.WriteString("\n**Notes:** " + notes + "\n")
	}
	return sb.String()
}

//nolint:gocognit // linear rendering of the summary sections; splitting it apart would not simplify it
func renderSummaryMD(sum *model.RunSummary) string {
	var sb strings.Builder
	sb.WriteString("# fixpoint run summary\n\n")
	// The scoreboard first, fenced so its column alignment survives: it is the same
	// text the run prints when it ends, and the part anyone reads before the detail.
	sb.WriteString("```\n" + RenderRunTable(sum) + "```\n\n")
	fmt.Fprintf(&sb, "- started: %s\n", sum.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&sb, "- finished: %s\n", sum.FinishedAt.Format(time.RFC3339))
	fmt.Fprintf(&sb, "- target: %s (%s)\n", sum.Path, sum.Mode)
	fmt.Fprintf(&sb, "- strategy: %s, review_only: %v\n", sum.Strategy, sum.ReviewOnly)
	fmt.Fprintf(&sb, "- rounds: %d\n", len(sum.Rounds))
	fmt.Fprintf(&sb, "- termination: **%s**\n", sum.Termination)
	if sum.LoopTermination != "" {
		fmt.Fprintf(&sb, "- loop termination (before the closing round failed): %s\n", sum.LoopTermination)
	}
	if sum.Error != "" {
		fmt.Fprintf(&sb, "- error: %s\n", sum.Error)
	}
	renderSources(&sb, sum.Sources)
	if steps, promptB, outB := ioTotals(sum); steps > 0 {
		fmt.Fprintf(&sb, "- agent I/O: %d invocation(s), prompts %s, outputs %s\n",
			steps, SizeDesc(promptB), SizeDesc(outB))
	}
	sb.WriteString("\n")
	for _, r := range sum.Rounds {
		fmt.Fprintf(&sb, "## Round %d\n\n", r.Round)
		sb.WriteString("Assignments: ")
		var parts []string
		for _, a := range r.Assignments {
			lens := config.LensName(a.Lens)
			p := fmt.Sprintf("%s→%s", lens, a.Agent)
			if a.Advisory {
				p += " (advisory)"
			}
			parts = append(parts, p)
		}
		sb.WriteString(strings.Join(parts, ", ") + "\n\n")
		if r.CoderError != "" {
			fmt.Fprintf(&sb, "**Coder failed (partial work salvaged):** %s\n\n", r.CoderError)
		}
		if len(r.ReviewErrors) > 0 {
			sb.WriteString("**Review errors:**\n")
			for _, e := range r.ReviewErrors {
				sb.WriteString("- " + e + "\n")
			}
			sb.WriteString("\n")
		}
		if len(r.Findings) == 0 {
			sb.WriteString("No non-advisory findings.\n\n")
		} else {
			fmt.Fprintf(&sb, "Findings (%d) — %d fixed, %d rejected:\n\n", len(r.Findings), r.Fixed, r.Rejected)
			for _, f := range r.Findings {
				fmt.Fprintf(&sb, "- **%s** [%s] (%s, %s) %s — %s (by %s/%s)\n", strings.ToUpper(f.VerdictOrDefault()), f.ID, f.Category, f.Severity, f.Loc(), f.Title, f.Agent, f.Lens)
				if f.VerdictDetail != "" {
					fmt.Fprintf(&sb, "  - %s\n", f.VerdictDetail)
				}
			}
			sb.WriteString("\n")
		}
		if len(r.Advisory) > 0 {
			fmt.Fprintf(&sb, "Advisory findings (%d, report only):\n\n", len(r.Advisory))
			for _, f := range r.Advisory {
				fmt.Fprintf(&sb, "- (%s) %s — see review log for detail\n", f.Severity, f.Title)
			}
			sb.WriteString("\n")
		}
		renderIssues(&sb, r)
		renderVerify(&sb, r)
		if len(r.Steps) > 0 {
			sb.WriteString("Steps:\n\n")
			for _, st := range r.Steps {
				status := ""
				if st.Failed {
					status = " **FAILED**"
				}
				fmt.Fprintf(&sb, "- %s %s/%s: prompt %s, output %s, %s%s\n",
					st.Role, st.Agent, st.Lens,
					SizeDesc(st.PromptBytes), SizeDesc(st.OutputBytes),
					(time.Duration(st.DurationMS) * time.Millisecond).Round(time.Second), status)
			}
			sb.WriteString("\n")
		}
		if r.CommitSHA != "" {
			// Show the abbreviated SHA, but never slice past the end: a partial or
			// failing git call can record a SHA shorter than 12 chars, and slicing
			// it unconditionally would panic the summary write and take down the run.
			sha := r.CommitSHA
			if len(sha) > 12 {
				sha = sha[:12]
			}
			fmt.Fprintf(&sb, "Committed: `%s`\n\n", sha)
		}
	}
	return sb.String()
}

// SizeDesc renders a byte count with its approximate token equivalent
// (~4 bytes/token for English/code): "12.1 KB ≈ 3.0k tok".
func SizeDesc(n int) string {
	return fmt.Sprintf("%.1f KB ≈ %.1fk tok", float64(n)/1024, float64(n)/4/1000)
}

// ioTotals sums the per-step I/O figures across every round of the run.
func ioTotals(sum *model.RunSummary) (steps, promptBytes, outputBytes int) {
	for _, r := range sum.Rounds {
		for _, st := range r.Steps {
			steps++
			promptBytes += st.PromptBytes
			outputBytes += st.OutputBytes
		}
	}
	return steps, promptBytes, outputBytes
}

// renderSources records which file every bundle name resolved to. Prompts steer
// agents that edit code and the config carries the trust gates, so reading a
// summary months later must not leave open the question of whether a local file
// shadowed the installed one.
// renderIssues reports the deduplicated view the coder actually worked from, and
// -- more usefully when reading a run back -- which issues several agents found
// independently. Corroboration is the strongest evidence a panel produces, and it
// was previously invisible: two agents agreeing looked like two unrelated findings.
func renderIssues(sb *strings.Builder, rec model.RoundRecord) {
	if len(rec.Issues) == 0 {
		return
	}
	corroborated := 0
	for _, it := range rec.Issues {
		if len(it.Agents()) > 1 {
			corroborated++
		}
	}
	fmt.Fprintf(sb, "\nIssues (%d distinct from %d observation(s)", len(rec.Issues), len(rec.Findings))
	if corroborated > 0 {
		fmt.Fprintf(sb, "; %d corroborated by more than one agent", corroborated)
	}
	sb.WriteString("):\n")
	for _, it := range rec.Issues {
		fmt.Fprintf(sb, "- **%s** [%s] (%s, %s) %s — %s",
			strings.ToUpper(it.StatusOrDefault()), it.ID, it.Category, it.Severity, it.Loc(), it.Title)
		if agents := it.Agents(); len(agents) > 1 {
			fmt.Fprintf(sb, " _(reported by %s)_", strings.Join(agents, ", "))
		}
		sb.WriteString("\n")
		if it.VerdictDetail != "" {
			fmt.Fprintf(sb, "  - %s\n", it.VerdictDetail)
		}
	}
}

// renderVerify records the deterministic gate's outcome for a round. It is the
// only evidence in the summary that is not a model's opinion, so it is reported
// per round rather than folded into a total.
func renderVerify(sb *strings.Builder, rec model.RoundRecord) {
	if len(rec.Verify) == 0 {
		return
	}
	sb.WriteString("\nVerification")
	if rec.VerifyRetried {
		sb.WriteString(" (after one coder correction attempt)")
	}
	sb.WriteString(":\n")
	for _, v := range rec.Verify {
		status := "PASS"
		switch {
		case v.Err != "":
			status = "ERROR: " + v.Err
		case !v.Passed:
			status = fmt.Sprintf("FAIL (exit %d)", v.ExitCode)
		}
		if v.Optional {
			status += " [optional]"
		}
		fmt.Fprintf(sb, "- %s: %s — `%s` in %s\n", v.Name, status, strings.Join(v.Argv, " "), v.Duration.Round(time.Millisecond))
	}
}

func renderSources(sb *strings.Builder, src model.RunSources) {
	if src.Config == "" {
		return
	}
	sb.WriteString("\n## Configuration sources\n")
	fmt.Fprintf(sb, "- config: `%s`\n", src.Config)
	if src.Extends != "" {
		fmt.Fprintf(sb, "- extends: `%s`\n", src.Extends)
	}
	for _, label := range []struct {
		name string
		m    map[string]string
	}{{"agent", src.Agents}, {"prompt", src.Prompts}} {
		names := make([]string, 0, len(label.m))
		for k := range label.m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(sb, "- %s %s: `%s`\n", label.name, n, label.m[n])
		}
	}
}
