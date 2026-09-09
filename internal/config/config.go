// Package config loads and validates fixpoint.yaml.
package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/dsaiko/fixpoint/internal/model"
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
	Create Create           `yaml:"create"`
	// Implement configures the implement-design pipeline; inert (and refused)
	// unless roles.planner is set. See docs/design/DESIGN.md.
	Implement Implement `yaml:"implement"`

	// PingAgents: before a run, invoke every agent used by the run with a
	// trivial prompt (in parallel) and abort if any fails. Catches expired
	// logins and broken CLIs before tokens are spent. Default true.
	PingAgents *bool `yaml:"ping_agents"`
	// Review is the verdict policy for review-only runs; ignored by a fix run,
	// which has no verdict.
	Review ReviewPolicy `yaml:"review"`
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
	// PRFromBranch says the number is to be resolved from the checked-out branch
	// because the invocation supplied none. It carries `yaml:"-"`, and that is
	// deliberate: it is an assertion about how the command was TYPED (no -pr),
	// which a file cannot make on the operator's behalf -- the same reason the
	// trust gates are flags. See Overrides.PRFromBranch, and
	// Orchestrator.resolvePR for where the number then comes from.
	PRFromBranch bool `yaml:"-"`
	// Document narrows a directory target to ONE FILE, which becomes the material
	// itself -- shown to the panel in full rather than as a listing entry. It is how
	// review-design reads a design document: the reviewers judge the text, and the
	// surrounding directory is still their working directory, so anything the
	// document references (diagrams, earlier drafts, the project itself) stays
	// readable.
	//
	// Relative to Path, or absolute. Usually set per invocation via -target, which
	// accepts a file or a directory and fills in Path/Document accordingly -- which
	// file to review is per-invocation by nature, the same argument as BaseRef and
	// PR.
	Document string `yaml:"document"`
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
	// Judge is the optional arbiter for a REVIEW run: a read-only agent that sees
	// the surviving findings and decides which are worth reporting.
	//
	// It is a separate role from the coder rather than the coder itself, and that is
	// load-bearing: the coder is can_edit, and a review- config's whole promise is
	// that it never invokes something that can modify the target. Same judgment,
	// different hands.
	Judge RoleRef `yaml:"judge"`
	// Editor is the pen-holder of a CREATE run: the read-only agent that reads
	// every proposal and critique and WRITES the design document. It exists
	// because the panel does not converge -- measured at under 4% corroboration
	// -- so "the models agree on a design" is a fiction; somebody must hold the
	// pen, and dissent is recorded rather than resolved.
	//
	// Read-only for the same reason as the judge: it is the single voice the
	// deliverable speaks with, fed exclusively by text other models wrote about
	// an untrusted assignment, and the decision-maker must not be the thing that
	// can also act.
	Editor RoleRef `yaml:"editor"`
	// Planner is the decomposer of an IMPLEMENT run: the read-only agent that
	// reads the design document once and returns the ordered task plan the coder
	// then executes one session at a time. Setting it is what makes a config an
	// implement config (IsImplement).
	//
	// Read-only for the same reason as the judge and the editor: the plan decides
	// what forty coder sessions will build, it is derived entirely from an
	// untrusted document, and the thing that decides must not be the thing that
	// can act. The plan itself is data, never executed -- see
	// docs/design/DESIGN.md §4.
	Planner RoleRef `yaml:"planner"`
	// Triage is the optional arbiter for the pull request's OPEN CONVERSATIONS: a
	// read-only agent that reads every unresolved comment and decides, one by one,
	// whether it names real work.
	//
	// Naming it turns comments from context into input. Without it a coder is shown
	// the conversations and told to leave them alone; with it, an accepted comment
	// becomes an issue that goes through the ordinary pipeline -- one session, the
	// verify gate, its own commit -- and a rejected one gets an answer saying why.
	// Every conversation ends with a decision and a reply either way.
	//
	// Read-only for the same reason as the judge: this agent reads text that anyone
	// with access to the pull request can write, and the decision it makes must not
	// be made by something that can also edit the tree.
	//
	// It is only meaningful in pr mode; in any other mode there are no
	// conversations and the step does not run.
	Triage RoleRef `yaml:"triage"`
}

// IsCreate reports whether this config runs the create-design pipeline rather
// than a review/fix loop. The discriminator is create.propose: a create run is
// defined by having proposals to make.
func (c *Config) IsCreate() bool { return c.Create.Propose != "" }

// IsImplement reports whether this config runs the implement-design pipeline: a
// planner decomposes a design document into tasks and a coder builds them, one
// session and one commit per task. The discriminator is roles.planner -- an
// implement run is defined by having a plan made.
func (c *Config) IsImplement() bool {
	return c.Roles.Planner.Agent != "" || c.Roles.Planner.Prompt != ""
}

// Create configures the create-design pipeline: a panel drafts independent
// proposals, critiques each other's anonymously, and the editor synthesizes the
// deliverable. Setting `propose` is what makes a config a CREATE config; the
// full pipeline shape is docs/design/create-design.md.
type Create struct {
	// Propose names the prompt each pool agent drafts its proposal from.
	Propose     string `yaml:"propose"`
	ProposePath string `yaml:"-"`
	// Critique names the prompt each agent critiques the OTHERS' proposals with.
	Critique     string `yaml:"critique"`
	CritiquePath string `yaml:"-"`
	// Object names the prompt for the bounded objection pass over the editor's
	// draft. Required only when Objections > 0.
	Object     string `yaml:"object"`
	ObjectPath string `yaml:"-"`
	// Out is where the deliverable is written; empty means DESIGN.md beside the
	// assignment. Usually set per invocation via -out (resolved absolute by the
	// CLI), and never under .fixpoint/ -- a deliverable is not a log. fixpoint
	// refuses to overwrite whatever it points at.
	Out string `yaml:"out"`
	// Objections is how many objection passes run over the editor's draft
	// (each followed by the editor's REVISE). 0 disables both phases; the
	// shipped config sets 1, and more than 1 is refused -- an objection loop
	// does not converge, the same measurement behind max_final_passes.
	Objections int `yaml:"objections"`
}

// Implement configures the implement-design pipeline: a planner decomposes a
// reviewed design document into ordered tasks, and the coder builds them into a
// fresh repository, one session and one gated commit per task. Setting
// roles.planner is what makes a config an IMPLEMENT config; the full pipeline
// shape is docs/design/DESIGN.md. Every number here is a guess recorded as a
// config key on purpose (§12.2 there): a wrong one surfaces as a refusal at
// startup, never as a truncated run.
type Implement struct {
	// MaxTasks caps the plan; over it is a refusal ("split the design"), not a
	// truncation. The deadline usually binds first (§4.2 rule 7).
	MaxTasks int `yaml:"max_tasks"`
	// MaxFilesPerTask bounds one task's advisory file list.
	MaxFilesPerTask int `yaml:"max_files_per_task"`
	// MaxTaskAttempts is how many fresh sessions a task gets before it is
	// failed. An attempt is one session and one gate run; infrastructure
	// failures do not consume attempts (§5.4).
	MaxTaskAttempts int `yaml:"max_task_attempts"`
	// MaxVacuousFrac ends the run incomplete when more than this fraction of
	// tasks report already_satisfied -- a badly decomposed plan, not progress.
	MaxVacuousFrac float64 `yaml:"max_vacuous_frac"`
	// MaxRunDuration is the whole run's deadline, checked between tasks and
	// enforced against the plan's WORST case at plan time (§4.2 rule 7).
	//
	// It decides how many tasks a run may plan, so it and the stack's
	// verify.timeout are one number in two places: the worst case is
	// tasks x attempts x (session + SUM of per-command timeouts + 1m), plus the
	// clone gate runs clean_check buys. At the shipped 32h the stacks admit 20
	// (go), 17 (node) and 30 (web) tasks -- and three fewer each at
	// clean_check: every. Raising this without touching the timeouts buys
	// proportionally more tasks; it is not a number to change on its own.
	MaxRunDuration Duration `yaml:"max_run_duration"`
	// MaxInfraTries is how many times an infrastructure failure -- a provider
	// refusal, a dead session, a gate command declared `infra` -- is retried
	// before the circuit breaker stops the run incomplete. Each retry waits
	// longer than the last (1m, 5m, 15m, 30m at the default of 4), and the wait
	// is charged against max_run_duration.
	//
	// A key rather than a constant because it is one of §12.2's guessed numbers:
	// how long an outage has to last before finishing later beats waiting is a
	// property of the operator's provider, not of this tool. It shipped as a
	// bare "two consecutive failures" with no wait at all, which meant the
	// second call landed inside the same rate-limit window as the first.
	MaxInfraTries int `yaml:"max_infra_tries"`
	// MaxTaskBytes bounds one attempt's un-ignored changes, checked during the
	// census walk before anything is hashed or gated.
	MaxTaskBytes ByteSize `yaml:"max_task_bytes"`
	// MinFreeDisk refuses at preflight and stops incomplete between tasks when
	// the write-target's filesystem drops under it.
	MinFreeDisk ByteSize `yaml:"min_free_disk"`
	// CleanCheck is when fixpoint clones HEAD and runs the gate in the clone,
	// proving the committed bytes alone satisfy it: "last" (default; once, at
	// the end of the run), "every" (after every task commit, for attribution),
	// or "off". Forced off on an ungated run.
	CleanCheck string `yaml:"clean_check"`
	// NoCoverageCheck skips §4.2 rule 6, letting a run proceed with a design
	// whose outline the coverage rule cannot read -- and making every report say
	// `coverage: unchecked` instead of implying a check that never ran (§7.4).
	//
	// `yaml:"-"`, and for the same reason as §7.4's other flags: this is a
	// per-invocation judgement about ONE document. A config key would let a
	// bundle -- including one resolved from inside the design's own repository --
	// switch off the rule that notices when the plan dropped a section, for every
	// run, silently. The operator waives it per run or not at all.
	NoCoverageCheck bool `yaml:"-"`
	// PlanOnly stops the run after the plan is validated: the artifacts are
	// written, no directory is claimed and no coder session is spent (§7.4).
	// yaml:"-" for the same reason as the rest of §7.4's flags -- what a single
	// invocation is FOR is not something a config file decides.
	PlanOnly bool `yaml:"-"`
	// Plan is a path to an operator-supplied or previously-generated plan to run
	// instead of a planner session (§7.4). yaml:"-": a config key naming a plan
	// file would let a bundle -- possibly one shipped inside the design's own
	// repository -- decide what gets built, bypassing the planner entirely.
	Plan string `yaml:"-"`
	// Continue resumes a project fixpoint built, taking the plan and the design
	// from that repository (§5.5). yaml:"-" like the rest: which project to
	// resume is an invocation, not a setting.
	Continue string `yaml:"-"`
	// GitignoreSeed is the stack's ignore entries, written by fixpoint into the
	// bootstrap commit's .gitignore -- a control artifact no session may edit.
	GitignoreSeed []string `yaml:"gitignore_seed"`
	// GateGenerated names committed files the GATE maintains (go.sum,
	// package-lock.json): staged into the task commit with gate attribution,
	// neither a mutation failure nor removable output (§5.2 step 7).
	GateGenerated []string `yaml:"gate_generated"`
}

// CleanCheck cadences.
const (
	CleanCheckLast  = "last"
	CleanCheckEvery = "every"
	CleanCheckOff   = "off"
)

