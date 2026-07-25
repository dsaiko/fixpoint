// Package config loads and validates fixpoint.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the whole of fixpoint.yaml: what to review, who reviews and
// fixes, how the loop terminates, and where logs go.
type Config struct {
	// Extends names another task config in the bundle whose values this one
	// inherits; keys set here override it. One level deep -- see loadWithExtends.
	Extends string `yaml:"extends"`

	// Description is a one-line summary shown by `fixpoint --list` and by shell
	// completion. Without it a list of bare names tells a new user nothing about
	// which config is safe to run.
	Description string `yaml:"description"`

	Target Target           `yaml:"target"`
	Roles  Roles            `yaml:"roles"`
	Agents map[string]Agent `yaml:"agents"`
	Loop   Loop             `yaml:"loop"`
	Logs   Logs             `yaml:"logs"`
	Verify Verify           `yaml:"verify"`

	// PingAgents: before a run, invoke every agent used by the run with a
	// trivial prompt (in parallel) and abort if any fails. Catches expired
	// logins and broken CLIs before tokens are spent. Default true.
	PingAgents *bool `yaml:"ping_agents"`
}

// Ping reports whether the preflight agent ping is enabled.
func (c *Config) Ping() bool { return c.PingAgents == nil || *c.PingAgents }

// Mode is how a round's review material is collected. The named type and its
// constants keep the literals in one place so every switch is compiler-checked.
type Mode string

// The three collection modes; see the target section of fixpoint.yaml.
const (
	ModeGitDiff   Mode = "git-diff"
	ModePR        Mode = "pr"
	ModeDirectory Mode = "directory"
)

// Strategy is how unpinned review lenses are assigned to the agent pool.
type Strategy string

// The three assignment strategies; see roles.review in fixpoint.yaml.
const (
	StrategyFixed  Strategy = "fixed"
	StrategyRotate Strategy = "rotate"
	StrategyAll    Strategy = "all"
)

// Target selects what a review round runs against.
type Target struct {
	Mode    Mode   `yaml:"mode"` // git-diff | pr | directory
	Path    string `yaml:"path"`
	BaseRef string `yaml:"base_ref"`
	PR      int    `yaml:"pr"`
	// Exclude removes paths from directory-mode collection. There is
	// deliberately no include/allowlist counterpart: an allowlist has to be
	// re-derived for every language and silently drops whatever it forgets,
	// so scope is denylist-only.
	Exclude []string `yaml:"exclude"`
}

// Roles binds the two roles of the loop -- the fixing coder and the review
// panel -- to agents and prompts.
type Roles struct {
	Coder  RoleRef `yaml:"coder"`
	Review Review  `yaml:"review"`
}

// RoleRef points one role at an agent and a prompt, both by BARE NAME:
// `agent: claude-coder`, `prompt: fix`. Names, not paths, so the same config
// works wherever its bundle was found and so a name is a stable identity in logs
// (a path would differ per machine). PromptPath holds what the name resolved to.
type RoleRef struct {
	Agent  string `yaml:"agent"`
	Prompt string `yaml:"prompt"`
	// PromptPath is filled in by the loader, never by YAML.
	PromptPath string `yaml:"-"`
}

// PromptFile returns the file to read the prompt from: the path the loader
// resolved, falling back to the name itself. The fallback lets a Config built
// directly in code (unit tests) name a path and still work, with no bundle
// resolver in the loop.
func (r RoleRef) PromptFile() string {
	if r.PromptPath != "" {
		return r.PromptPath
	}
	return r.Prompt
}

// Review is the roles.review section: the lens list, the agent pool, and the
// strategy that assigns one to the other each round.
type Review struct {
	Strategy Strategy     `yaml:"strategy"` // fixed | rotate | all
	Agents   []string     `yaml:"agents"`
	Prompts  []ReviewLens `yaml:"prompts"`
}

// ActiveAgents returns the distinct reviewer agents the configured strategy
// will actually use: every pinned lens agent, plus the shared pool for
// rotate/all (fixed assigns only pinned agents, so its pool is inert).
// Validation and the preflight ping both derive their agent set from here so
// an unused pool entry can never block a run.
func (r Review) ActiveAgents() []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, l := range r.Prompts {
		add(l.Agent)
	}
	if r.Strategy != StrategyFixed {
		for _, a := range r.Agents {
			add(a)
		}
	}
	return out
}

// LensAgents returns every agent a review lens can be assigned to across rounds
// and strategies: its pinned agent if it has one, otherwise the whole shared
// pool (rotate picks one per round, all uses them together -- the union is the
// same set). Both the orchestrator's per-round assignment fan-out and the
// log-collision validator's identity enumeration derive from this one policy, so
// they cannot disagree about who may write a lens's artifacts.
func (r Review) LensAgents(l ReviewLens) []string {
	if l.Agent != "" {
		return []string{l.Agent}
	}
	return r.Agents
}