// applyDefaults fills the implement numbers in code -- a stack config carries
// only what an operator would edit (DESIGN.md §7.1), because the loader's
// one-level extends is deliberate and a shared implement-design.yaml base would
// need a second. Each is a guess (§12.2 there): a key, so a wrong one is the
// operator's to correct, and zero-only defaulting so Validate still rejects
// negatives.
func (i *Implement) applyDefaults() {
	if i.MaxTasks == 0 {
		i.MaxTasks = 40
	}
	if i.MaxFilesPerTask == 0 {
		i.MaxFilesPerTask = 12
	}
	if i.MaxTaskAttempts == 0 {
		i.MaxTaskAttempts = 2
	}
	if i.MaxVacuousFrac == 0 {
		i.MaxVacuousFrac = 0.34
	}
	if i.MaxInfraTries == 0 {
		i.MaxInfraTries = 4
	}
	if i.MaxRunDuration == 0 {
		i.MaxRunDuration = Duration(32 * time.Hour)
	}
	if i.MaxTaskBytes == 0 {
		i.MaxTaskBytes = 64 << 20
	}
	if i.MinFreeDisk == 0 {
		i.MinFreeDisk = 2 << 30
	}
	if i.CleanCheck == "" {
		i.CleanCheck = CleanCheckLast
	}
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

	// Final holds the lens out of the review->fix loop entirely and runs it once,
	// on EVERY agent in the pool, in a single closing round after the loop has
	// finished -- its findings then go to the coder like any other.
	//
	// It exists for a lens whose subject is the FINISHED code rather than the code
	// in front of it. review-tests is the case: run inside the loop it assesses
	// coverage of work later rounds rewrite, so it demands tests for intermediate
	// states and re-reports the gap once the code moves. In a five-round run that
	// made it the largest producer (30 of 68 reports) and the largest waster (18
	// deferred), and 65% of everything the run wrote was test code. Deferring it to
	// the end means the coverage question is asked once, about code that has stopped
	// changing, and the whole panel asks it -- it is the last look, so breadth is
	// worth more than rotation.
	//
	// Unlike `advisory: true` the findings are still FIXED; unlike a normal lens it
	// cannot starve the loop, because nothing follows it.
	Final bool `yaml:"final"`

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
		case "agent", "prompt", "advisory", "once", "final":
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

// ReformatLensSuffix is appended to a lens name for the second invocation the
// orchestrator makes when a reviewer's reply broke the output contract. That
// invocation logs its own prompt and step under the suffixed name, so the suffix
// is part of the log identity space (logIdentities) and the name is reserved:
// both live here so validation cannot drift from the paths actually written.
const ReformatLensSuffix = "-reformat"

// ReformatLensName is the {prompt} log token for a lens's contract-salvage
// invocation.
func ReformatLensName(lens string) string { return lens + ReformatLensSuffix }

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

	// PromptBudget is the largest prompt, in bytes, this agent will be handed
	// (0 = no limit, the default). Over it, the invocation is refused BEFORE the
	// process starts and the step is recorded as failed.
	//
	// It exists because the alternative is worse in both directions. Sending the
	// prompt anyway is what happens today: one reviewer came back with
	// `exit status 1: Prompt is too long` after a full round of wall clock, and a
	// context-limit refusal is indistinguishable from a broken agent in the run
	// summary. Silently trimming the material is worse still -- a reviewer shown
	// two thirds of a diff reports nothing about the rest, which reads exactly like
	// a clean bill of health, and the run can then converge over code nobody saw.
	//
	// So the budget refuses LOUDLY and early: no tokens are spent, the reason names
	// both sizes, and because a failed reviewer resets the clean-round streak, a
	// round that lost a reviewer this way cannot be mistaken for a clean one.
	//
	// Sizing it is per-agent and empirical, which is why there is no default: a
	// model's advertised context window is in tokens, this is bytes, and the
	// agentic session adds file reads and tool results on top of whatever fixpoint
	// sends. Set it below where that CLI actually refuses, not at its nominal
	// limit. target's own material cap is a separate, global bound on the collected
	// diff or listing; this one bounds the whole rendered prompt.
	PromptBudget int `yaml:"prompt_budget"`
	// CanEdit is a claim about what the COMMAND permits, and everything
	// downstream believes it: Validate keeps write-capable agents out of the
	// reviewer pool, and the fix-round trust gate exists precisely because the
	// coder edits with permission checks off. permissionBypassFlag below refuses
	// the one contradiction fixpoint can see from here -- a read-only claim made
	// by a command that grants the write tools anyway.
	CanEdit bool `yaml:"can_edit"`

	// Env controls what this agent's process can see of fixpoint's environment.
	Env AgentEnv `yaml:"env"`

	// Usage tells fixpoint how to read what this invocation actually cost, from
	// the CLI's own machine-readable output.
	Usage AgentUsage `yaml:"usage"`
}

// AgentUsage describes where a CLI reports its token usage and cost, so fixpoint
// can account for a run without knowing anything about the CLI.
//
// Measuring at fixpoint's own boundary does not work, and not by a small margin.
// The bytes fixpoint hands over and gets back are the two ends of an agentic
// session that reads files, calls tools, and takes many model turns in between --
// none of which crosses this process. A one-word probe whose boundary I/O was ~30
// bytes reported 21,072 tokens and $0.06: a bytes/4 estimate would have called it
// zero. Only the CLI knows, and every CLI worth using will say if asked.
//
// So fixpoint asks, and the ASKING is configuration rather than code: which flag
// switches a CLI to machine-readable output, and where the numbers sit in it, is
// exactly the knowledge this file already exists to hold. A provider-agnostic tool
// cannot enumerate that centrally without growing a per-vendor table that goes
// stale; whoever wrote `command` knows it today.
//
// Cost is reported when the CLI computes it and left empty otherwise -- a
// subscription-authenticated CLI has no per-request price to report, and inventing
// one from a hardcoded price list would be a guess presented as an audit figure.
// There is no pricing API to consult; published rates change, differ per auth mode,
// and say nothing about how much of a prompt was served from cache. A number the
// CLI computed from its own billing is worth more than one fixpoint derived.
type AgentUsage struct {
	// Format is how to read stdout: "json" (one object) or "jsonl" (one object
	// per line -- the LAST value found for each path wins, which is how a
	// streaming CLI's final totals are picked up). Empty disables usage parsing.
	Format string `yaml:"format"`

	// Text is the dotted path to the agent's actual reply inside the envelope.
	// Required with a format: fixpoint replaces the raw stdout with this before
	// the output contract is extracted, so machine-readable mode stays invisible
	// to everything downstream. Without it the envelope itself would reach the
	// extractor and every round would fail to parse.
	Text string `yaml:"text"`

	// Dotted paths to the numbers. Each is optional: a CLI that reports tokens
	// but not cost simply leaves cost_usd unset, and the scoreboard shows no cost
	// for it rather than a fabricated one.
	InputTokens      string `yaml:"input_tokens"`
	OutputTokens     string `yaml:"output_tokens"`
	CacheReadTokens  string `yaml:"cache_read_tokens"`
	CacheWriteTokens string `yaml:"cache_write_tokens"`
	CostUSD          string `yaml:"cost_usd"`

	// ErrorStatus is the dotted path to the HTTP-ish status the CLI reports when
	// its PROVIDER refused the request, rather than the CLI itself failing. Also
	// optional: without it a provider refusal is just a non-zero exit.
	//
	// It exists because those two are the same exit code and mean opposite things.
	// A real run ended with `coder failed: exit status 1`, which reads as a defect
	// in the coder -- while the CLI's own envelope said 429 and "You've hit your
	// session limit · resets 8:20pm". One is a bug to investigate, the other is a
	// clock to wait on, and the summary could not tell them apart. See
	// agent.Result.ProviderStatus.
	ErrorStatus string `yaml:"error_status"`
}

// Enabled reports whether this agent's output carries usage fixpoint can read.
func (u AgentUsage) Enabled() bool { return u.Format != "" }

// validate rejects a usage block that would silently do nothing, or worse. The
// missing-text case is the dangerous one: the CLI would be switched to
// machine-readable output while fixpoint kept extracting the contract from the
// raw envelope, so EVERY round would fail to parse -- an error whose cause is two
// files away from its symptom. Catch it at startup instead.
func (u AgentUsage) validate(name string) error {
	if u.Format == "" {
		// No usage block. A stray path without a format is a typo worth naming,
		// since it reads as configured and does nothing.
		if u.Text != "" || u.InputTokens != "" || u.OutputTokens != "" ||
			u.CacheReadTokens != "" || u.CacheWriteTokens != "" || u.CostUSD != "" ||
			u.ErrorStatus != "" {
			return fmt.Errorf("agents.%s: usage paths are set but usage.format is empty, so none of them are read; set format to %s",
				name, strings.Join(usageFormats, " or "))
		}
		return nil
	}
	if !slices.Contains(usageFormats, u.Format) {
		return fmt.Errorf("agents.%s: unknown usage.format %q (want %s)", name, u.Format, strings.Join(usageFormats, " | "))
	}
	if u.Text == "" {
		return fmt.Errorf("agents.%s: usage.format is set but usage.text is not; fixpoint would hand the raw envelope to the output-contract extractor and every round would fail to parse", name)
	}
	return nil
}

// Usage output formats.
const (
	UsageFormatJSON  = "json"
	UsageFormatJSONL = "jsonl"
)

var usageFormats = []string{UsageFormatJSON, UsageFormatJSONL}

// AgentEnv declares the environment an agent runs with. Everything not covered
// here is absent from the process.
//
// The environment is an exfiltration surface distinct from the filesystem, and the
// only one a container does not close: the agents' own credentials must be inside
// the container for the CLIs to work at all. A reviewer runs in a mode that denies
// EDITS, not READS -- on Linux a process can read its own /proc/self/environ, and
// any CLI with a shell tool can just run `env` -- so a prompt-injected reviewer can
// quote a value into a finding, which is logged and, in a fix run, echoed into the
// commit body. On-disk redaction only masks fixed-shape tokens, so a bare database
// password or an opaque internal token passes through unmasked.
//
// Filtering is possible even though fixpoint is provider-agnostic and cannot know
// what any given CLI needs, because it does not have to know: the agent's own file
// declares it, and whoever wrote its `command` is exactly who knows.
//
// This shrinks the blast radius rather than closing it. An agent that authenticates
// via an environment variable must still be given that variable, so its own
// credential stays reachable by the process that needs it -- but your GitHub token
// is no longer in the code reviewer.
type AgentEnv struct {
	// Pass names variables inherited from fixpoint's environment when set. A name
	// that is not set in fixpoint's environment is simply absent, not an error:
	// agents commonly accept either an env var or a config file for credentials.
	Pass []string `yaml:"pass"`
	// Set provides literal values, and overrides anything inherited via Pass.
	Set map[string]string `yaml:"set"`
	// InheritAll restores the old behavior of passing fixpoint's whole environment.
	// An escape hatch for a CLI whose requirements are not known, at the cost of
	// re-exposing every exported secret to that agent. fixpoint warns at run start.
	//
	// It is refused outright in a file resolved from inside the target, whatever the
	// operator asserted: see rejectProjectSuppliedInheritAll.
	InheritAll bool `yaml:"inherit_all"`
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
// template token: it is split on whitespace so "--model fable" becomes two argv
// elements, each field's placeholders are substituted, and a token referencing a
// placeholder that resolves to empty is dropped whole (so a flag and its value
// drop together).
//
// SECURITY: the split happens BEFORE substitution, so a placeholder's value is
// always exactly one argv element however it is spelled. Splitting afterwards
// would let a model/effort value carry its own arguments -- `model: claude-opus-5
// --setting-sources target` would append a second --setting-sources after the one
// config/agents/claude.yaml hardcodes, and the CLI's last-flag-wins parsing would
// load the target's settings, hooks and MCP servers after all. Validate refuses
// such a value outright (see validCommandValue); this keeps the refusal from
// being the only thing between a config field and the argument list.
func (a Agent) Argv() []string {
	values := a.placeholderValues()
	var argv []string
	for _, tok := range a.Command {
		fields := strings.Fields(tok)
		subs := make([]string, 0, len(fields))
		drop := false
		for _, f := range fields {
			sub, keep := expandToken(f, values)
			if !keep {
				drop = true
				break
			}
			subs = append(subs, sub)
		}
		if drop {
			continue
		}
		argv = append(argv, subs...)
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

// permissionBypassFlags are the standalone switches that hand a coding CLI's
// write tools to the model: either by turning the permission system off outright,
// so that every tool request -- reads, shell, and writes alike -- is
// auto-approved, or by selecting a preset that auto-approves the edits. The list
// is short and literal on purpose: it names the flags fixpoint's own agent files
// use or could plausibly grow, and a CLI it does not know about simply is not
// checked. Missing one costs nothing that is not already the status quo; a false
// positive would reject a working config, so nothing goes in here on suspicion.
var permissionBypassFlags = map[string]string{
	"--dangerously-skip-permissions":             "claude, agy",
	"--dangerously-bypass-approvals-and-sandbox": "codex",
	"--yolo": "gemini-cli, qwen-code",
	// codex's full-auto preset: workspace-write sandbox plus on-failure approvals,
	// i.e. edits inside the target land without ever being asked about.
	"--full-auto": "codex",
}

// writeGrantingModes are the flags a CLI spells as a mode VALUE rather than as a
// standalone switch, mapped to the values that let the model write files. Same
// admission rule as permissionBypassFlags: only values the CLI documents as
// permitting writes, never a value that merely looks permissive.
//
// A mode is enough on its own -- it need not be the full bypass. A reviewer whose
// only granted tools are Edit and Write is exactly the outcome can_edit: false is
// there to prevent, and in -p/exec mode there is no interactive approver left to
// stop it.
// codex's two write-permitting sandbox values. Named because each is spelled in
// three places -- the flag map, its short alias, and the `-c key=value` setting
// map -- and a typo in any one of them would silently stop rejecting a
// write-granting argument on a reviewer that claims can_edit: false.
const (
	sandboxWorkspaceWrite   = "workspace-write"
	sandboxDangerFullAccess = "danger-full-access"
)

var writeGrantingModes = map[string][]string{
	// claude: bypassPermissions drops every check; acceptEdits auto-approves the
	// edit tools specifically, which reads like a safe middle ground and is not one.
	"--permission-mode": {"bypassPermissions", "acceptEdits"},
	// codex: read-only is the enforced no-write sandbox that earns a read-only
	// claim (see config/agents/codex.yaml); both other modes permit writes.
	// Matched under the short spelling too -- `-s` is codex's own alias, and the
	// value is specific enough that no other CLI collides with it.
	"--sandbox": {sandboxWorkspaceWrite, sandboxDangerFullAccess},
	"-s":        {sandboxWorkspaceWrite, sandboxDangerFullAccess},
	// agy: --mode takes only plan or accept-edits (see config/agents/agy.yaml).
	"--mode": {"accept-edits"},
	// gemini-cli, qwen-code: the value form of --yolo, plus its edits-only preset.
	"--approval-mode": {"yolo", "auto_edit"},
}

// writeGrantingConfigSettings are the same grants spelled as a SETTING rather
// than as a flag, for a CLI that exposes its whole config file on the command
// line -- codex's `-c key=value` / `--config key=value`. A setting override
// outranks nothing here: `codex exec --sandbox read-only -c sandbox_mode=danger-full-access`
// runs unsandboxed, so checking only the flag spelling leaves the check evadable
// by writing the same grant one argument differently.
//
// Same admission rule as the maps above: only settings whose documented values
// permit writes. What this cannot see is indirection -- `-c profile=<name>`, or a
// profile chosen with --profile, selects a sandbox_mode from ~/.codex/config.toml,
// which is not on argv and is not fixpoint's to read. can_edit: false remains a
// claim about the whole command; argv is only the part fixpoint can check.
var writeGrantingConfigSettings = map[string][]string{
	// codex: the setting --sandbox sets. read-only is the enforced no-write value
	// that earns a read-only claim; both others permit writes.
	"sandbox_mode": {sandboxWorkspaceWrite, sandboxDangerFullAccess},
}

// dataScopeFlags are the flags whose value is a directory the CLI merely adds to
// what the model may READ. That is the one argv position where a directory under
// the target is not the substitution TargetSuppliedArg guards against: no code is
// executed from it and no file in it is read as the CLI's own configuration.
//
// Same admission rule as the maps above, and it matters more here because being
// wrong opens a hole rather than closing one: only flags documented to take a
// plain scope directory go in, and a CLI fixpoint does not know about keeps the
// guard.
var dataScopeFlags = map[string]string{
	"--add-dir":             "claude, agy",
	"--include-directories": "gemini-cli, qwen-code",
}

// configOverridePrefixes are the config-override spellings that pack the flag and
// the assignment into ONE argument. The separated form ("-c" "k=v") needs no
// prefix: the assignment is a token of its own and is matched as one.
var configOverridePrefixes = []string{"--config=", "--config", "-c"}

// configSetting reports the setting a token assigns, in every spelling a
// config-override flag accepts: "sandbox_mode=v" (the argument after -c/--config),
// "-csandbox_mode=v", "-c=sandbox_mode=v", "--config=sandbox_mode=v", and
// "--configsandbox_mode=v".
// Quotes are stripped because the value is TOML, so a string may arrive quoted.
func configSetting(tok string) (key, val string, ok bool) {
	for _, p := range configOverridePrefixes {
		if strings.HasPrefix(tok, p) && len(tok) > len(p) {
			// The "=" between a flag and its value is a separator, not part of the
			// key: clap accepts it on the short spelling too, so `-c=sandbox_mode=v`
			// sets sandbox_mode exactly as `-c sandbox_mode=v` does. Leaving it on
			// would cut an empty key out of the token and hide the whole override.
			tok = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(tok, p)), "=")
			break
		}
	}
	key, val, ok = strings.Cut(tok, "=")
	if !ok {
		return "", "", false
	}
	unquote := func(s string) string {
		return strings.Trim(strings.TrimSpace(s), `"'`)
	}
	key, val = unquote(key), unquote(val)
	if key == "" {
		// Not a setting fixpoint can read -- and not a spelling any parser accepts
		// either. Reporting no key is right; silently treating it as the empty
		// setting is what let a grant through.
		return "", "", false
	}
	return key, val, true
}

// permissionBypassFlag returns the write-granting token in argv, or "" when there
// is none. Flags are matched in both spellings a CLI may accept (--flag=value and
// --flag value), because the check is worth nothing if it can be evaded by
// writing the same argument differently.
func permissionBypassFlag(argv []string) string {
	for i, tok := range argv {
		key, val, hasVal := strings.Cut(tok, "=")
		if _, ok := permissionBypassFlags[key]; ok {
			// A standalone switch spelled with an explicit =false is an opt-OUT, not
			// a grant: no CLI parser these flags belong to reads "false" as on (yargs
			// and Go's flag read it as off; clap, commander and argparse reject the
			// spelling outright and never start), so reporting it as a write grant
			// would be a false positive on a config that asked for the safe thing.
			// Only that one literal is excused -- the bare flag, =true, and any value
			// fixpoint cannot interpret all still count as a bypass.
			if !hasVal || !strings.EqualFold(val, "false") {
				return key
			}
			continue
		}
		if setting, value, isSetting := configSetting(tok); isSetting {
			for _, mode := range writeGrantingConfigSettings[setting] {
				if strings.EqualFold(value, mode) {
					return setting + "=" + mode
				}
			}
		}
		modes, ok := writeGrantingModes[key]
		if !ok {
			continue
		}
		if !hasVal && i+1 < len(argv) {
			val = argv[i+1]
		}
		for _, mode := range modes {
			if strings.EqualFold(val, mode) {
				// Report the flag and the offending value: with several accepted
				// values per flag, the flag name alone would not say which one.
				return key + " " + mode
			}
		}
	}
	return ""
}

// TargetSuppliedArg returns the first element of argv that names a file inside
// root, or "" when none does. Relative elements are resolved the way both the OS
// and the agent CLI itself resolve them: against the process working directory,
// which agent.Run sets to target.path -- so what this reports is the part of an
// agent command whose CONTENT the code under review gets to supply. argv[0] is
// the executable fixpoint execs; a later element is a script or config file the
// CLI reads, which for a `node ./reviewer.cjs` style command is the same code
// path by one more step.
//
// Exported for the orchestrator's warning on the trust-asserted path; the
// refusal itself lives in Validate.
//
// A bare command name is measured by TargetSuppliedPATHDir instead: it names no
// path, so PATH -- not argv -- decides which file it runs.
//
// An element counts as a path when it is absolute or explicitly relative
// ("./x", "../x"), or when it contains a separator and something sits there now.
// Both the whole element and, for a packed option ("--require=./hook.js"), the
// value after the "=" are measured -- see argPathSpellings.
// The one exemption is an existing DIRECTORY named as the value of a data-scope
// flag (see dataScopeFlags), in any spelling. The existence requirement is what
// keeps the separator-bearing strings that are not paths at all out of the answer
// -- an OpenRouter model id such as moonshotai/kimi-k2 is one, and refusing it
// would reject a working config. That makes the plain "dir/file" spelling
// best-effort, since existence is measured before any `gh pr checkout`: a path
// only the PR creates is missed there, while the explicitly relative form -- how
// a target-local script is normally written, and the only form that can name
// argv[0], which LookPath has already proven exists -- is always caught outside
// that one exemption.
func TargetSuppliedArg(argv []string, root string) string {
	for i, tok := range argv {
		// Whether a directory here is only a read scope is decided by the element
		// BEFORE it, which pathLikeArg cannot see. Matched as a whole token: the
		// packed "--add-dir=./sub" spelling carries its value itself, so reading a
		// prefix off it would exempt the NEXT element, which the flag says nothing
		// about. argv[0] has no predecessor and so is never exempt.
		dataScope := false
		if i > 0 {
			_, dataScope = dataScopeFlags[argv[i-1]]
		}
		for _, cand := range argPathSpellings(tok, dataScope) {
			if !pathLikeArg(cand.tok, root, cand.dataScope) {
				continue
			}
			// Only a relative element resolves against the working directory; joining
			// root onto an absolute one would fabricate a path under the target and
			// report every absolute command as target-supplied.
			p := cand.tok
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			if within(p, root) {
				return tok
			}
			// A path that is outside the target lexically can still LAND inside it
			// through a symlink, which withinTree resolves -- but withinTree answers
			// "inside" for a path it cannot canonicalize at all, so ask it only about
			// one that exists. Otherwise an absolute argument naming no file
			// (--config /etc/absent.toml) would be reported as PR-supplied.
			if _, err := os.Lstat(p); err == nil && withinTree(p, root) {
				return tok
			}
		}
	}
	return ""
}

// argSpelling is one substring of an argv element that may name a filesystem
// path, with the data-scope verdict that applies to THAT substring.
type argSpelling struct {
	tok       string
	dataScope bool
}

// argPathSpellings returns the substrings of tok to measure against the target:
// the whole token, plus -- when tok is an option that PACKS its value with "=" --
// the value after the first "=".
//
// The packed form is the hole the whole-token test cannot see. `--require=./hook.js`
// contains a separator but names no file, so pathLikeArg's existence rule drops it
// and the ./hook.js inside it is never measured -- yet node loads that hook out of
// the post-checkout worktree and runs it as part of the agent process. Reading the
// value out is the same reasoning permissionBypassFlag already applies to packed
// flag spellings: a check worth anything cannot be evaded by writing the same
// argument one character differently.
//
// The data-scope exemption is re-derived from the packed KEY rather than inherited
// from the token's predecessor: `--add-dir=./sub` says the same thing about ./sub
// that `--add-dir ./sub` does, while the element after it is still unexempt (that
// is why the caller matches dataScopeFlags on the whole previous token).
func argPathSpellings(tok string, dataScope bool) []argSpelling {
	out := []argSpelling{{tok: tok, dataScope: dataScope}}
	// Only an OPTION packs a value this way. Without this the "key=value" split
	// would also fire on a bare argument that merely contains "=", measuring a
	// suffix no CLI reads as a path of its own.
	if !strings.HasPrefix(tok, "-") {
		return out
	}
	key, val, ok := strings.Cut(tok, "=")
	if !ok || val == "" {
		return out
	}
	_, scope := dataScopeFlags[key]
	return append(out, argSpelling{tok: val, dataScope: scope})
}

// checkTargetSuppliedExecutable refuses an agent command the reviewed material
// could choose, for an untrusted target.
//
// The two spellings are gated differently, on purpose:
//
//   - An EXPLICIT target-relative element (./agent.sh) is refused in pr mode
//     only. Elsewhere it is a deliberate choice an operator wrote down -- a
//     project shipping its own agent wrapper is a supported shape -- and a
//     config the TARGET supplies is already gated by the project-supplied-policy
//     rule, which needs -trusted-bundle. In pr mode it is different in kind:
//     `gh pr checkout` replaces the tree after validation, so the file that runs
//     is not the file that was checked.
//   - A BARE name resolved through a PATH entry inside the target is refused in
//     EVERY untrusted mode. Nobody wrote that path down: the operator typed
//     "claude" and PATH silently turned it into the reviewed repository's file.
//     A directory review-only run asserts no trust at all, which made it the
//     least protected mode and the easiest to reach (review run
//     20260819-104919).
func (c *Config) checkTargetSuppliedExecutable(name string, argv []string) error {
	if c.Loop.TrustedTarget || c.Loop.AllowUntrustedFix {
		return nil
	}
	if c.Target.Mode == ModePR {
		if tok := TargetSuppliedArg(argv, c.Target.Path); tok != "" {
			return fmt.Errorf("agents.%s: command element %q resolves inside target %s, and in mode pr that path's content is the PR's -- `gh pr checkout` writes the branch before the first round, so fixpoint would execute PR-authored code as the agent process, with the credentials this agent declares (env.pass/env.set) and before any reviewer sandbox. Point the command at a binary outside the target (a bare name on PATH, or an absolute path), or pass -trusted-target/-allow-untrusted-fix if you trust the PR author", name, tok, c.Target.Path)
		}
	}
	if dir := TargetSuppliedPATHDir(argv[0], c.Target.Path); dir != "" {
		if c.Target.Mode == ModePR {
			return fmt.Errorf("agents.%s: command %q is a bare name resolved through PATH, and PATH entry %s lies inside target %s -- in mode pr `gh pr checkout` writes the PR's content there before the first round, so the PR can supply or shadow that executable and fixpoint would execute PR-authored code as the agent process, with the credentials this agent declares (env.pass/env.set) and before any reviewer sandbox. Give the command an absolute path outside the target, drop that entry from PATH, or pass -trusted-target/-allow-untrusted-fix if you trust the PR author", name, argv[0], dir, c.Target.Path)
		}
		return fmt.Errorf("agents.%s: command %q is a bare name resolved through PATH, and PATH entry %s lies inside target %s -- the reviewed material can supply that executable, or shadow one further down PATH with a file of the same name, and fixpoint would run it as the agent process with the credentials this agent declares (env.pass/env.set). Give the command an absolute path outside the target, drop that entry from PATH, or pass -trusted-target if this checkout is yours", name, argv[0], dir, c.Target.Path)
	}
	return nil
}

// TargetSuppliedPATHDir returns the PATH entry that lies inside root, or "" when
// bin is not resolved through PATH or no entry does. It answers the question
// TargetSuppliedArg cannot: a BARE command name (no separator) names no path at
// all, so nothing in argv reveals that the file behind it is the target's.
//
// exec.LookPath proves a bare name resolves to SOME file now, but it is re-resolved
// against the same PATH at every invocation, and in mode pr `gh pr checkout` has
// rewritten the target by then. An operator PATH carrying a directory inside the
// target -- /repo/bin, a repo-local toolchain shim -- therefore lets the PR supply
// that executable outright, or shadow one resolved further down PATH by adding a
// file of the same name. Either way fixpoint execs PR-authored code as the agent
// process itself, which is exactly what the argv gate refuses.
//
// The whole PATH is measured rather than only where the name resolves today,
// because the shadowing case is the one where today's resolution is outside the
// target and tomorrow's is not.
//
// Exported alongside TargetSuppliedArg for the orchestrator's trust-asserted
// warning; the refusal itself lives in Validate.
func TargetSuppliedPATHDir(bin, root string) string {
	if bin == "" || filepath.IsAbs(bin) || strings.ContainsRune(bin, '/') || strings.ContainsRune(bin, filepath.Separator) {
		// Not a PATH lookup: an absolute or separator-bearing command names its file
		// directly, and TargetSuppliedArg already measures that spelling.
		return ""
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		// A relative entry -- including the empty one, which means the working
		// directory -- can never decide what runs: exec.LookPath reports ErrDot for a
		// name resolved through one, so the LookPath check in Validate has already
		// rejected the command and exec.Cmd would refuse to start it. Skipping them
		// also keeps the very common trailing-colon PATH from reading as "the target
		// is on PATH" whenever fixpoint is launched from inside the target.
		if !filepath.IsAbs(dir) {
			continue
		}
		// Lexically first, so an entry the PR has yet to CREATE (/repo/bin in a tree
		// that has no bin/ yet) counts; then resolved, for an entry that is a symlink
		// into the target. Same order and same reason as TargetSuppliedArg.
		if within(dir, root) {
			return dir
		}
		if _, err := os.Lstat(dir); err == nil && withinTree(dir, root) {
			return dir
		}
	}
	return ""
}

// pathLikeArg reports whether tok should be read as a filesystem path at all.
// See TargetSuppliedArg for why existence decides the ambiguous spelling.
// dataScope says tok sits in the one position where a directory is merely a read
// scope rather than something the command can run.
//
// Both '/' and filepath.Separator count, as in Validate's binary resolution:
// Windows accepts a forward slash too, so checking only the native separator
// would let "./reviewer.sh" pass unexamined there.
func pathLikeArg(tok, root string, dataScope bool) bool {
	if tok == "" {
		return false
	}
	explicit := filepath.IsAbs(tok)
	for _, p := range []string{"./", "../", `.\`, `..\`} {
		if strings.HasPrefix(tok, p) {
			explicit = true
			break
		}
	}
	if !explicit && !strings.ContainsRune(tok, '/') && !strings.ContainsRune(tok, filepath.Separator) {
		return false
	}
	p := tok
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	st, err := os.Stat(p)
	if err != nil {
		// Nothing there to inspect. An explicit spelling stays path-like -- a file
		// only the PR creates is precisely the case to catch -- while the ambiguous
		// "dir/file" one needs the file to exist to count as a path at all.
		return explicit
	}
	// A directory behind a data-scope flag (--add-dir some/sub, --add-dir ./sub) is
	// not the substitution this guards against: that flag widens what the model may
	// read and nothing more, and a subdirectory of the tree under review is a normal
	// thing to widen it to. The test comes after the explicit spellings rather than
	// inside the ambiguous branch so every spelling of that argument is exempted.
	//
	// Anywhere else a directory IS a code path and keeps the guard: `node ./lib`
	// executes ./lib/index.js, and a CLI handed a config DIRECTORY reads what it
	// finds there. Both are PR-authored content running as the agent process once
	// `gh pr checkout` has written the branch, which is exactly what this refuses.
	if dataScope && st.IsDir() {
		return false
	}
	return true
}

// Commit policies: how a round's per-fix commits are grouped. The coder always
// works one issue at a time; only the grouping differs.
const (
	// CommitPerFix keeps one commit per fixed issue -- each individually
	// revertable, each with the verify gate that passed it.
	CommitPerFix = "per_fix"
	// CommitPerRound squashes a round's fix commits into one round commit. The
	// default, and what fixpoint has always produced.
	CommitPerRound = "per_round"
	// CommitPerRun squashes every round into a single commit for the whole run.
	CommitPerRun = "per_run"
)

var commitPolicies = []string{CommitPerFix, CommitPerRound, CommitPerRun}

// DefaultMaxFinalPasses bounds the repeating half of the closing round when
// loop.max_final_passes is unset. Exported because a Loop built in code rather
// than loaded from YAML never passes through the defaulting step, and a zero there
// must not silently mean "run the closing phase zero times".
//
// One, lowered from two on measurement. Pass 2's premise was "confirm the fix did
// not open something new", and in a seven-round run it did not do that: its 7
// findings were 4 critiques of the tests pass 1 had just written, 2 restatements of
// what pass 1 had already reported, and 1 real regression -- which pass 1's OWN fix
// had introduced. A pass whose main yield is repairing the previous pass is not
// converging on the code; the verify gate (which runs the project's tests, with
// -race here) is the check that catches what a fix broke, and it runs per fix.
const DefaultMaxFinalPasses = 1

// Loop controls how the review->fix cycle iterates, terminates, and commits,
// and holds the trust gates that permit fix rounds at all.
type Loop struct {
	MaxIterations int `yaml:"max_iterations"`

	// MaxFinalPasses caps how many times the actionable half of the closing round
	// repeats (0 = the default below). The phase already stops on its own the
	// moment a pass finds nothing or fixes nothing, so this only ever catches a
	// lens that never runs out of things to say -- and review-tests IS that lens:
	// coverage can always be wanted more of.
	//
	// It used to borrow max_iterations, which conflated two unrelated budgets. A
	// measured run shows why that is too loose: the closing phase spent 51 minutes
	// over two passes, and pass 2 still surfaced four brand-new issues, because
	// each pass reviewed the tests the PREVIOUS pass had just written. Every
	// recurring issue id in that run came from this phase -- a real bug fixed in
	// the loop, then re-opened as "the test for that fix is flaky", then as "the
	// test for the test". The loop's own rounds did not repeat themselves at all.
	//
	// One is the default (see DefaultMaxFinalPasses above for the measurement that
	// lowered it from two): a single pass to fix what the finished tree still needs.
	// A second pass is already largely reviewing the first pass's tests, which is
	// where the yield goes negative.
	MaxFinalPasses int `yaml:"max_final_passes"`

	// FinalSkipRunEdits hides files THIS RUN wrote from the CLOSING round's review
	// material: a glob list, matched against the paths the run's own commits changed.
	// Empty (the default) hides nothing, which is the behavior every project had
	// before this existed.
	//
	// It is aimed at one specific non-convergence, not at self-review in general.
	// Reviewing what an earlier round changed is productive inside the loop -- that is
	// how a fix's own bug gets caught -- so this deliberately does not apply there.
	// The closing round is different only because review-tests lives there: it asks
	// for a test, the coder writes one, and the next pass reviews THAT TEST rather
	// than the code. A measured run produced 7 such findings out of 14 across two
	// closing passes, every one of them against a test file the run had committed
	// minutes earlier.
	//
	// Set it to the test-file shape of the project's language ("**/*_test.go",
	// "**/test_*.py", "**/*.spec.ts"); see target.Collector.HideRunEdits for why the
	// blunter "hide everything this run touched" was measured and rejected. Patterns
	// use the same syntax as target.exclude.
	FinalSkipRunEdits []string `yaml:"final_skip_run_edits"`

	// MaxFindingsPerRound caps how many ISSUES a round hands to the coder
	// (0 = unlimited, the default). Ordered worst-severity-first; the overflow is
	// marked deferred and re-surfaces in later rounds, gaining a severity tier each
	// time so the tail cannot be starved.
	//
	// It defaults to unlimited because the reason it existed is gone. The cap was
	// sized to ONE coder session's timeout -- ~17 issues blew 30m, ~8 fit -- and the
	// coder now gets exactly one issue per session, so no amount of findings can
	// overrun a session. What is left is a cost lever: set it to bound how much a
	// round spends, knowing the remainder waits for a later round.
	//
	// It was never a free bound. In one five-round run a cap of 8 deferred 18 issues,
	// and in the CLOSING round -- where no later round follows -- it dropped 5
	// coverage gaps outright.
	MaxFindingsPerRound int  `yaml:"max_findings_per_round"`
	ReviewOnly          bool `yaml:"review_only"`

	// CommitPolicy groups the round's per-fix commits: per_fix keeps them, per_round
	// squashes each round's into one, per_run squashes the whole run into one.
	//
	// It is only ever a GROUPING. The coder is handed one issue per session under
	// every policy, because that is the only way a commit can honestly claim to
	// contain one fix: a session given eight issues edits files for all eight at
	// once, and nothing in its report says which change served which issue. Working
	// one at a time also means the verify gate names the fix that broke the build
	// rather than the round, and every fix is revertable on its own.
	//
	// per_fix is the default because it is the granularity the work happened at, and
	// squashing is information-destroying: a round commit cannot tell you which of
	// its eight fixes broke something. Squash afterwards if you want fewer commits.
	CommitPolicy      string `yaml:"commit_policy"`
	CommitMessage     string `yaml:"commit_message"`
	CleanRoundsToStop int    `yaml:"clean_rounds_to_stop"`

	// AllowUntrustedFix permits fix rounds in pr mode. A PR is
	// externally-authored code: its content flows through reviewer findings
	// into the coder prompt, and the coder edits files with permission checks
	// disabled, so a malicious PR can steer it via prompt injection. Off by
	// default.
	//
	// Set ONLY by the -allow-untrusted-fix flag; see TrustedTarget for why
	// neither trust field is readable from YAML.
	AllowUntrustedFix bool `yaml:"-"`

	// TrustedTarget asserts that a directory/git-diff target contains only code
	// the operator trusts. Fix rounds run the coder with permission checks
	// disabled and NOT confined to target.path, so any prompt-injection payload
	// in reviewed content could steer it into writing anywhere on the machine.
	// That risk is not unique to pr mode: a directory/git-diff tree can hold
	// vendored deps, a fetched base_ref, or a cloned third-party project. So
	// fix rounds in those modes are refused unless this is set (fail-closed by
	// default); pr mode uses AllowUntrustedFix instead.
	//
	// Set ONLY by the -trusted-target flag: `yaml:"-"` is load-bearing SECURITY,
	// not style. Bundles are shadowable and the FIRST search location is
	// <project>/config -- the target's own directory -- so a YAML-readable trust
	// field lets the code under review assert that it is trustworthy. One line in
	// a hostile repository's own config would then authorize both executing the
	// agent definitions it ships and running the write-capable coder, with no
	// operator involvement. The two gates that consult these fields
	// (allowProjectSuppliedPolicy and checkFixTrust) exist precisely to defend
	// against target-supplied configuration, so their input must come from the
	// invocation, which the target cannot influence.
	//
	// Keeping the keys out of the struct also makes the attempt LOUD rather than
	// ignored: the decoder runs with KnownFields(true), so a config setting either
	// key fails to load. rejectTrustKeys turns that into a message that explains
	// the flag, but the type is what makes silent acceptance impossible.
	//
	// This is also what makes "trust is asserted per invocation, never inherited"
	// true. A trust default in an operator-owned bundle would be no safer: it would
	// apply to every project the operator ever runs it in, including the next
	// untrusted repository they clone.
	TrustedTarget bool `yaml:"-"`

	// TrustedBundle asserts ONLY that the bundle files this run was built from may
	// be executed and sent to agents, even though they were resolved from inside
	// the target -- see Loaded.ProjectSuppliedPolicy. It is the narrow half of
	// TrustedTarget, and it exists because the two claims are separable and, for a
	// pull request, have different answers.
	//
	// The bundle is resolved and read ENTIRELY before Prepare: LoadBundle resolves
	// every agent and prompt path eagerly, and Orchestrator.New parses each prompt
	// template at construction. So in pr mode the files this assertion covers are
	// the ones on the pre-checkout tree -- the operator's own commit -- while
	// `gh pr checkout` replaces the worktree afterwards. A run that reviews its own
	// repository's pull requests therefore needs to trust its bundle without
	// trusting the branch, which is exactly what asserting TrustedTarget for it
	// would get wrong: that flag also downgrades the pr-mode refusals of a
	// target-relative agent command (validateAgents) and of externally-defined
	// content filters and diff drivers (orchestrator.guardActivatableConfig), both
	// of which are about content that lands only AFTER the checkout.
	//
	// TrustedTarget implies this: trusting the whole target trusts its bundle.
	// Nothing else consults it, and it never permits a fix round.
	//
	// Set ONLY by the -trusted-bundle flag, `yaml:"-"` for exactly the reason given
	// on TrustedTarget: the first bundle search location is the target's own
	// directory, so a YAML-readable form would let a bundle authorize its own
	// execution.
	TrustedBundle bool `yaml:"-"`
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
	// Redact holds extra Go regular expressions whose matches are masked in every
	// persisted artifact and log line, on top of the built-in credential shapes.
	//
	// The built-in rules recognize SHAPES (sk-ant-..., ghp_..., a JWT, a
	// key/secret/token/password assignment), so a site's own opaque token -- an
	// x-internal-auth header, a bare bearer string in a vendor format, a
	// connection URL under a name that looks like nothing -- matches none of them
	// and lands in the .raw log verbatim. No shape list can close that; only the
	// operator knows what their secrets look like. A pattern with a capture group
	// keeps group 1 and masks the rest of the match, which is how a site-specific
	// key gets masked while the surrounding line stays readable.
	Redact []string `yaml:"redact"`
}

// RedactPatterns compiles logs.redact. Validate rejects a config whose patterns
// do not compile, so a run installs these once at startup and can treat a
// failure here as impossible.
func (l Logs) RedactPatterns() ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(l.Redact))
	for i, p := range l.Redact {
		re, err := redactPattern(i, p)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

// redactPattern compiles one logs.redact entry and rejects the two ways an
// operator's pattern can make the artifacts worse instead of safer.
func redactPattern(i int, p string) (*regexp.Regexp, error) {
	if strings.TrimSpace(p) == "" {
		return nil, fmt.Errorf("logs.redact[%d] is empty; every entry must be a regular expression matching the secret to mask", i)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return nil, fmt.Errorf("logs.redact[%d] %q does not compile: %w", i, p, err)
	}
	// A pattern that matches "" matches at every position, so the replacement is
	// spliced between every character of every artifact -- the logs are destroyed
	// and, worse, they still look redacted. Rejected at startup because the damage
	// is only visible by reading a finished run's artifacts.
	if re.MatchString("") {
		return nil, fmt.Errorf("logs.redact[%d] %q matches the empty string, so it would match at every position and mask the whole of every artifact; anchor it or require at least one character (e.g. use + instead of *)", i, p)
	}
	return re, nil
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

// ByteSize wraps a byte count for YAML ("64MB", "2GB", "512KB", or a bare
// integer of bytes). Suffixes are 1024-based -- the numbers gate file sizes and
// disk headroom, where the powers-of-two convention is what an operator's `df`
// and `du` already speak.
type ByteSize int64

// UnmarshalYAML parses "<number><KB|MB|GB>" (case-insensitive) or bare bytes.
func (b *ByteSize) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	t := strings.TrimSpace(strings.ToUpper(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(t, "GB"):
		mult, t = 1<<30, strings.TrimSuffix(t, "GB")
	case strings.HasSuffix(t, "MB"):
		mult, t = 1<<20, strings.TrimSuffix(t, "MB")
	case strings.HasSuffix(t, "KB"):
		mult, t = 1<<10, strings.TrimSuffix(t, "KB")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid byte size %q (want e.g. 64MB, 2GB, or bytes): %w", s, err)
	}
	*b = ByteSize(n * mult)
	return nil
}

// Int64 is the plain byte count.
func (b ByteSize) Int64() int64 { return int64(b) }

// String renders the size the way an operator wrote it: the largest suffix
// that divides it evenly, so refusal messages quote "64MB", not 67108864.
func (b ByteSize) String() string {
	n := int64(b)
	switch {
	case n >= 1<<30 && n%(1<<30) == 0:
		return fmt.Sprintf("%dGB", n/(1<<30))
	case n >= 1<<20 && n%(1<<20) == 0:
		return fmt.Sprintf("%dMB", n/(1<<20))
	case n >= 1<<10 && n%(1<<10) == 0:
		return fmt.Sprintf("%dKB", n/(1<<10))
	}
	return strconv.FormatInt(n, 10)
}

// Load reads, defaults, and validates a single configuration file, with no bundle
// resolution and no `extends`. A run uses LoadBundle instead; this is for the
// narrow case of checking one file on its own.
//
// There is deliberately no unvalidated variant. One existed so the CLI could load,
// then mutate the result with flag overrides, then validate -- an order every
// caller had to reproduce correctly, because an override changes what a valid
// configuration is. Overrides now belong to the compile step (see Overrides), so
// "loaded but not yet valid" is no longer a state any caller needs to hold.
func Load(path string) (*Config, error) {
	cfg, err := decodeFile(path)
	if err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Target.Path == "" {
		c.Target.Path = "."
	}
	// A create config has no lens list for a strategy to schedule, but the pool
	// checks live on the rotate/all branch of the strategy switch and a create run
	// uses the WHOLE pool by definition -- so "all" is not a guess, it is the only
	// value that means what a create run does.
	if c.IsCreate() && c.Roles.Review.Strategy == "" {
		c.Roles.Review.Strategy = StrategyAll
	}
	if c.IsImplement() {
		c.Implement.applyDefaults()
	}
	// Default only the absent (zero) case; a negative value is invalid operator
	// input, not "unset", and Validate rejects it rather than silently masking it
	// with a positive default.
	if c.Loop.MaxIterations == 0 {
		c.Loop.MaxIterations = 5
	}
	if c.Loop.MaxFinalPasses == 0 {
		c.Loop.MaxFinalPasses = DefaultMaxFinalPasses
	}
	if c.Loop.CleanRoundsToStop == 0 {
		c.Loop.CleanRoundsToStop = 1
	}
	if c.Loop.CommitPolicy == "" {
		c.Loop.CommitPolicy = CommitPerFix
	}
	if c.Loop.CommitMessage == "" {
		// Per-fix shaped, matching the default commit_policy. roundCommitMessage
		// substitutes a round-shaped header when a squashing policy is selected.
		c.Loop.CommitMessage = "fixpoint: {issue} — {title}"
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
	// A document is only meaningful where the material would otherwise be a
	// listing. In the git modes the material is a diff, and a silently inert
	// `document:` key would sit in the config looking like a narrowing that never
	// happens -- same rule as every other refused-inert key here.
	if c.Target.Document != "" && c.Target.Mode != ModeDirectory {
		return fmt.Errorf("target.document is set but target.mode is %q: a document target reviews one file as the material, which only mode directory supports", c.Target.Mode)
	}
	// A pr-mode run needs a number, but it does not need one YET when the
	// invocation said to take the branch's own pull request: that resolution runs
	// git and gh inside the target, so it has to wait for the target-integrity
	// preflight, which is long after validation (Orchestrator.resolvePR).
	//
	// The exemption covers a MISSING number, never a bad one. A negative value was
	// typed by somebody and is refused whatever else is set, so the two conditions
	// stay separate rather than collapsing into "not positive".
	if c.Target.Mode == ModePR && (c.Target.PR < 0 || (c.Target.PR == 0 && !c.Target.PRFromBranch)) {
		return fmt.Errorf("target.pr: mode pr needs a pull request number, got %d -- pass -pr <number>, or run from the branch the pull request is on and fixpoint resolves it", c.Target.PR)
	}
	// An empty base_ref means "review the unstaged working changes", which a fix
	// run can never see: it requires a clean working tree at start and commits
	// each round's edits, so the unstaged diff is empty by construction in round 1
	// and in every round after it. Reviewers would receive no material, report no
	// findings, and the run would exit 0 as "converged" having reviewed nothing.
	if c.Target.Mode == ModeGitDiff && !c.Loop.ReviewOnly && c.Target.BaseRef == "" {
		return errors.New("target.base_ref: a fix run in mode git-diff needs a base ref -- an empty base_ref reviews only unstaged working changes, and fix rounds require a clean working tree at start, so that diff is always empty; set target.base_ref, or use loop.review_only")
	}
	if c.Loop.MaxFindingsPerRound < 0 {
		return fmt.Errorf("loop.max_findings_per_round: must not be negative, got %d", c.Loop.MaxFindingsPerRound)
	}
	if !slices.Contains(commitPolicies, c.Loop.CommitPolicy) {
		return fmt.Errorf("loop.commit_policy: unknown policy %q (want %s)", c.Loop.CommitPolicy, strings.Join(commitPolicies, " | "))
	}
	if c.Loop.MaxIterations < 0 {
		return fmt.Errorf("loop.max_iterations: must not be negative, got %d", c.Loop.MaxIterations)
	}
	if c.Loop.MaxFinalPasses < 0 {
		return fmt.Errorf("loop.max_final_passes: must not be negative, got %d", c.Loop.MaxFinalPasses)
	}
	if c.Loop.CleanRoundsToStop < 0 {
		return fmt.Errorf("loop.clean_rounds_to_stop: must not be negative, got %d", c.Loop.CleanRoundsToStop)
	}
	if j := c.Roles.Judge; j.Agent != "" || j.Prompt != "" {
		if j.Agent == "" || j.Prompt == "" {
			return errors.New("roles.judge: both agent and prompt are required when either is set")
		}
	}
	if t := c.Roles.Triage; t.Agent != "" || t.Prompt != "" {
		if t.Agent == "" || t.Prompt == "" {
			return errors.New("roles.triage: both agent and prompt are required when either is set")
		}
		if c.Target.Mode != ModePR {
			return fmt.Errorf("roles.triage is set but target.mode is %q: conversations exist only on a pull request", c.Target.Mode)
		}
	}
	if e := c.Roles.Editor; e.Agent != "" || e.Prompt != "" {
		if e.Agent == "" || e.Prompt == "" {
			return errors.New("roles.editor: both agent and prompt are required when either is set")
		}
	}
	// The pipeline discriminators are mutually exclusive, and the refusal must
	// fire before either pipeline's own shape rules get a chance to complain
	// about the other's missing parts.
	if c.IsCreate() && c.IsImplement() {
		return errors.New("create.propose and roles.planner are both set; a config is one pipeline -- create drafts a design, implement builds one")
	}
	if err := c.validateCreate(); err != nil {
		return err
	}
	if err := c.validateImplement(); err != nil {
		return err
	}
	if c.Review.BlockAt != "" && !model.ValidSeverity(c.Review.BlockAt) {
		return fmt.Errorf("review.block_at: unknown severity %q (want %s)", c.Review.BlockAt, strings.Join(model.Severities, " | "))
	}
	if c.Review.RefuteAt != "" && !model.ValidSeverity(c.Review.RefuteAt) {
		return fmt.Errorf("review.refute_at: unknown severity %q (want %s)", c.Review.RefuteAt, strings.Join(model.Severities, " | "))
	}
	// A floor stricter than the block floor would leave a blocking finding the
	// refutation round never saw -- and applyJudgment only lets the judge drop a
	// blocker where refutation recorded doubt, so such a finding could never be
	// dropped however wrong it was. Refused at load rather than silently widened:
	// the operator asked for two settings that cannot both hold.
	if c.Review.Refute != "" && model.WorseSeverity(c.Review.RefuteFloor(), c.Review.BlockFloor()) {
		return fmt.Errorf("review.refute_at (%s) is stricter than review.block_at (%s): a blocking finding would skip refutation, and the judge may only drop a blocker that refutation doubted",
			c.Review.RefuteFloor(), c.Review.BlockFloor())
	}
	for i, g := range c.Loop.FinalSkipRunEdits {
		// An empty pattern compiles to ^$, which matches no real path -- so it would
		// sit in the config looking like an active rule and hide nothing. Refused for
		// the same reason logs.redact refuses one: a silently inert entry is worse
		// than a load error.
		if strings.TrimSpace(g) == "" {
			return fmt.Errorf("loop.final_skip_run_edits[%d]: empty pattern; remove the entry or give it a glob", i)
		}
	}

	if len(c.Roles.Review.Prompts) == 0 && !c.IsCreate() && !c.IsImplement() {
		return errors.New("roles.review.prompts: at least one review lens is required. A config with no lenses is a base meant to be inherited with `extends`, not run directly -- `fixpoint --list` marks which configs are runnable")
	}
	// An implement config with the strategy unset skips the check: it has no
	// panel -- no lenses (refused above), and the inherited pool is inert by
	// design (§7.3) -- so a strategy over nothing is nothing to validate, and
	// forcing a value would imply a schedule that never runs. A set strategy is
	// still validated, wherever it appears.
	if !c.IsImplement() || c.Roles.Review.Strategy != "" {
		if err := c.validateReviewStrategy(); err != nil {
			return err
		}
	}

	// once and final are contradictory schedules: one pins the lens to the first
	// round, the other holds it back until after the last. Silently honoring either
	// would run the lens somewhere the author did not ask for.
	for i, l := range c.Roles.Review.Prompts {
		if l.Once && l.Final {
			return fmt.Errorf("roles.review.prompts: lens %d (%s) sets both once and final; once runs it in round 1 only and final runs it in the closing round after the loop -- pick one", i, l.Prompt)
		}
	}

	// A fix run must have at least one recurring reviewer lens: one that is neither
	// once (round 1 only) nor final (after the loop). If EVERY lens is one of those,
	// the rounds in between have no reviewers, the fix is "verified" by an empty
	// review, and the run converges without any finding ever being re-checked.
	// review-only runs are a single round, so this does not apply -- and create
	// and implement runs have no lenses at all.
	if !c.Loop.ReviewOnly && !c.IsCreate() && !c.IsImplement() {
		recurring := false
		for _, l := range c.Roles.Review.Prompts {
			if !l.Once && !l.Final {
				recurring = true
				break
			}
		}
		if !recurring {
			return errors.New("roles.review.prompts: a fix run needs at least one recurring reviewer lens to verify each round, but every lens is once: true (round 1 only) or final: true (after the loop); clear one of those, or use loop.review_only")
		}
	}

	// Every referenced agent must be defined, its binary on PATH, and its
	// prompt_via valid.
	check := func(ctx, name string) error {
		a, ok := c.Agents[name]
		if !ok {
			return fmt.Errorf("%s: agent %q is not defined in the agents section", ctx, name)
		}
		if err := validPathIdent("agent", name); err != nil {
			return fmt.Errorf("%s: %w", ctx, err)
		}
		if err := validCommandValue(name, "model", a.Model); err != nil {
			return err
		}
		if a.PromptBudget < 0 {
			return fmt.Errorf("agents.%s: prompt_budget must not be negative, got %d (0 means no limit)", name, a.PromptBudget)
		}
		if err := validCommandValue(name, "effort", a.Effort); err != nil {
			return err
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
		//
		// Both '/' and filepath.Separator count: Windows accepts a forward slash
		// as a separator too, so checking only the native one would leave
		// "./agent.sh" validated against fixpoint's working directory there.
		if !filepath.IsAbs(bin) && (strings.ContainsRune(bin, '/') || strings.ContainsRune(bin, filepath.Separator)) {
			abs, err := filepath.Abs(filepath.Join(c.Target.Path, bin))
			if err != nil {
				return fmt.Errorf("agents.%s: resolving %q against target.path: %w", name, bin, err)
			}
			bin = abs
		}
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("agents.%s: binary %q not found on PATH", name, argv[0])
		}
		// SECURITY (mode pr): the LookPath check above proves the file exists NOW,
		// on the pre-checkout tree, but agent.Run re-resolves the same relative argv
		// against Cmd.Dir at EVERY invocation -- and in pr mode Prepare's
		// `gh pr checkout` has replaced that tree with the PR's by the time round 1
		// runs. A command element that lands inside target.path is therefore chosen
		// by the code under review: the PR ships (or edits) the script at that path
		// and fixpoint execs it as the agent process itself, with the credentials the
		// agent declares and no sandbox at all -- a stronger primitive than the
		// prompt injection the pr path is built to contain, and it fires even in the
		// shipped review-only configuration, which passes no other trust gate.
		//
		// Refused here rather than warned about, for the same reason
		// orchestrator.guardActivatableConfig refuses pr mode: the content that
		// decides what runs is not on disk yet, so there is nothing for a preflight
		// to inspect and "continue anyway" means running PR-authored code sight
		// unseen. With trust asserted the run proceeds and
		// orchestrator.warnTargetSuppliedCommand says what was accepted.
		// See checkTargetSuppliedExecutable for which spelling is refused where.
		if err := c.checkTargetSuppliedExecutable(name, argv); err != nil {
			return err
		}
		// A read-only claim contradicted by the command's own argv. Reviewers are
		// the agents that carry can_edit: false, they read untrusted content, they
		// run concurrently against the shared worktree, and a review-only run
		// asserts no trust flag at all -- so this declaration is the only thing
		// standing between a prompt injection and the write tools. Do not take it
		// on faith when the command hands the write tools to the model: whatever
		// remains (a --plan style flag) is the CLI's business to enforce, not
		// something fixpoint can check or has verified.
		if !a.CanEdit {
			if flag := permissionBypassFlag(argv); flag != "" {
				return fmt.Errorf("agents.%s: can_edit: false, but the command passes %s, which lets the agent write files without being asked; a read-only claim must be backed by the command itself (drop the flag, or use the CLI's enforced read-only sandbox such as codex --sandbox read-only), otherwise declare can_edit: true so the agent is kept out of roles.review", name, flag)
			}
		}
		if err := a.Usage.validate(name); err != nil {
			return err
		}
		// envName, not name: shadowing the agent name here put the offending
		// VARIABLE in the agents.<name> position, pointing at an agent that does
		// not exist and hiding which file to fix.
		for i, envName := range a.Env.Pass {
			if err := validEnvName(envName); err != nil {
				return fmt.Errorf("agents.%s: env.pass[%d] (%q): %w", name, i, envName, err)
			}
		}
		for k := range a.Env.Set {
			if err := validEnvName(k); err != nil {
				return fmt.Errorf("agents.%s: env.set: %w", name, err)
			}
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

	// A REVIEW-ONLY config needs no coder, and naming one is worse than pointless:
	// the coder is the one role that must be able to edit, so a review- config
	// declaring it puts a write-capable agent in a configuration whose entire
	// promise is that nothing modifies the target. A reader then has to work out
	// from the loop settings that it never runs.
	//
	// It stays REQUIRED for a fix run, and stays validated whenever it is present,
	// so a config that names a coder still cannot name a broken one.
	switch {
	case c.Roles.Coder.Agent == "" && c.Roles.Coder.Prompt == "":
		if !c.Loop.ReviewOnly && !c.IsCreate() {
			return errors.New("roles.coder.agent: required (a fix run needs a coder; set loop.review_only for a config that only reviews)")
		}
	case c.Roles.Coder.Agent == "" || c.Roles.Coder.Prompt == "":
		return errors.New("roles.coder: both agent and prompt are required when either is set")
	default:
		if err := check("roles.coder", c.Roles.Coder.Agent); err != nil {
			return err
		}
		if !c.Agents[c.Roles.Coder.Agent].CanEdit {
			return fmt.Errorf("roles.coder: agent %q has can_edit: false -- the coder must be able to edit files", c.Roles.Coder.Agent)
		}
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
	// Triage goes through the SAME check as every other invoked agent, not a
	// reduced one: it is exec'd like any other, so the pr-mode refusal of a
	// command element resolving inside target.path binds it too -- otherwise
	// triage would be one more way to run PR-authored code as a fixpoint agent.
	// A triage agent that is not also in the reviewer pool has no other place to
	// be validated.
	if t := c.Roles.Triage; t.Agent != "" {
		if err := check("roles.triage", t.Agent); err != nil {
			return err
		}
		// Same can_edit rule as the judge below, and here it is the load-bearing half of the feature:
		// triage reads comments written by anyone who can reach the pull request and
		// decides what work they commission. An agent that could also edit would let
		// that text reach the tree without passing through the coder, the verify gate
		// and a commit -- which are the three things that make a commissioned change
		// reviewable.
		if c.Agents[t.Agent].CanEdit {
			return fmt.Errorf("roles.triage.agent: %q declares can_edit; triage decides what untrusted comments commission and must not be able to act on them itself", t.Agent)
		}
	}
	// The judge goes through the SAME check as every other invoked agent, not a
	// reduced one. Two of check's rules are security controls the judge needs most:
	// the pr-mode refusal of a command element resolving inside target.path (the
	// judge would otherwise be a second way to exec PR-authored code as a fixpoint
	// agent), and the permission-bypass cross-check that keeps a can_edit: false
	// claim from being contradicted by the argv. The shipped configs pin the judge
	// to an agent that is also in the reviewer pool and so was checked there; a
	// judge-only agent has no other place to be validated.
	if e := c.Roles.Editor; e.Agent != "" {
		if err := check("roles.editor", e.Agent); err != nil {
			return err
		}
		// The editor is the single voice the deliverable speaks with, fed
		// exclusively by model-written text about an untrusted assignment. Same
		// rule, same reason as the judge: the decision-maker must not be able to
		// act on the tree it decides about.
		if c.Agents[e.Agent].CanEdit {
			return fmt.Errorf("roles.editor.agent: %q declares can_edit; the editor writes the deliverable through fixpoint, never files -- it must be read-only", e.Agent)
		}
	}
	if p := c.Roles.Planner; p.Agent != "" {
		if err := check("roles.planner", p.Agent); err != nil {
			return err
		}
		// The planner decides what forty coder sessions will build, from an
		// untrusted document. Same rule, same reason as the judge and the editor:
		// the thing that decides must not be the thing that can act.
		if c.Agents[p.Agent].CanEdit {
			return fmt.Errorf("roles.planner.agent: %q declares can_edit; the planner only decomposes -- it must be read-only", p.Agent)
		}
	}
	if j := c.Roles.Judge; j.Agent != "" {
		if err := check("roles.judge", j.Agent); err != nil {
			return err
		}
		// The judge decides what a review reports; it never edits. Allowing a
		// write-capable agent here would put an editing agent inside a review- config,
		// whose entire promise is that it cannot modify the target -- the same rule
		// that keeps write-capable agents out of the reviewer pool.
		if c.Agents[j.Agent].CanEdit {
			return fmt.Errorf("roles.judge.agent: %q declares can_edit; the judge must be read-only, since a review config never modifies its target", j.Agent)
		}
	}

	// A lens's identity in logs is its prompt basename (LensName), and the
	// per-step log path is {role}-{agent}-{prompt}-... So two lenses whose
	// prompt files share a basename (prompts/a/review.md and prompts/b/review.md)
	// produce the same {prompt} token; under strategy all one agent runs both in
	// a round and their reviewer goroutines race to os.WriteFile the same path,
	// silently losing one durable record. Require distinct lens names so the
	// per-assignment log path is always unique.
	//
	// A lens's contract salvage logs under LensName+ReformatLensSuffix, so that
	// name is reserved too: lenses "review-bugs" and "review-bugs-reformat" would
	// otherwise share one {prompt} token, and no logs.pattern can separate two
	// writes whose whole identity is equal.
	seenLens := map[string]int{}
	for i, l := range c.Roles.Review.Prompts {
		name := LensName(l.Prompt)
		if j, dup := seenLens[name]; dup {
			return fmt.Errorf("roles.review.prompts: lenses %d (%s) and %d (%s) both resolve to log name %q; prompt basenames must be unique so parallel reviewers' step logs do not overwrite each other -- rename one prompt file", j, c.Roles.Review.Prompts[j].Prompt, i, l.Prompt, name)
		}
		seenLens[name] = i
	}
	for i, l := range c.Roles.Review.Prompts {
		name := LensName(l.Prompt)
		if j, dup := seenLens[ReformatLensName(name)]; dup && j != i {
			return fmt.Errorf("roles.review.prompts: lens %d (%s) resolves to log name %q, which is where lens %d (%s) logs the reply it is asked to restate when its output breaks the contract; those two step logs would overwrite each other -- rename one prompt file so no lens name is another's name plus %q", j, c.Roles.Review.Prompts[j].Prompt, ReformatLensName(name), i, l.Prompt, ReformatLensSuffix)
		}
	}

	// Every role must name a prompt, and the resolved file must be readable. The
	// loader fills PromptPath; it is empty when Validate is called on a config
	// that was never resolved through a bundle (unit tests), in which case the
	// name itself is treated as the path.
	type promptRef struct{ name, file string }
	refs := make([]promptRef, 0, len(c.Roles.Review.Prompts)+2)
	if c.Roles.Coder.Prompt != "" {
		refs = append(refs, promptRef{c.Roles.Coder.Prompt, c.Roles.Coder.PromptFile()})
	}
	if j := c.Roles.Judge; j.Prompt != "" {
		refs = append(refs, promptRef{j.Prompt, j.PromptFile()})
	}
	if p := c.Roles.Planner; p.Prompt != "" {
		refs = append(refs, promptRef{p.Prompt, p.PromptFile()})
	}
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
		// The lens name is what logs.pattern substitutes for {prompt}. Checked here
		// rather than in the uniqueness loop above so a missing or unreadable prompt
		// still reports as exactly that.
		if err := validPathIdent("lens log", LensName(ref.name)); err != nil {
			return fmt.Errorf("prompt %s: %w", ref.name, err)
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
		//
		// Compare the path the logstore actually writes, not the raw rendering: it
		// writes filepath.Join(roundDir, StepPath(...)), and Join cleans slash and
		// dot segments, so "a/../b.md" and "b.md" are one file that a string compare
		// of the renderings would call distinct. Cleaning against a probe root also
		// catches a pattern whose own literals climb out of the round directory.
		seenPath := map[string][3]string{}
		for _, id := range c.logIdentities() {
			p := c.renderLogPattern(id)
			full := filepath.Clean(filepath.Join(logProbeRoot, p))
			if !strings.HasPrefix(full, logProbeRoot+string(filepath.Separator)) {
				return fmt.Errorf("logs.pattern %q renders identity %v to %q, which normalizes outside its round directory; step logs must stay beneath the round directory, else a run scatters artifacts over the tree it is reviewing -- remove the leading .. segments from the pattern", c.Logs.Pattern, id, p)
			}
			if prev, dup := seenPath[full]; dup && prev != id {
				return fmt.Errorf("logs.pattern %q maps distinct log identities %v and %v to the same path %q; under strategy all their step/prompt writes run concurrently and would race onto it, silently overwriting one reviewer's or the coder's record -- add a separator between (or reorder) the {role}/{agent}/{prompt} placeholders so every identity renders a distinct path", c.Logs.Pattern, prev, id, p)
			}
			seenPath[full] = id
		}
	}
	if c.Logs.SummaryPattern != "" && !strings.Contains(c.Logs.SummaryPattern, "{ext}") {
		return fmt.Errorf("logs.summary_pattern %q must contain {ext}, else the JSON summary overwrites the Markdown one (and its path is returned as the Markdown path)", c.Logs.SummaryPattern)
	}
	// The extra redaction patterns are compiled here, at startup, rather than on
	// first use: a pattern that does not compile would otherwise be discovered by
	// the write that was supposed to mask a secret, with the run already underway.
	if _, err := c.Logs.RedactPatterns(); err != nil {
		return err
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
// (Review.LensAgents, the same policy the orchestrator's fan-out consumes) and
// that lens's contract salvage, which logs under ReformatLensName, plus the
// coder. It is the identity space the step-log path must render injectively over.
//
// The reformat identity belongs here because it races the same way: it is written
// mid-round while the other lenses on that agent are still running, so a pattern
// that maps one lens's reformat onto another lens's normal path loses a record
// just as silently.
func (c *Config) logIdentities() [][3]string {
	var ids [][3]string
	rv := c.Roles.Review
	for _, l := range rv.Prompts {
		name := LensName(l.Prompt)
		for _, a := range rv.LensAgents(l) {
			ids = append(ids, [3]string{"review", a, name}, [3]string{"review", a, ReformatLensName(name)})
		}
	}
	if c.Roles.Coder.Agent != "" {
		ids = append(ids, [3]string{"fix", c.Roles.Coder.Agent, LensName(c.Roles.Coder.Prompt)})
	}
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

// logProbeRoot stands in for the round directory when validation normalizes a
// rendered logs.pattern. Any absolute directory works: only the shape of what
// filepath.Clean does to the pattern relative to it is being measured.
var logProbeRoot = string(filepath.Separator) + "round"

// validPathIdent rejects an identifier that logs.pattern substitutes into a file
// path. The logstore writes filepath.Join(roundDir, StepPath(...)), and Join
// cleans slash and dot segments away, so an agent named "x/../review-b" and one
// named "review-b" render to two strings that the collision validator sees as
// distinct but that name ONE file: their parallel Prompt and Step writes then
// truncate each other's artifacts nondeterministically. A leading ".." is worse
// -- it walks the artifact out of the round directory into the tree under
// review. Keeping separators and dot segments out of the identifiers is what
// makes comparing the normalized rendered paths sound.
func validPathIdent(kind, name string) error {
	switch {
	case name == "":
		return fmt.Errorf("%s name is empty; it is substituted into logs.pattern and every identity must render a distinct path", kind)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("%s name %q must not contain a path separator: it is substituted into logs.pattern, and the joined path is cleaned before it is written, so the artifact silently lands on another identity's file or outside the round directory", kind, name)
	case name == "." || name == "..":
		return fmt.Errorf("%s name %q must not be a dot segment: it is substituted into logs.pattern, and the joined path is cleaned before it is written, so the artifact silently lands on another identity's file or outside the round directory", kind, name)
	}
	return nil
}

// validCommandValue refuses a model/effort value that would read as an argument
// rather than as a value. Both are substituted into the command template, so a
// value carrying whitespace is trying to be more than one argv element, and one
// starting with '-' is trying to be a flag: `--model --setting-sources` is parsed
// by some CLIs as a bare --setting-sources with --model left to default, which is
// exactly the hardcoded flag config/agents/claude.yaml exists to guarantee.
//
// Neither shape has a legitimate use -- no CLI names a model or an effort level
// with a space or a leading dash -- so this costs nothing and closes the field as
// an injection point, whatever a file inside the target declares. Argv keeps the
// value to a single element regardless; this makes the attempt an error the
// operator sees rather than a silently odd argument.
//
// Whitespace is unicode.IsSpace and not the ASCII six, because that is exactly
// what Argv's strings.Fields splits on: a validator that called U+00A0 a normal
// character would wave through the one spelling of `model: claude-opus-5<NBSP>
// --setting-sources<NBSP>target` that the splitter still reads as three words.
func validCommandValue(agent, field, v string) error {
	switch {
	case strings.IndexFunc(v, unicode.IsSpace) >= 0:
		return fmt.Errorf("agents.%s: %s %q must not contain whitespace: it is substituted into the command template, and a value spelled as several words is an attempt to append arguments of its own to the agent's command line", agent, field, v)
	case strings.HasPrefix(v, "-"):
		return fmt.Errorf("agents.%s: %s %q must not start with '-': it is substituted into the command template, where a leading dash makes the value read as a flag rather than as the argument of the one it follows", agent, field, v)
	}
	return nil
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
	// Both dotenv spellings: .env.local/.env.production, and the equally common
	// production.env/docker.env -- compose and dotenv tooling read either.
	"**/.env", "**/.env.*", "**/*.env",
	// direnv's own file, which none of the dotenv patterns reach: ".envrc" has no
	// dot after "env", so "**/.env.*" does not match it. It is a shell script that
	// exports secrets by arbitrary names ("export CUSTOM_API_KEY=..."), and
	// agent.RedactSecrets only masks fixed-shape tokens, so anything it exports
	// under a name outside that set survives into the prompt verbatim. .envrc.local
	// is the sourced-sibling habit .env.local is; .direnv/ is where direnv caches
	// the dumped environment those exports produce.
	"**/.envrc", "**/.envrc.*", "**/.direnv/**",
	"**/*.pem", "**/*.p12", "**/*.pfx", "**/*.key",
	"**/*.jks", "**/*.keystore", "**/*.ppk", // Java keystores, PuTTY keys
	"**/id_rsa", "**/id_dsa", "**/id_ecdsa", "**/id_ed25519",
	"**/.npmrc", "**/.netrc", "**/.pgpass", "**/.git-credentials",
	"**/credentials", // ~/.aws/credentials shape
	"**/kubeconfig",
	"**/*.kdbx",
}

// mandatoryExcludeSet indexes mandatoryExcludes so FoldExclude can still
// recognize one after EffectiveExcludes has flattened the two sources into a
// single list.
var mandatoryExcludeSet = func() map[string]bool {
	set := make(map[string]bool, len(mandatoryExcludes))
	for _, g := range mandatoryExcludes {
		set[g] = true
	}
	return set
}()

// FoldExclude reports whether an exclude glob is matched case-insensitively.
//
// The mandatory patterns above are spelled entirely in lowercase, so matching
// them as written lets a committed PRODUCTION.ENV, .Env, server.PEM, ID_RSA or
// KUBECONFIG past the one exclusion no config can drop -- spellings that are
// ordinary in real repositories, and that fixpoint's own supported filesystems
// may not even distinguish. In the git modes that exclusion is what keeps the
// file's full CONTENT out of every reviewer prompt and out of the always-written
// .prompt artifact, not merely its path, and agent.RedactSecrets only masks
// fixed-shape tokens. So the credential patterns fold rather than trying to name
// every spelling.
//
// Configured excludes are matched as written: they are the operator's own intent,
// and folding them would start dropping files nobody asked to hide -- a "Test/"
// directory for an excluded "test/" -- with nothing in the material saying so.
// Both matchers consult this, the compiled regexps in compileGlobs and the git
// :(exclude,glob) pathspecs in collectPathspec, so the two keep agreeing.
func FoldExclude(glob string) bool { return mandatoryExcludeSet[glob] }

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
// target is itself treated as untrusted input -- see Loaded.ProjectSuppliedPolicy.
type Verify struct {
	Policy VerifyPolicy `yaml:"policy"`
	// Timeout bounds EACH command. A hung test suite must not hang the run.
	Timeout  Duration        `yaml:"timeout"`
	Commands []VerifyCommand `yaml:"commands"`
	// Gate names a file under <bundle>/gates/ that supplies Commands (and the
	// closing round's loop.final_skip_run_edits) instead of this config spelling
	// them out. It exists because those two keys are the ONLY part of a fix config
	// that changes with the project's language, and the shipped fix-* configs
	// were spelling out Go's -- `go vet`, `go test -race`, `**/*_test.go` -- in
	// files that ship to every user. On a Rust or Node tree under no_regressions
	// that gate went silently vacuous: every command red at the baseline, every
	// one tolerated, nothing ever blocked, and no line of output said "wrong
	// language". A gate is named, in this key or with -gate on the command line,
	// and `-check` prints which file it resolved to.
	//
	// Mutually exclusive with Commands: a config that sets both is refused rather
	// than merged, because "which of these actually runs?" must be answerable
	// from the file. A gate file cannot name another gate -- the same one-level
	// rule as `extends`, for the same reason: the effective gate is readable from
	// two files, the config and the gate it names. Resolution inlines the gate's
	// commands here, so everything downstream reads Commands and never knows.
	Gate string `yaml:"gate"`
}

// Gate is the shape of a <bundle>/gates/<name>.yaml file: the language's share of
// a task config. Both keys are REQUIRED and both land in the config that names
// the gate -- Commands into verify.commands, SkipRunEdits into
// loop.final_skip_run_edits (only when the config left that empty; an explicit
// list in the config wins, except under -gate, which replaces both).
type Gate struct {
	Commands []VerifyCommand `yaml:"commands"`
	// SkipRunEdits is the test-file shape of the language ("**/*_test.go",
	// "**/test_*.py"), which is why it lives here beside the commands: what
	// counts as a test file is a property of the language, not of the task.
	SkipRunEdits []string `yaml:"skip_run_edits"`
}

// ReviewPolicy holds what a REVIEW run concludes with, as opposed to what it
// looks for. It is empty by default and every field has a working default, so a
// config that says nothing about reviews still produces a verdict.
type ReviewPolicy struct {
	// BlockAt is the severity at or above which a surviving finding forces
	// CHANGES_REQUESTED (default: high). Lower it to medium for a stricter gate,
	// knowing what that costs: across 19 measured runs the panel produced 322
	// issues of which only 66 were high or critical, so a medium floor blocks
	// nearly every review -- and a gate that always fires is one people route
	// around.
	BlockAt string `yaml:"block_at"`
	// Signature is appended to the rendered review, with {agents} {run} {version}
	// {config} {verdict} substituted. Empty uses review.DefaultSignature.
	//
	// It is rendered by fixpoint from fixpoint's own facts and placed outside every
	// region carrying agent text: a signature composed from a finding's prose could
	// be forged by whatever wrote that prose.
	Signature string `yaml:"signature"`
	// ReplySignature signs an answer posted into an existing conversation, with the
	// same placeholders. Empty uses review.DefaultReplySignature.
	//
	// Its own key because a reply is not a review: the review signature says
	// "Reviewed by", which is a claim a two-line answer in a thread does not
	// support, and {agents} here is the single coder that wrote the reply rather
	// than the panel that reviewed.
	ReplySignature string `yaml:"reply_signature"`
	// Refute names the prompt for the refutation round, or is empty to skip it.
	//
	// One key rather than a bool plus a name, because the two could disagree and
	// then the config would say something it does not do.
	//
	// The round exists because agreement cannot sort signal from noise in this
	// panel: measured across 19 runs, under 4% of findings were reported by more
	// than one reviewer, and 0 of 25 false positives were among them. Keeping only
	// corroborated findings -- the obvious alternative -- would have discarded 237
	// of 248 confirmed defects. Asking every reviewer to take an evidenced position
	// on the merged set gets a judgment on each finding instead of a popularity
	// count.
	Refute string `yaml:"refute"`
	// RefutePath is the resolved file for Refute; filled in during loading.
	RefutePath string `yaml:"-"`
	// RefuteAt is the severity floor for what the round is asked about (default:
	// high). Findings below it skip refutation entirely and go straight to the
	// judge.
	//
	// The round pays for itself as a SAFETY GATE, not as a filter, and the gate only
	// covers blocking findings -- applyJudgment honors a drop on one of those only
	// where refutation recorded doubt, so no single agent can delete a blocker.
	// Below the floor the judge already decides alone and a second opinion changes
	// nothing about what it may do.
	//
	// Measured over the two runs that ran it (97 findings): 29 contested, zero
	// dropped unanimously, so as a filter it never fired. It is not worthless --
	// contested findings were dropped by the judge at 55% against 26% for the rest
	// (Fisher two-sided p = 0.010), and the judge does not see the flag, so those are
	// two independent judgments agreeing -- but agreeing with a decision the judge
	// makes anyway is not worth a full extra pass per reviewer per round. Restricting
	// it to the severities the gate protects cut it to roughly a third of that cost:
	// 37 of those 97 findings were high or critical.
	//
	// Set it to low to refute everything, as every run before 2026-08-06 did.
	RefuteAt string `yaml:"refute_at"`

	// Post and PostVerdict are set ONLY by -post / -post-verdict on the command
	// line; `yaml:"-"` is load-bearing security, not style. Publishing is an action
	// on somebody else's pull request, and the first bundle on the search path is
	// the target's own -- so a YAML-readable key would let reviewed code arrange to
	// have a review posted under the operator's identity. Same argument as
	// Loop.TrustedTarget, and the decoder's KnownFields(true) makes an attempt to
	// set them a load error rather than a silent no-op -- rejectTrustKeys names both
	// keys so that error reads as the boundary it is rather than as a typo.
	Post        bool `yaml:"-"`
	PostVerdict bool `yaml:"-"`
}

// validateCreate is every rule about the create-design pipeline's shape.
//
// The refusals of coder, judge, and review lenses are the same house rule as
// every other refused-inert key: a create run drafts a document -- nothing is
// fixed, no findings exist for a judge to filter, no lens ever runs -- so those
// keys sitting in the config would describe a run that does not happen. A reader
// must be able to trust that what a config says is what its run does.
func (c *Config) validateCreate() error {
	if !c.IsCreate() {
		return c.refusePartialCreate()
	}
	if c.Create.Critique == "" {
		return errors.New("create.critique: required in a create config -- the critique phase is what stops a synthesis of unexamined proposals (its runtime skipping below two proposals is a degradation, not a configuration)")
	}
	if c.Roles.Editor.Agent == "" {
		return errors.New("roles.editor: required in a create config -- the panel does not converge, so somebody must hold the pen")
	}
	switch {
	case c.Create.Objections < 0:
		return fmt.Errorf("create.objections: must not be negative, got %d", c.Create.Objections)
	case c.Create.Objections > 1:
		// The same measurement behind max_final_passes: a second pass over the
		// revision critiques the revision, and the loop does not converge.
		return fmt.Errorf("create.objections: at most 1, got %d -- an objection loop does not converge; the check on the revision is the review-design run that follows", c.Create.Objections)
	case c.Create.Objections == 1 && c.Create.Object == "":
		return errors.New("create.object: required when create.objections is 1 -- the objection pass needs its prompt")
	case c.Create.Objections == 0 && c.Create.Object != "":
		return errors.New("create.object is set but create.objections is 0, so the pass it names never runs")
	}
	if c.Target.Mode != ModeDirectory {
		return fmt.Errorf("create: target.mode must be directory (the assignment is a local file or directory), got %q", c.Target.Mode)
	}
	if len(c.Roles.Review.Prompts) > 0 {
		return errors.New("roles.review.prompts is set in a create config; a create run drafts, it does not review -- the lenses belong to the review-design that follows")
	}
	if c.Roles.Coder.Agent != "" || c.Roles.Coder.Prompt != "" {
		return errors.New("roles.coder is set in a create config; nothing is fixed here, and a write-capable agent has no place in a run whose only product fixpoint itself writes")
	}
	if c.Roles.Judge.Agent != "" || c.Roles.Judge.Prompt != "" {
		return errors.New("roles.judge is set in a create config; there are no findings to filter -- the editor is this pipeline's arbiter")
	}
	if c.Review.Refute != "" {
		return errors.New("review.refute is set in a create config; refutation runs over findings, which a create run does not produce -- the critique phase is its analog here")
	}
	if len(c.Roles.Review.Agents) == 0 {
		return errors.New("roles.review.agents: a create config needs the pool -- it is who proposes and critiques")
	}
	return nil
}

// validateReviewStrategy checks the lens/pool schedule; split out of Validate
// so the one shape with no panel at all (implement) can skip it whole.
func (c *Config) validateReviewStrategy() error {
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
	return nil
}

// validateImplement is every rule about the implement-design pipeline's shape.
// The inert-key refusals are the house rule: an implement run plans and builds
// -- no panel reviews, no judge filters, no editor writes -- so those keys in
// the config would describe a run that does not happen.
func (c *Config) validateImplement() error {
	if !c.IsImplement() {
		return c.refusePartialImplement()
	}
	if c.Roles.Planner.Agent == "" || c.Roles.Planner.Prompt == "" {
		return errors.New("roles.planner: both agent and prompt are required in an implement config -- the planner is what turns a design into tasks")
	}
	if c.Roles.Coder.Agent == "" || c.Roles.Coder.Prompt == "" {
		return errors.New("roles.coder: required in an implement config -- the coder is the only role that builds")
	}
	if len(c.Roles.Review.Prompts) > 0 {
		return errors.New("roles.review.prompts is set in an implement config; nothing reviews during implementation -- the lenses belong to the review-code run that follows")
	}
	if c.Roles.Judge.Agent != "" || c.Roles.Judge.Prompt != "" {
		return errors.New("roles.judge is set in an implement config; there are no findings to filter here")
	}
	if c.Roles.Editor.Agent != "" || c.Roles.Editor.Prompt != "" {
		return errors.New("roles.editor is set in an implement config; the deliverable is a repository, not a document")
	}
	if c.Review.Refute != "" {
		return errors.New("review.refute is set in an implement config; refutation runs over findings, which an implement run does not produce")
	}
	if c.Loop.CommitPolicy != CommitPerFix {
		return fmt.Errorf("loop.commit_policy: an implement run is %q only -- squashing collapses one-task-one-revert, the property the pipeline exists to provide (got %q)", CommitPerFix, c.Loop.CommitPolicy)
	}
	if c.Loop.ReviewOnly {
		return errors.New("loop.review_only is set in an implement config; an implement run builds. To plan without building, use -plan-only (§7.4), which stops after the plan is written and validated")
	}
	// Gate policy (DESIGN.md §7.2): a no-regressions baseline is captured on
	// the pristine tree, which is EMPTY here, so every check would be exempted
	// for the whole run -- a gate that can never block, worse than none. And an
	// empty command list must be an assertion, not an accident: on an existing
	// repository silence is tolerable, on a project that does not exist yet it
	// is far more likely a half-finished config.
	if c.Verify.Policy == VerifyNoRegressions {
		return errors.New("verify.policy: no_regressions is refused in an implement config -- the baseline tree is empty, so the gate could never block; use must_pass (defaults.yaml's policy does not carry over usefully here)")
	}
	if len(c.Verify.Commands) == 0 && c.Verify.Policy != VerifyOff {
		return errors.New("verify.commands is empty; an implement config must either configure the gate or assert `verify.policy: off` -- an ungated new project has to be a statement, not a half-finished config")
	}
	// The write-target still rides in the create key (§12.8 records that as an
	// open question), so it is passed in rather than mirrored into a second field.
	if err := c.Implement.validateModes(c.Create.Out); err != nil {
		return err
	}
	return c.Implement.validate()
}

// validateModes refuses the §7.4 flag combinations that contradict each other,
// at load time rather than three phases in. Each pair is incoherent rather than
// merely unusual, so a refusal naming both is more useful than a precedence
// rule nobody could remember.
func (i Implement) validateModes(out string) error {
	switch {
	case i.Continue != "" && out != "":
		return errors.New("-continue and -out name two different projects: -continue resumes the one already built, and takes its plan and design from that repository")
	case i.Continue != "" && i.Plan != "":
		return errors.New("-continue and -plan cannot combine: a resumed run reads the plan from the project's committed PLAN.json, which is the plan its history was built against")
	case i.Continue != "" && i.PlanOnly:
		return errors.New("-continue and -plan-only cannot combine: one finishes a build, the other refuses to start one")
	case i.PlanOnly && i.Plan != "":
		return errors.New("-plan-only and -plan cannot combine: -plan-only exists to PRODUCE a plan, and -plan supplies one already made")
	}
	return nil
}

// validate checks the implement section's own numbers; the pipeline-shape rules
// stay in validateImplement, split only for the complexity limit.
func (i Implement) validate() error {
	switch {
	case i.MaxTasks < 1:
		return fmt.Errorf("implement.max_tasks: must be at least 1, got %d", i.MaxTasks)
	case i.MaxFilesPerTask < 1:
		return fmt.Errorf("implement.max_files_per_task: must be at least 1, got %d", i.MaxFilesPerTask)
	case i.MaxTaskAttempts < 1:
		return fmt.Errorf("implement.max_task_attempts: must be at least 1, got %d", i.MaxTaskAttempts)
	case i.MaxVacuousFrac < 0 || i.MaxVacuousFrac > 1:
		return fmt.Errorf("implement.max_vacuous_frac: must be within 0..1, got %g", i.MaxVacuousFrac)
	case i.MaxInfraTries < 1:
		return fmt.Errorf("implement.max_infra_tries: must be at least 1, got %d", i.MaxInfraTries)
	case i.MaxRunDuration < 0:
		return fmt.Errorf("implement.max_run_duration: must not be negative, got %s", i.MaxRunDuration.Std())
	case i.MaxTaskBytes < 0:
		return fmt.Errorf("implement.max_task_bytes: must not be negative, got %s", i.MaxTaskBytes)
	case i.MinFreeDisk < 0:
		return fmt.Errorf("implement.min_free_disk: must not be negative, got %s", i.MinFreeDisk)
	}
	if cc := i.CleanCheck; cc != CleanCheckLast && cc != CleanCheckEvery && cc != CleanCheckOff {
		return fmt.Errorf("implement.clean_check: want %s | %s | %s, got %q", CleanCheckLast, CleanCheckEvery, CleanCheckOff, cc)
	}
	for n, g := range i.GateGenerated {
		if g == "" || strings.HasPrefix(g, "/") || strings.Contains(g, "..") {
			return fmt.Errorf("implement.gate_generated[%d]: %q must be a relative path inside the project", n, g)
		}
	}
	return i.validateReservedNames()
}

// implementControlArtifacts are the four files fixpoint owns in the project it
// builds (DESIGN.md §4.3). Named here rather than imported: internal/implement
// depends on this package, so the reverse is impossible, and one list refusing
// what the other protects is the point.
var implementControlArtifacts = []string{"DESIGN.md", "PLAN.md", "PLAN.json", ".gitignore"}

// validateReservedNames refuses the two ways gate_generated and gitignore_seed
// can contradict each other or the control artifacts.
//
// Neither had a rule (review run 20260813-180828, i60). An ignored
// gate_generated lockfile is the sharp one: the gate writes package-lock.json,
// .gitignore hides it, the commit omits it, and the final clean-clone check
// then fails on a project whose every task passed -- a whole run lost to two
// lines of stack config that never looked wrong. Naming a control artifact in
// either list is the other: it would put a file fixpoint owns under a rule
// fixpoint also enforces, in opposite directions.
func (i Implement) validateReservedNames() error {
	ignored := make(map[string]bool, len(i.GitignoreSeed))
	for _, s := range i.GitignoreSeed {
		ignored[strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "/"))] = true
	}
	for _, g := range i.GateGenerated {
		if ignored[g] {
			return fmt.Errorf("implement.gate_generated names %q and implement.gitignore_seed ignores it -- the gate would write a file no commit could ever carry, and the clean-clone check would fail a project whose every task passed", g)
		}
	}
	for _, reserved := range implementControlArtifacts {
		if ignored[reserved] {
			return fmt.Errorf("implement.gitignore_seed names %q, a control artifact fixpoint writes and protects (§4.3)", reserved)
		}
		for _, g := range i.GateGenerated {
			if g == reserved {
				return fmt.Errorf("implement.gate_generated names %q, a control artifact no gate may maintain (§4.3)", reserved)
			}
		}
	}
	return nil
}

// refusePartialImplement rejects implement.* keys in a config that is not an
// implement config -- the same inert-key refusal as refusePartialCreate.
func (c *Config) refusePartialImplement() error {
	i := c.Implement
	if i.MaxTasks != 0 || i.MaxFilesPerTask != 0 || i.MaxTaskAttempts != 0 ||
		i.MaxVacuousFrac != 0 || i.MaxInfraTries != 0 || i.MaxRunDuration != 0 || i.MaxTaskBytes != 0 ||
		i.MinFreeDisk != 0 || i.CleanCheck != "" || len(i.GitignoreSeed) > 0 ||
		len(i.GateGenerated) > 0 {
		// NoCoverageCheck is deliberately absent: it is yaml:"-", so it can only
		// have come from the flag, and a flag is not a partial config.
		return errors.New("implement.* is set but roles.planner is not; an implement run is defined by having a plan made")
	}
	return nil
}

// refusePartialCreate rejects create.* keys in a config that is not a create
// config -- each is the same inert-key refusal, split out of validateCreate only
// to keep that function under the complexity limit.
func (c *Config) refusePartialCreate() error {
	switch {
	case c.Create.Critique != "":
		return errors.New("create.critique is set but create.propose is not; a create run is defined by having proposals to make")
	case c.Create.Object != "":
		return errors.New("create.object is set but create.propose is not; a create run is defined by having proposals to make")
	case c.Create.Objections != 0:
		return errors.New("create.objections is set but create.propose is not; a create run is defined by having proposals to make")
	}
	return nil
}

// BlockFloor is the resolved severity floor that forces CHANGES_REQUESTED.
//
// BlockFloor and RefuteFloor resolve the two floors the same way every consumer
// must, so validation compares the values that will actually be applied rather than
// the raw -- possibly empty -- strings, and the orchestrator selecting findings for
// refutation reads the same answer this file validated.
func (r ReviewPolicy) BlockFloor() string {
	if r.BlockAt == "" {
		return model.DefaultBlockAt
	}
	return model.NormalizeSeverity(r.BlockAt)
}

// RefuteFloor is the resolved severity floor for the refutation round.
func (r ReviewPolicy) RefuteFloor() string {
	if r.RefuteAt == "" {
		return model.DefaultRefuteAt
	}
	return model.NormalizeSeverity(r.RefuteAt)
}

// VerifyCommand is one check. Argv, not a shell string: there is no shell to
// quote wrongly, and it matches how agents are configured.
type VerifyCommand struct {
	Name string   `yaml:"name"`
	Run  []string `yaml:"run"`
	// Optional records the result without ever failing the round. For a check
	// that is informative but not a gate (a linter mid-cleanup, say).
	Optional bool `yaml:"optional"`
	// Infra marks a command whose failure is a fact about the ENVIRONMENT rather
	// than about the code: it reaches a network, a registry, a proxy or a cache
	// that the project does not control. `npm ci` is the shipped example.
	//
	// It exists because the infrastructure/result split covered the agent's
	// provider and not the gate's dependencies (review run 20260813-222753). A
	// provider refusal is "a fact about the morning" and costs no attempt; a
	// five-minute registry 503 exiting non-zero from `npm ci` was a verdict on
	// code nobody found fault with -- it burned both attempts, wrote a PERMANENT
	// failed marker, skipped the task's whole dependent subtree, and (with
	// -continue unbuilt) cost the rest of the run. Marking a command infra
	// routes its failure exactly as a provider refusal: no attempt consumed, no
	// marker written, backed off and fed to the circuit breaker.
	//
	// Only meaningful in an implement config, and deliberately opt-in: fixpoint
	// cannot tell a network failure from a real one by reading an exit code, so
	// the operator -- who knows what each command reaches -- declares it.
	Infra bool `yaml:"infra"`
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

// validEnvName rejects a name that cannot be an environment variable, so a typo
// like `env.pass: [FOO=bar]` fails at startup instead of silently passing nothing.
func validEnvName(name string) error {
	switch {
	case name == "":
		return errors.New("an environment variable name must not be empty")
	case strings.ContainsAny(name, "= \t\n\x00"):
		return fmt.Errorf("%q is not a valid environment variable name: list names only, not assignments (use env.set for values)", name)
	}
	return nil
}