// ReviewLens is one entry under roles.review.prompts: either a bare string
// (unpinned prompt path) or a map {agent, prompt, advisory}.
type ReviewLens struct {
	Agent    string `yaml:"agent"`
	Prompt   string `yaml:"prompt"`
	Advisory bool   `yaml:"advisory"`

	// Once runs the lens in round 1 only. Made for advisory lenses whose
	// subject barely changes between fix rounds (architecture/design): one
	// report per run instead of one per round.
	Once bool `yaml:"once"`

	// PromptPath is filled in by the loader, never by YAML.
	PromptPath string `yaml:"-"`
}

// PromptFile returns the file to read the lens's prompt from; see
// RoleRef.PromptFile for why the name is a fallback.
func (l ReviewLens) PromptFile() string {
	if l.PromptPath != "" {
		return l.PromptPath
	}
	return l.Prompt
}

// UnmarshalYAML accepts both lens spellings: a bare string (unpinned prompt
// path) and the map form.
func (l *ReviewLens) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&l.Prompt)
	}
	// KnownFields does not propagate into custom unmarshalers, so reject
	// unknown keys ourselves.
	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "agent", "prompt", "advisory", "once":
		default:
			return fmt.Errorf("line %d: unknown review lens field %q", node.Content[i].Line, node.Content[i].Value)
		}
	}
	type plain ReviewLens
	return node.Decode((*plain)(l))
}

// LensName derives the lens identity used in logs from a prompt path: the file
// basename without extension. Every caller (orchestrator, logstore, Validate)
// has a prompt path in hand, so this takes one rather than a whole ReviewLens.
func LensName(promptPath string) string {
	base := filepath.Base(promptPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// How a prompt reaches an agent CLI. The set is closed, so it gets named
// constants like Mode and Strategy: three packages compare against these values,
// and a single home keeps their spellings from drifting.
const (
	PromptViaStdin = "stdin"
	PromptViaArg   = "arg"
)

// Agent is one provider-agnostic command template from the agents section:
// a CLI invocation that receives a prompt and prints text to stdout.
type Agent struct {
	Model   string   `yaml:"model"`
	Effort  string   `yaml:"effort"`
	Command []string `yaml:"command"`
	// PromptVia is how the prompt reaches the CLI: "stdin" (default) or "arg".
	// SECURITY: "arg" appends the whole prompt to the process argument list,
	// which is world-readable to any local user (ps, /proc/<pid>/cmdline) for
	// the invocation's lifetime. The prompt embeds the collected review
	// material -- in git-diff/pr mode the diff, which often IS the credential
	// under review -- plus any secret a reviewer quotes into a finding, so arg
	// mode bypasses the 0600 log permissions and the on-disk redaction pass.
	// Prefer "stdin" for any material that may contain secrets.
	PromptVia string   `yaml:"prompt_via"` // stdin | arg
	Timeout   Duration `yaml:"timeout"`
	CanEdit   bool     `yaml:"can_edit"`
}

// placeholderValues maps each supported command placeholder to its
// configured value. Supporting a new optional setting means adding one entry
// here; Argv's expansion rules apply to it automatically.
func (a Agent) placeholderValues() map[string]string {
	return map[string]string{
		"{{model}}":  a.Model,
		"{{effort}}": a.Effort,
	}
}

// Argv renders the command template into argv. Each Command element is a
// template token: placeholders are substituted, a token referencing a
// placeholder that resolves to empty is dropped whole (so a flag and its
// value drop together), and surviving tokens are split on whitespace so
// "--model fable" becomes two argv elements.
func (a Agent) Argv() []string {
	values := a.placeholderValues()
	var argv []string
	for _, tok := range a.Command {
		sub, keep := expandToken(tok, values)
		if !keep {
			continue
		}
		argv = append(argv, strings.Fields(sub)...)
	}
	return argv
}

// expandToken substitutes placeholders in one command token. keep is false
// when the token references a placeholder whose value is empty, meaning the
// whole token must be omitted from argv.
func expandToken(tok string, values map[string]string) (sub string, keep bool) {
	for ph, v := range values {
		if !strings.Contains(tok, ph) {
			continue
		}
		if v == "" {
			return "", false
		}
		tok = strings.ReplaceAll(tok, ph, v)
	}
	return tok, true
}

// Loop controls how the review->fix cycle iterates, terminates, and commits,
// and holds the trust gates that permit fix rounds at all.
type Loop struct {
	MaxIterations int `yaml:"max_iterations"`

	// MaxFindingsPerRound caps how many findings a single fix round hands to
	// the coder (0 = unlimited). Findings are ordered worst-severity-first;
	// the overflow is marked deferred and re-surfaces in later rounds. Keeps
	// one coder session's workload inside its timeout.
	MaxFindingsPerRound int    `yaml:"max_findings_per_round"`
	ReviewOnly          bool   `yaml:"review_only"`
	CommitMessage       string `yaml:"commit_message"`
	CleanRoundsToStop   int    `yaml:"clean_rounds_to_stop"`

	// AllowUntrustedFix permits fix rounds in pr mode. A PR is
	// externally-authored code: its content flows through reviewer findings
	// into the coder prompt, and the coder edits files with permission checks
	// disabled, so a malicious PR can steer it via prompt injection. Off by
	// default; also available as the -allow-untrusted-fix CLI flag.
	AllowUntrustedFix bool `yaml:"allow_untrusted_fix"`

	// TrustedTarget asserts that a directory/git-diff target contains only code
	// the operator trusts. Fix rounds run the coder with permission checks
	// disabled and NOT confined to target.path, so any prompt-injection payload
	// in reviewed content could steer it into writing anywhere on the machine.
	// That risk is not unique to pr mode: a directory/git-diff tree can hold
	// vendored deps, a fetched base_ref, or a cloned third-party project. So
	// fix rounds in those modes are refused unless this is set (fail-closed by
	// default); pr mode uses AllowUntrustedFix instead. Also available as the
	// -trusted-target CLI flag.
	TrustedTarget bool `yaml:"trusted_target"`
}

// Logs configures where run artifacts are written and in which renderings.
type Logs struct {
	// Dir is a path TEMPLATE, not a fixed directory. It may contain
	// {timestamp} (the run's start time formatted per TimestampFormat -- one
	// value for the whole run, so every artifact of a run lands together) and
	// {round} (the loop iteration). Three paths are derived from it, each
	// documented on its method: StaticBase, RunPath, and RoundPath.
	Dir             string   `yaml:"dir"`
	Formats         []string `yaml:"formats"` // md | json | raw
	Pattern         string   `yaml:"pattern"`
	SummaryPattern  string   `yaml:"summary_pattern"`
	TimestampFormat string   `yaml:"timestamp_format"`
}

// StepPath renders logs.pattern into a per-step artifact path (relative to the
// round directory) for one agent invocation's identity. It is the single
// substitution shared by logstore -- which writes each artifact through it --
// and the log-collision validator, which renders every identity a run can
// produce (with round/timestamp/ext held fixed) to prove the pattern maps them
// to distinct paths. Sharing one renderer binds the collision check to the paths
// actually written, so the two cannot drift as placeholders are added.
func (l Logs) StepPath(role, agentName, promptName string, round int, ext, timestamp string) string {
	return strings.NewReplacer(
		"{role}", role,
		"{agent}", agentName,
		"{prompt}", promptName,
		"{round}", strconv.Itoa(round),
		"{timestamp}", timestamp,
		"{ext}", ext,
	).Replace(l.Pattern)
}

// dirSegments splits the logs.dir template into slash-separated segments. A
// leading empty segment (an absolute template) is preserved, so rejoining with
// "/" reproduces the original root.
func (l Logs) dirSegments() []string {
	return strings.Split(filepath.ToSlash(l.Dir), "/")
}

// joinSegments rebuilds a path from dirSegments output. strings.Join rather than
// filepath.Join because Join discards the leading empty segment of an absolute
// path, silently turning "/var/log/x" into "var/log/x".
func joinSegments(segs []string) string {
	return filepath.FromSlash(strings.Join(segs, "/"))
}

// roundSplit returns the index of the first logs.dir segment containing {round},
// or len(segments) when the template has none.
func (l Logs) roundSplit() ([]string, int) {
	segs := l.dirSegments()
	for i, s := range segs {
		if strings.Contains(s, "{round}") {
			return segs, i
		}
	}
	return segs, len(segs)
}

// StaticBase returns the leading literal segments of logs.dir -- everything
// before the first segment containing any placeholder. It is the only part of
// the path identical across every run and round, which makes it what the
// orchestrator excludes from round commits, clean checks, and collected
// material. Validation guarantees it is non-empty, so those exclusions can never
// silently cover nothing and let a run commit its own logs.
func (l Logs) StaticBase() string {
	segs := l.dirSegments()
	for i, s := range segs {
		if strings.Contains(s, "{") {
			return joinSegments(segs[:i])
		}
	}
	return joinSegments(segs)
}

// runSplit returns the logs.dir segments and the index just past the run-level
// part: the segment carrying {timestamp}, which validation guarantees exists and
// precedes any {round}. Splitting there rather than at the first {round} segment
// keeps intermediate literals like the "rounds" in ".../{timestamp}/rounds/{round}"
// on the per-round side, so the summary stays at the run root instead of landing
// in a directory that only groups rounds. Falls back to the pre-{round} part for a
// template with no {timestamp}, which only an unvalidated config has.
func (l Logs) runSplit() ([]string, int) {
	segs := l.dirSegments()
	for i, s := range segs {
		if strings.Contains(s, "{timestamp}") {
			return segs, i + 1
		}
	}
	_, cut := l.roundSplit()
	return segs, cut
}

// RunPath renders the run-level directory: the logs.dir segments up to and
// including the {timestamp} one. The summary lives here because it spans all
// rounds, and this is the directory claimed atomically at run start -- which is
// why validation requires {timestamp} in it, making each run's directory unique.
func (l Logs) RunPath(timestamp string) string {
	segs, cut := l.runSplit()
	return replaceDirPlaceholders(joinSegments(segs[:cut]), 0, timestamp)
}

// RoundPath renders one round's artifact directory beneath an already-claimed
// runDir (which RunPath produced, possibly with a collision suffix): every
// logs.dir segment after the {timestamp} one. When nothing follows it the round
// shares the run directory, which is why validation then requires {round} in
// logs.pattern.
func (l Logs) RoundPath(runDir string, round int, timestamp string) string {
	segs, cut := l.runSplit()
	if cut >= len(segs) {
		return runDir
	}
	return filepath.Join(runDir, replaceDirPlaceholders(joinSegments(segs[cut:]), round, timestamp))
}

func replaceDirPlaceholders(s string, round int, timestamp string) string {
	return strings.NewReplacer(
		"{timestamp}", timestamp,
		"{round}", strconv.Itoa(round),
	).Replace(s)
}

// dirPlaceholders returns every {placeholder} token in a logs.dir template, so
// an unknown one (a typo) can be rejected rather than becoming a directory
// literally named "{tiemstamp}".
func dirPlaceholders(dir string) []string {
	var out []string
	for i := 0; i < len(dir); {
		open := strings.IndexByte(dir[i:], '{')
		if open < 0 {
			break
		}
		open += i
		end := strings.IndexByte(dir[open:], '}')
		if end < 0 {
			out = append(out, dir[open:]) // unterminated; report it verbatim
			break
		}
		end += open
		out = append(out, dir[open:end+1])
		i = end + 1
	}
	return out
}

// Duration wraps time.Duration for YAML ("10m", "1h30m").
type Duration time.Duration

// UnmarshalYAML parses a Go duration string ("10m", "1h30m").
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

// Std converts to the standard library's time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Load reads, defaults, and validates the configuration file. Most callers
// want this. A caller that must apply CLI overrides to the effective
// configuration before it is validated (e.g. -review-only, which exempts a run
// from the recurring-lens rule) uses LoadUnvalidated and calls Validate itself
// once the overrides are in place.
func Load(path string) (*Config, error) {
	cfg, err := LoadUnvalidated(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// LoadUnvalidated reads and defaults the configuration file WITHOUT running
// Validate. It exists so the CLI can apply flag overrides to the loaded config
// and then validate the effective result: validating before the overrides land
// would, for example, reject an all-once lens list that -review-only makes
// valid. Callers that do not override anything should use Load.
func LoadUnvalidated(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // typos in keys are errors, not silent defaults
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Target.Path == "" {
		c.Target.Path = "."
	}
	// Default only the absent (zero) case; a negative value is invalid operator
	// input, not "unset", and Validate rejects it rather than silently masking it
	// with a positive default.
	if c.Loop.MaxIterations == 0 {
		c.Loop.MaxIterations = 5
	}
	if c.Loop.CleanRoundsToStop == 0 {
		c.Loop.CleanRoundsToStop = 1
	}
	if c.Loop.CommitMessage == "" {
		c.Loop.CommitMessage = "fixpoint: round {round} ({fixed} fixed, {rejected} rejected)"
	}
	if c.Logs.Dir == "" {
		c.Logs.Dir = ".fixpoint/{timestamp}/round-{round}"
	}
	if len(c.Logs.Formats) == 0 {
		c.Logs.Formats = []string{"md", "json", "raw"}
	}
	if c.Logs.Pattern == "" {
		// Round is conveyed by the round-N/ subdirectory, so it is not
		// repeated in the filename by default.
		c.Logs.Pattern = "{role}-{agent}-{prompt}-{timestamp}.{ext}"
	}
	if c.Logs.SummaryPattern == "" {
		c.Logs.SummaryPattern = "summary-{timestamp}.{ext}"
	}
	if c.Logs.TimestampFormat == "" {
		c.Logs.TimestampFormat = "20060102-150405"
	}
	c.Verify.applyDefaults()
	for name, a := range c.Agents {
		a.applyDefaults()
		c.Agents[name] = a
	}
}

// Validate performs the static startup validation documented in the config
// header: references resolve, binaries exist, prompt files are readable, the
// coder can edit, and the strategy is satisfiable.
//
//nolint:gocognit,cyclop // a flat checklist of independent rules; splitting it apart would obscure the sequence
func (c *Config) Validate() error {
	switch c.Target.Mode {
	case ModeGitDiff, ModePR, ModeDirectory:
	default:
		return fmt.Errorf("target.mode: unknown mode %q (want git-diff | pr | directory)", c.Target.Mode)
	}
	if c.Target.Mode == ModePR && c.Target.PR <= 0 {
		return errors.New("target.pr: PR number required for mode pr")
	}
	if c.Loop.MaxFindingsPerRound < 0 {
		return fmt.Errorf("loop.max_findings_per_round: must not be negative, got %d", c.Loop.MaxFindingsPerRound)
	}
	if c.Loop.MaxIterations < 0 {
		return fmt.Errorf("loop.max_iterations: must not be negative, got %d", c.Loop.MaxIterations)
	}
	if c.Loop.CleanRoundsToStop < 0 {
		return fmt.Errorf("loop.clean_rounds_to_stop: must not be negative, got %d", c.Loop.CleanRoundsToStop)
	}

	if len(c.Roles.Review.Prompts) == 0 {
		return errors.New("roles.review.prompts: at least one review lens is required. A config with no lenses is a base meant to be inherited with `extends`, not run directly -- `fixpoint --list` marks which configs are runnable")
	}
	switch c.Roles.Review.Strategy {
	case StrategyFixed:
		for _, l := range c.Roles.Review.Prompts {
			if l.Agent == "" {
				return fmt.Errorf("roles.review: strategy fixed requires every lens to pin an agent (%s does not)", l.Prompt)
			}
		}
	case StrategyRotate, StrategyAll:
		if len(c.Roles.Review.Agents) == 0 {
			return fmt.Errorf("roles.review.agents: strategy %s requires a non-empty agent pool", c.Roles.Review.Strategy)
		}
		// Reject blank pool entries directly: ActiveAgents (which drives the
		// binary/prompt_via checks below and the preflight ping) silently drops
		// empty names, so a pool like [""] would otherwise pass validation, yet
		// assignments still selects it and agent.Run would index an empty argv.
		seenAgent := map[string]int{}
		for i, a := range c.Roles.Review.Agents {
			if strings.TrimSpace(a) == "" {
				return fmt.Errorf("roles.review.agents: entry %d is empty; every agent pool entry must name a defined agent", i)
			}
			// A duplicate pool entry produces duplicate assignments: under strategy
			// all the same agent is invoked twice per lens, recording the same review
			// twice (with distinct IDs) so the coder receives and counts it twice.
			if j, dup := seenAgent[a]; dup {
				return fmt.Errorf("roles.review.agents: entries %d and %d both name %q; pool entries must be distinct so a reviewer is not assigned (and its findings counted) twice", j, i, a)
			}
			seenAgent[a] = i
		}
	default:
		return fmt.Errorf("roles.review.strategy: unknown strategy %q (want fixed | rotate | all)", c.Roles.Review.Strategy)
	}

	// A fix run must have at least one recurring (non-once) reviewer lens.
	// once-only lenses run in round 1 only, so if EVERY lens is once, rounds
	// after the first have no reviewers: the fix would then be "verified" by an
	// empty review and the run could converge without any finding ever being
	// re-checked. review-only runs are a single round, so this does not apply.
	if !c.Loop.ReviewOnly {
		recurring := false
		for _, l := range c.Roles.Review.Prompts {
			if !l.Once {
				recurring = true
				break
			}
		}
		if !recurring {
			return errors.New("roles.review.prompts: a fix run needs at least one recurring reviewer lens to verify each round, but every lens is once: true (they run only in round 1); set once: false on one, or use loop.review_only")
		}
	}

	// Every referenced agent must be defined, its binary on PATH, and its
	// prompt_via valid.
	check := func(ctx, name string) error {
		a, ok := c.Agents[name]
		if !ok {
			return fmt.Errorf("%s: agent %q is not defined in the agents section", ctx, name)
		}
		argv := a.Argv()
		if len(argv) == 0 {
			return fmt.Errorf("agents.%s: empty command", name)
		}
		bin := argv[0]
		// agent.Run sets Cmd.Dir = target.path, and the OS resolves a relative
		// executable that contains a path separator against THAT directory, not
		// fixpoint's working directory. Validate it where it will actually run
		// so a target-relative binary is neither wrongly rejected nor wrongly
		// accepted because it happens to exist in the launch dir. A bare command
		// name (no separator) is a PATH lookup and is unaffected by Cmd.Dir.
		//
		// Resolve to an ABSOLUTE path: filepath.Join cleans "./agent.sh" with
		// target.path "." down to "agent.sh", which drops the separator and would
		// send exec.LookPath back to a PATH search. Making it absolute keeps a
		// separator so LookPath validates the file directly.
		if !filepath.IsAbs(bin) && strings.ContainsRune(bin, filepath.Separator) {
			abs, err := filepath.Abs(filepath.Join(c.Target.Path, bin))
			if err != nil {
				return fmt.Errorf("agents.%s: resolving %q against target.path: %w", name, bin, err)
			}
			bin = abs
		}
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("agents.%s: binary %q not found on PATH", name, argv[0])
		}
		if a.PromptVia != PromptViaStdin && a.PromptVia != PromptViaArg {
			return fmt.Errorf("agents.%s: prompt_via must be stdin or arg, got %q", name, a.PromptVia)
		}
		// A negative timeout survives applyDefaults (which only fills a zero
		// value), and context.WithTimeout would then build an already-expired
		// context so every invocation fails instantly. Reject it at validation.
		if a.Timeout.Std() < 0 {
			return fmt.Errorf("agents.%s: timeout must not be negative, got %s", name, a.Timeout.Std())
		}
		return nil
	}

	if c.Roles.Coder.Agent == "" {
		return errors.New("roles.coder.agent: required")
	}
	if err := check("roles.coder", c.Roles.Coder.Agent); err != nil {
		return err
	}
	if !c.Agents[c.Roles.Coder.Agent].CanEdit {
		return fmt.Errorf("roles.coder: agent %q has can_edit: false -- the coder must be able to edit files", c.Roles.Coder.Agent)
	}
	for _, name := range c.Roles.Review.ActiveAgents() {
		if err := check("roles.review", name); err != nil {
			return err
		}
		// Reviewers run in parallel against the one shared working tree. A
		// write-capable reviewer could edit, delete, or regenerate the same
		// files as another reviewer (or as the serialized coder phase),
		// corrupting or losing edits and letting stray reviewer changes be
		// committed as coder fixes. Keep write access exclusive to the coder.
		if c.Agents[name].CanEdit {
			return fmt.Errorf("roles.review: agent %q has can_edit: true -- reviewers run concurrently against the shared working tree and must be read-only; only roles.coder may edit files", name)
		}
	}

	// A lens's identity in logs is its prompt basename (LensName), and the
	// per-step log path is {role}-{agent}-{prompt}-... So two lenses whose
	// prompt files share a basename (prompts/a/review.md and prompts/b/review.md)
	// produce the same {prompt} token; under strategy all one agent runs both in
	// a round and their reviewer goroutines race to os.WriteFile the same path,
	// silently losing one durable record. Require distinct lens names so the
	// per-assignment log path is always unique.
	seenLens := map[string]int{}
	for i, l := range c.Roles.Review.Prompts {
		name := LensName(l.Prompt)
		if j, dup := seenLens[name]; dup {
			return fmt.Errorf("roles.review.prompts: lenses %d (%s) and %d (%s) both resolve to log name %q; prompt basenames must be unique so parallel reviewers' step logs do not overwrite each other -- rename one prompt file", j, c.Roles.Review.Prompts[j].Prompt, i, l.Prompt, name)
		}
		seenLens[name] = i
	}

	// Every role must name a prompt, and the resolved file must be readable. The
	// loader fills PromptPath; it is empty when Validate is called on a config
	// that was never resolved through a bundle (unit tests), in which case the
	// name itself is treated as the path.
	type promptRef struct{ name, file string }
	refs := []promptRef{{c.Roles.Coder.Prompt, c.Roles.Coder.PromptFile()}}
	for _, l := range c.Roles.Review.Prompts {
		refs = append(refs, promptRef{l.Prompt, l.PromptFile()})
	}
	for _, ref := range refs {
		if ref.name == "" {
			return errors.New("roles: every role entry must reference a prompt file by name")
		}
		if _, err := os.Stat(ref.file); err != nil {
			return fmt.Errorf("prompt file %s (%s): %w", ref.name, ref.file, err)
		}
	}

	for _, f := range c.Logs.Formats {
		if f != "md" && f != "json" && f != "raw" {
			return fmt.Errorf("logs.formats: unknown format %q (want md | json | raw)", f)
		}
	}

	// Log filenames must stay distinct across every artifact a run writes.
	// applyDefaults fills empty patterns with known-good ones, so only validate
	// patterns the config actually set. Both patterns render each configured
	// format (the step pattern also writes a .prompt; the summary writes md +
	// json), so a pattern without {ext} makes those writes target one path -- a
	// JSON write clobbers the Markdown one and Summary returns that JSON path as
	// the Markdown path. The step pattern additionally separates every
	// role/agent/prompt in a round: under strategy all one agent runs several
	// lenses whose reviewer goroutines write concurrently, so dropping {prompt}
	// (or {agent}/{role}) races them onto one path and silently loses a record.
	if c.Logs.Pattern != "" {
		if !strings.Contains(c.Logs.Pattern, "{ext}") {
			return fmt.Errorf("logs.pattern %q must contain {ext}, else the md/json/raw and prompt writes overwrite one file", c.Logs.Pattern)
		}
		for _, ph := range []string{"{role}", "{agent}", "{prompt}"} {
			if !strings.Contains(c.Logs.Pattern, ph) {
				return fmt.Errorf("logs.pattern %q must contain %s, else concurrent step logs (parallel reviewer lenses, or a reviewer vs the coder) overwrite one file", c.Logs.Pattern, ph)
			}
		}
		// Presence of {role}/{agent}/{prompt} does not make their concatenation
		// injective: "{role}{agent}{prompt}.{ext}" maps reviewer (agent=a,
		// prompt=bc) and (agent=ab, prompt=c) to one path. Render every identity a
		// run can produce -- holding {round}/{timestamp}/{ext} fixed, as they are
		// equal for the concurrent same-round writes that race -- and reject any two
		// distinct identities that collide onto one path.
		seenPath := map[string][3]string{}
		for _, id := range c.logIdentities() {
			p := c.renderLogPattern(id)
			if prev, dup := seenPath[p]; dup && prev != id {
				return fmt.Errorf("logs.pattern %q maps distinct log identities %v and %v to the same path %q; under strategy all their step/prompt writes run concurrently and would race onto it, silently overwriting one reviewer's or the coder's record -- add a separator between (or reorder) the {role}/{agent}/{prompt} placeholders so every identity renders a distinct path", c.Logs.Pattern, prev, id, p)
			}
			seenPath[p] = id
		}
	}
	if c.Logs.SummaryPattern != "" && !strings.Contains(c.Logs.SummaryPattern, "{ext}") {
		return fmt.Errorf("logs.summary_pattern %q must contain {ext}, else the JSON summary overwrites the Markdown one (and its path is returned as the Markdown path)", c.Logs.SummaryPattern)
	}
	if err := c.Verify.validate(); err != nil {
		return err
	}
	return c.validateLogsDir()
}

// validateLogsDir checks the three properties the logs layout depends on. Each
// failure is silent data loss (or a committed logs dir) if it slips through, so
// they are rejected at startup rather than discovered in the artifacts.
func (c *Config) validateLogsDir() error {
	// Empty means "unset": Load substitutes the default template, which satisfies
	// every rule below. Same convention as logs.pattern's checks -- validating the
	// zero value here would reject a config that simply omits the key.
	if c.Logs.Dir == "" {
		return nil
	}
	for _, ph := range dirPlaceholders(c.Logs.Dir) {
		if ph != "{timestamp}" && ph != "{round}" {
			return fmt.Errorf("logs.dir %q: unknown placeholder %s (logs.dir supports {timestamp} and {round}); a typo would otherwise create a directory literally named that", c.Logs.Dir, ph)
		}
	}
	// A literal leading segment is what the orchestrator excludes from round
	// commits, clean checks, and collected material. Without one there is nothing
	// stable to exclude and a run would sweep its own logs into a fix commit.
	switch base := c.Logs.StaticBase(); base {
	case "", ".", string(filepath.Separator):
		return fmt.Errorf("logs.dir %q must begin with at least one literal path segment before its first placeholder (e.g. .fixpoint/{timestamp}/round-{round}); that literal prefix is what round commits, clean checks, and collected material exclude, so without it a run would commit and re-review its own logs", c.Logs.Dir)
	}
	// The run-level part (everything before the first {round} segment) is claimed
	// atomically at run start and holds the summary. Without {timestamp} there,
	// consecutive runs share one directory and overwrite each other.
	segs, cut := c.Logs.roundSplit()
	if runPart := joinSegments(segs[:cut]); !strings.Contains(runPart, "{timestamp}") {
		return fmt.Errorf("logs.dir %q: the part before the first {round} segment (%q) must contain {timestamp}, else consecutive runs share one directory and overwrite each other's artifacts and summary", c.Logs.Dir, runPart)
	}
	// Rounds must be separated by the directory or by the filename; otherwise
	// round 2 overwrites round 1 artifact for artifact.
	if cut == len(segs) && !strings.Contains(c.Logs.Pattern, "{round}") {
		return fmt.Errorf("logs.dir %q has no {round} segment, so every round writes into one directory -- add {round} to logs.dir (e.g. .../round-{round}) or to logs.pattern, else each round overwrites the previous one's artifacts", c.Logs.Dir)
	}
	return nil
}

// logIdentities returns every (role, agent, prompt) triple a run can log a step
// or prompt for: each reviewer lens under every agent it can be assigned to
// (Review.LensAgents, the same policy the orchestrator's fan-out consumes), plus
// the coder. It is the identity space the step-log path must render injectively
// over.
func (c *Config) logIdentities() [][3]string {
	var ids [][3]string
	rv := c.Roles.Review
	for _, l := range rv.Prompts {
		name := LensName(l.Prompt)
		for _, a := range rv.LensAgents(l) {
			ids = append(ids, [3]string{"review", a, name})
		}
	}
	ids = append(ids, [3]string{"fix", c.Roles.Coder.Agent, LensName(c.Roles.Coder.Prompt)})
	return ids
}

// renderLogPattern fills logs.pattern for one identity, holding {round},
// {timestamp}, and {ext} fixed: those are equal for the concurrent same-round
// writes whose collisions this check guards against, so only the identity
// placeholders may distinguish the resulting paths. It renders through the same
// Logs.StepPath the logstore writes with, so validation cannot drift from the
// paths actually produced.
func (c *Config) renderLogPattern(id [3]string) string {
	return c.Logs.StepPath(id[0], id[1], id[2], 1, "", "")
}

// applyDefaults fills an agent's optional fields. It is a method on Agent rather
// than inline in Config.applyDefaults because agents are loaded from their own
// files too, and both paths must default identically.
func (a *Agent) applyDefaults() {
	if a.PromptVia == "" {
		a.PromptVia = PromptViaStdin
	}
	if a.Timeout == 0 {
		a.Timeout = Duration(10 * time.Minute)
	}
}

// mandatoryExcludes are removed from directory-mode collection no matter what
// target.exclude says. Every reviewer runs unsandboxed and can read any path, so
// a prompt-injected one that is *pointed at* a credential file is the realistic
// leak path -- and once configs can be modularized, inherited, and shadowed, an
// exclude list is one omission away from dropping that protection silently. A
// safety property must not be something a config can forget, so these live in
// code and are appended to whatever the config asks for.
//
// This bounds the damage; it is not a sandbox. The security notes in the shipped
// configs still apply.
var mandatoryExcludes = []string{
	"**/.env", "**/.env.*", // .env.local, .env.production, ...
	"**/*.pem", "**/*.p12", "**/*.pfx", "**/*.key",
	"**/id_rsa", "**/id_dsa", "**/id_ecdsa", "**/id_ed25519",
	"**/.npmrc", "**/.netrc", "**/.pgpass",
	"**/credentials", // ~/.aws/credentials shape
	"**/*.kdbx",
}

// EffectiveExcludes returns the configured exclude globs plus the mandatory
// credential patterns, deduplicated.
func (t Target) EffectiveExcludes() []string {
	seen := make(map[string]bool, len(t.Exclude)+len(mandatoryExcludes))
	out := make([]string, 0, len(t.Exclude)+len(mandatoryExcludes))
	for _, g := range append(append([]string{}, t.Exclude...), mandatoryExcludes...) {
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	return out
}

// VerifyPolicy decides how a verification failure is treated. Verification is
// the only non-model signal in the loop -- everything else is one agent's opinion
// checked by another agent's opinion -- so what "failed" means is a policy
// decision, not a detail.
type VerifyPolicy string

const (
	// VerifyOff runs nothing. Also the effective policy when no commands are set.
	VerifyOff VerifyPolicy = "off"
	// VerifyNoRegressions compares against a baseline captured before the first
	// fix round: a command already failing then may keep failing, but one that
	// passed must not start failing. The right default for a real repository,
	// which may well start red.
	VerifyNoRegressions VerifyPolicy = "no_regressions"
	// VerifyMustPass requires every required command to succeed regardless of the
	// baseline.
	VerifyMustPass VerifyPolicy = "must_pass"
)

var verifyPolicies = []VerifyPolicy{VerifyOff, VerifyNoRegressions, VerifyMustPass}

// Verify configures the deterministic quality gate fixpoint runs itself, after
// the coder and before the round commit.
//
// It is deliberately fixpoint's job rather than an instruction in the coder
// prompt: the most objective step in the workflow should not depend on the least
// deterministic participant, and a coder that self-reports "fixed" with nothing
// checking it makes both "fixed" and "converged" mean only that a model said so.
//
// SECURITY: these commands run code from, and defined by, the target project.
// They therefore execute only on the fix path, which already requires an explicit
// trust assertion (see Loop.TrustedTarget), and a bundle resolved from inside the
// target is itself treated as untrusted input -- see Config.ProjectSuppliedExec.
type Verify struct {
	Policy VerifyPolicy `yaml:"policy"`
	// Timeout bounds EACH command. A hung test suite must not hang the run.
	Timeout  Duration        `yaml:"timeout"`
	Commands []VerifyCommand `yaml:"commands"`
}

// VerifyCommand is one check. Argv, not a shell string: there is no shell to
// quote wrongly, and it matches how agents are configured.
type VerifyCommand struct {
	Name string   `yaml:"name"`
	Run  []string `yaml:"run"`
	// Optional records the result without ever failing the round. For a check
	// that is informative but not a gate (a linter mid-cleanup, say).
	Optional bool `yaml:"optional"`
}

// Enabled reports whether verification will run. No commands means off, whatever
// the policy says, so a config can carry a policy without every project having to
// define commands for it.
func (v Verify) Enabled() bool {
	return v.Policy != VerifyOff && len(v.Commands) > 0
}

func (v *Verify) applyDefaults() {
	if v.Policy == "" {
		// no_regressions rather than must_pass: fixpoint runs against repositories
		// it did not write, and demanding green from the first round would refuse to
		// work on any project with a pre-existing failure.
		v.Policy = VerifyNoRegressions
	}
	if v.Timeout == 0 {
		v.Timeout = Duration(10 * time.Minute)
	}
}

func (v Verify) validate() error {
	// Empty means unset: Load substitutes the default. Validating the zero value
	// would reject a config that simply omits the section -- same convention as
	// logs.dir and logs.pattern.
	if v.Policy == "" && len(v.Commands) == 0 && v.Timeout == 0 {
		return nil
	}
	if !slices.Contains(verifyPolicies, v.Policy) {
		return fmt.Errorf("verify.policy %q is unknown (want %v)", v.Policy, verifyPolicies)
	}
	if v.Timeout.Std() < 0 {
		return fmt.Errorf("verify.timeout %s must not be negative", v.Timeout.Std())
	}
	seen := map[string]bool{}
	for i, c := range v.Commands {
		if c.Name == "" {
			return fmt.Errorf("verify.commands[%d]: name is required; it identifies the check in logs, in the summary, and in the report handed back to the coder", i)
		}
		if seen[c.Name] {
			return fmt.Errorf("verify.commands: duplicate name %q; names must be unique so a baseline result maps to exactly one command", c.Name)
		}
		seen[c.Name] = true
		if len(c.Run) == 0 {
			return fmt.Errorf("verify.commands[%s]: run must be a non-empty argv list", c.Name)
		}
	}
	return nil
}
