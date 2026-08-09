package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Source records where every file a run is built from was resolved, so the run
// can report and persist it. These files decide what agents are told to do and
// which trust gates apply, so a bare "loaded config X" is not enough provenance:
// a project-local prompt shadowing the installed one changes behavior in a way
// only the resolved path reveals.
type Source struct {
	Config  string            // the task config
	Extends string            // the config it inherited from, if any
	Agents  map[string]string // agent name -> file
	Prompts map[string]string // prompt name -> file
}

// Loaded is a fully resolved configuration plus the provenance of its parts.
//
// It is IMMUTABLE after LoadBundle returns. Everything that shapes the effective
// configuration -- file inheritance, defaults, path anchoring, and the run's CLI
// overrides -- happens inside LoadBundle, so there is exactly one place where the
// question "what will this run actually do?" is decided. Nothing outside this
// package writes to Config afterwards.
//
// That ordering is load-bearing rather than stylistic: an override changes what a
// VALID configuration is (-review-only exempts an all-once lens list from the
// recurring-lens rule), so validation has to see the post-override value. When
// overrides were applied by the caller after loading, every future caller had to
// remember to re-validate in the right order; folding them in makes the wrong
// order unrepresentable.
type Loaded struct {
	Config      *Config
	Source      Source
	ProjectRoot string
	// Overrides records the command-line assertions that were folded into Config,
	// so a run can report which settings came from flags rather than from the files
	// listed in Source. Provenance is the point: the config path alone no longer
	// explains the effective run.
	Overrides Overrides
	// ownInlineAgents names the agents the task config's OWN `agents:` map declared,
	// as opposed to the ones it inherited from its extends base. Source cannot say
	// this: an inline agent has no file of its own, so both files are candidates
	// until the merge is known. It is unexported because it is an input to the
	// provenance checks rather than provenance a run reports -- see
	// agentDefinitionPaths.
	ownInlineAgents map[string]bool
}

// Overrides are the run's command-line assertions. They are applied while the
// configuration is compiled, not to a Config that already exists -- see Loaded.
//
// These are deliberately NOT config keys. The bundle may itself come from the
// repository under review, and code being reviewed must not be able to declare
// itself trustworthy; a trust gate is therefore an assertion the operator makes
// per invocation, which only a flag can express.
type Overrides struct {
	ReviewOnly    bool
	MaxIterations int
	// BaseRef overrides target.base_ref in git-diff mode. Unlike the other
	// overrides this one names WHAT gets reviewed rather than relaxing a limit,
	// and it exists because the base is the one setting that legitimately differs
	// per invocation: fix-branch defaults to `@{upstream}...`, which a branch that
	// has never been pushed does not have, and editing YAML to review a branch is
	// not a workflow.
	BaseRef string
	// PR overrides target.pr in pr mode, and exists for the same reason BaseRef
	// does: which pull request to review is per-invocation by nature, so the
	// bundled review-pr config carries no usable number and would otherwise have to
	// be copied and edited once per PR.
	PR int
	// Target points a directory-mode run at a file or a directory, as an ABSOLUTE
	// path (the caller resolves it against its own working directory -- this layer
	// cannot know where the flag was typed). A directory becomes target.path; a
	// file becomes target.document with its parent as target.path, so the panel
	// reads the document as the material while still working inside the directory
	// that gives it context. Which file to review is per-invocation by nature --
	// the same argument as BaseRef and PR.
	Target            string
	AllowUntrustedFix bool
	// Post publishes the review on the pull request, and PostVerdict additionally
	// lets it carry the verdict (approve / request changes) instead of a comment.
	//
	// Flags, never config keys, for the same reason the trust gates are: publishing
	// is an action on somebody else's pull request, and the config that would grant
	// it may have been shipped by the repository under review. See TrustedTarget.
	Post          bool
	PostVerdict   bool
	TrustedTarget bool
	// TrustedBundle asserts trust in the bundle alone, not in the target's content;
	// see Loop.TrustedBundle for why the two are separate assertions.
	TrustedBundle bool
}

// apply folds the overrides into the configuration being compiled.
func (o Overrides) apply(c *Config) {
	// The booleans can only force ON. Turning a gate off is the config's job, so a
	// flag can never silently weaken a run someone configured deliberately.
	if o.ReviewOnly {
		c.Loop.ReviewOnly = true
	}
	if o.AllowUntrustedFix {
		c.Loop.AllowUntrustedFix = true
	}
	if o.Post {
		c.Review.Post = true
	}
	if o.PostVerdict {
		c.Review.PostVerdict = true
	}
	if o.TrustedTarget {
		c.Loop.TrustedTarget = true
	}
	if o.TrustedBundle {
		c.Loop.TrustedBundle = true
	}
	// Any nonzero value applies, including a negative one: an explicit
	// -max-iterations -1 must reach Validate so its must-not-be-negative rule
	// rejects the bad input rather than being swallowed as "no flag supplied".
	// Zero stays "use config", per the flag help.
	if o.MaxIterations != 0 {
		c.Loop.MaxIterations = o.MaxIterations
	}
	// Empty stays "use config": clearing a base_ref would silently turn a fix run
	// into an unstaged-changes review, which Validate then rejects for a reason the
	// operator never asked for.
	if o.BaseRef != "" {
		c.Target.BaseRef = o.BaseRef
	}
	// Same rule as MaxIterations: any nonzero value applies, so an explicit -pr -1
	// is rejected by Validate rather than swallowed as "no flag supplied". Zero
	// stays "use config", which is also the placeholder the bundled review-pr
	// config carries.
	if o.PR != 0 {
		c.Target.PR = o.PR
	}
}

// applyTarget folds the -target override in, and is the one override that can
// FAIL: it names something on disk, and a typo must be a load error rather than a
// run over the wrong directory. Separate from apply because apply cannot error
// and retrofitting an error onto nine infallible assignments for the sake of one
// stat would make every caller handle a failure eight of them cannot have.
func (o Overrides) applyTarget(c *Config) error {
	if o.Target == "" {
		return nil
	}
	if !filepath.IsAbs(o.Target) {
		// The contract with the caller, restated as an error: this layer anchors
		// relative paths against projectRoot, and the flag was typed relative to the
		// operator's cwd, which may differ. Refusing is better than resolving against
		// the wrong root and reviewing whatever happens to be there.
		return fmt.Errorf("-target %q must be an absolute path", o.Target)
	}
	info, err := os.Stat(o.Target)
	if err != nil {
		return fmt.Errorf("-target %s: %w", o.Target, err)
	}
	if info.IsDir() {
		c.Target.Path = o.Target
		return nil
	}
	c.Target.Path = filepath.Dir(o.Target)
	c.Target.Document = filepath.Base(o.Target)
	return nil
}

// Applied names the overrides that changed the configuration, for the run log.
// It reports what was ASSERTED, so a flag whose value the config already set
// still appears: the operator's assertion is the thing worth recording.
func (o Overrides) Applied() []string {
	var out []string
	if o.ReviewOnly {
		out = append(out, "review_only=true")
	}
	if o.AllowUntrustedFix {
		out = append(out, "allow_untrusted_fix=true")
	}
	// Recorded for the same reason the trust gates are: publishing acts on somebody
	// else's pull request, so the authorization to do it belongs in the run log even
	// when the run ends in a comment rather than a verdict.
	if o.Post {
		out = append(out, "post=true")
	}
	if o.PostVerdict {
		out = append(out, "post_verdict=true")
	}
	if o.TrustedTarget {
		out = append(out, "trusted_target=true")
	}
	if o.TrustedBundle {
		out = append(out, "trusted_bundle=true")
	}
	if o.MaxIterations != 0 {
		out = append(out, "max_iterations="+strconv.Itoa(o.MaxIterations))
	}
	if o.BaseRef != "" {
		// Worth recording above all the others: it decides WHAT was reviewed, so a
		// summary that omitted it would describe findings without their scope.
		out = append(out, "base_ref="+o.BaseRef)
	}
	if o.PR != 0 {
		// Recorded for the same reason as base_ref: in pr mode it decides WHAT was
		// reviewed.
		out = append(out, "pr="+strconv.Itoa(o.PR))
	}
	if o.Target != "" {
		// Same reason again: it decides WHAT was reviewed.
		out = append(out, "target="+o.Target)
	}
	return out
}

// LoadBundle compiles a run's configuration from a bundle name (or a path),
// anchored at projectRoot: resolve the task config, apply `extends` inheritance,
// load every referenced agent from agents/<name>.yaml, resolve every prompt to a
// path, apply defaults and path anchoring, and fold in ov. The result is the
// EFFECTIVE configuration and is immutable -- see Loaded.
//
// Resolution stays eager and complete here -- an unresolvable prompt or agent must
// fail before any agent process starts, not mid-run after tokens are spent.
//
// It does not call Validate. The caller runs the project-supplied-policy trust gate
// (which reads the now-final TrustedTarget) and then Loaded.Validate, so a
// configuration that is both untrusted and invalid still reports the trust refusal
// first, and a validation failure is still preceded by the resolved-source listing
// that usually explains it.
func LoadBundle(r *Resolver, nameOrPath, projectRoot string, ov Overrides) (*Loaded, error) {
	path, err := r.Config(nameOrPath)
	if err != nil {
		return nil, err
	}
	cfg, extendsPath, ownInline, err := loadWithExtends(r, path)
	if err != nil {
		return nil, err
	}
	src := Source{
		Config:  path,
		Extends: extendsPath,
		Agents:  map[string]string{},
		Prompts: map[string]string{},
	}
	// Defaults first, then anchor: logs.dir gets its default value in
	// applyDefaults, and anchoring before that would leave the default relative to
	// the working directory -- scattering artifact directories through the project.
	cfg.applyDefaults()
	cfg.anchor(projectRoot)
	if err := cfg.resolveAgents(r, src.Agents); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.resolvePrompts(r, src.Prompts); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	// Last, so the operator's assertions win over every file in the bundle, and so
	// Validate (run by the caller) sees the value the run will actually use.
	ov.apply(cfg)
	if err := ov.applyTarget(cfg); err != nil {
		return nil, err
	}
	l := &Loaded{Config: cfg, Source: src, ProjectRoot: projectRoot, Overrides: ov, ownInlineAgents: ownInline}
	// Refused while the configuration is compiled rather than left to the caller's
	// trust gate: no flag rescues it (see rejectProjectSuppliedInheritAll), so making
	// it part of loading covers every entry point -- a run, --check, --check-live --
	// without each one having to remember the check.
	if err := l.rejectProjectSuppliedInheritAll(); err != nil {
		return nil, err
	}
	return l, nil
}

// Validate checks the effective configuration, naming the task config the run was
// built from: with `extends` and a per-file search path, "which file is wrong?" is
// not answerable from the message alone.
func (l *Loaded) Validate() error {
	if err := l.Config.Validate(); err != nil {
		return fmt.Errorf("%s: %w", l.Source.Config, err)
	}
	return nil
}

// loadWithExtends decodes a task config and, when it names a base with
// `extends`, merges it over that base. Inheritance is ONE level deep on purpose:
// a chain makes the effective value of any field require reading N files, and the
// whole point of naming the base explicitly is that the reader can see it.
func loadWithExtends(r *Resolver, path string) (*Config, string, map[string]bool, error) {
	if err := rejectTrustKeys(path); err != nil {
		return nil, "", nil, err
	}
	cfg, err := decodeFile(path)
	if err != nil {
		return nil, "", nil, err
	}
	// Recorded before the merge, which is the only moment the two `agents:` maps are
	// still distinguishable: the provenance checks need to know which file declared
	// an inline agent (see Loaded.ownInlineAgents).
	own := make(map[string]bool, len(cfg.Agents))
	for name := range cfg.Agents {
		own[name] = true
	}
	if cfg.Extends == "" {
		return cfg, "", own, nil
	}
	basePath, err := r.configByName(cfg.Extends)
	if err != nil {
		return nil, "", nil, fmt.Errorf("%s: extends: %w", path, err)
	}
	// The base is checked too: inheritance would otherwise be the way around the
	// rule, since a hostile bundle can ship both files.
	if err := rejectTrustKeys(basePath); err != nil {
		return nil, "", nil, err
	}
	base, err := decodeFile(basePath)
	if err != nil {
		return nil, "", nil, err
	}
	if base.Extends != "" {
		return nil, "", nil, fmt.Errorf("%s: extends %s, which itself extends %s; inheritance is one level deep so the effective configuration stays readable from two files",
			path, cfg.Extends, base.Extends)
	}
	// Re-decode the child over the base: yaml overwrites only the keys the child
	// actually sets, which gives per-key override without a hand-written merge (and
	// without a deep merge's surprises). A list the child sets REPLACES the base's
	// list rather than appending -- appending would make an inherited entry
	// impossible to remove, and the entries that must never be dropped are
	// enforced in code instead (see mandatoryExcludes).
	merged := base
	if err := decodeInto(path, merged); err != nil {
		return nil, "", nil, err
	}
	merged.Extends = cfg.Extends
	return merged, basePath, own, nil
}

// The two reasons a key is refused here, spelled out for the operator reading the
// error: both start from the same fact -- the first bundle on the search path is
// the target's own -- and differ in what the key would buy the repository that
// shipped it.
const (
	whyNotTrust = "Configs are searched in the target's own directory first, so a config that could grant trust would let reviewed code authorize fixpoint to execute its agent definitions and run the coder against it -- the very thing that assertion is meant to gate"
	whyNotPost  = "Configs are searched in the target's own directory first, so a config that could turn publishing on would let reviewed code arrange for a review -- or, with post_verdict, an approval -- to be published on its own pull request under the operator's identity"
)

// trustKeys are the authorization keys a task config may not set: the trust
// assertions and the publishing switches. They are deliberately absent from the
// Loop and Review structs (see Loop.TrustedTarget and Review.Post for why), so the
// decoder would already reject them as unknown fields -- but as "field
// trusted_target not found in type config.Loop", which reads like a schema
// mismatch to fix rather than a boundary being enforced. Naming them here is what
// turns the refusal into an explanation.
var trustKeys = []struct {
	section, key, flag, why string
}{
	{"loop", "trusted_target", "-trusted-target", whyNotTrust},
	{"loop", "trusted_bundle", "-trusted-bundle", whyNotTrust},
	{"loop", "allow_untrusted_fix", "-allow-untrusted-fix", whyNotTrust},
	{"review", "post", "-post", whyNotPost},
	{"review", "post_verdict", "-post-verdict", whyNotPost},
}

// rejectTrustKeys fails when a task config tries to assert its own trust or turn
// on publishing.
//
// Configs are resolved from <project>/config first, so this file may well have
// come from the repository being reviewed: a config that could grant trust would
// let the code under review authorize executing its own agent definitions and
// running the write-capable coder against itself, and one that could set
// review.post would let it publish on its own pull request as the operator.
func rejectTrustKeys(path string) error {
	data, err := readBundleFile(path)
	if err != nil {
		return err
	}
	// A permissive probe: this runs BEFORE the strict decode, so it must not fail
	// on unrelated keys and steal the better error message the real decode gives.
	var probe struct {
		Loop   map[string]yaml.Node `yaml:"loop"`
		Review map[string]yaml.Node `yaml:"review"`
	}
	if err := yaml.Unmarshal(data, &probe); err != nil {
		// Not this function's error to report: it runs BEFORE the strict decode, so
		// returning a parse error here would replace the decoder's precise message
		// (with line and column) with a worse one from a probe the caller did not
		// ask about. Nothing is skipped by continuing -- a file the permissive probe
		// cannot parse cannot be parsed by the strict decode either, so it fails a
		// few lines later and never reaches a run.
		return nil //nolint:nilerr // deliberate: the strict decode reports this file's syntax properly
	}
	sections := map[string]map[string]yaml.Node{"loop": probe.Loop, "review": probe.Review}
	for _, tk := range trustKeys {
		if _, ok := sections[tk.section][tk.key]; ok {
			return fmt.Errorf("%s: %s.%s cannot be set in a configuration file; pass %s on the command line instead. %s",
				path, tk.section, tk.key, tk.flag, tk.why)
		}
	}
	return nil
}

func decodeFile(path string) (*Config, error) {
	var cfg Config
	if err := decodeInto(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// decodeInto decodes a YAML file over an existing value, which is what gives
// `extends` its per-key override: decoding the child over the base leaves keys
// the child omits untouched. Unknown keys are errors, so a typo in any bundle
// file fails at startup instead of being silently ignored.
func decodeInto(path string, target any) error {
	// readBundleFile rather than os.ReadFile: `extends` names a config the file
	// under review chose, and this decode happens before the trust gate, so the
	// same regular-file and size limits every other bundle read has apply here too.
	data, err := readBundleFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // typos in keys are errors, not silent defaults
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// resolveAgents loads every agent the run references from its own file. Agents
// live in one file each so a task config states which agents it uses without
// restating how to invoke them -- and so fixing an agent's command fixes it for
// every config at once.
func (c *Config) resolveAgents(r *Resolver, into map[string]string) error {
	names := c.referencedAgents()
	if c.Agents == nil {
		c.Agents = map[string]Agent{}
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, done := c.Agents[name]; done {
			continue
		}
		path, err := r.Agent(name)
		if err != nil {
			return err
		}
		var a Agent
		if err := decodeInto(path, &a); err != nil {
			return err
		}
		a.applyDefaults()
		c.Agents[name] = a
		into[name] = path
	}
	return nil
}

// referencedAgents names the agents a run is built around: every active reviewer
// plus the coder and the judge. Resolution and the env.inherit_all gate judge
// exactly this set, so "the agents this configuration is about" has one
// definition. The coder is included even for -review-only: it is resolved
// eagerly there too, and an agent declaration that would be refused is better
// refused before a later invocation makes it live. The judge is included for the
// same reason and because it is invoked on the review-only path, where it reads
// the untrusted code and the findings written about it -- an inline-only judge
// agent that never appears here would skip resolution and slip past the
// unconditional env.inherit_all refusal. Unset roles contribute an empty name,
// which resolveAgents and the refusal loop already skip.
func (c *Config) referencedAgents() []string {
	return append(c.Roles.Review.ActiveAgents(), c.Roles.Coder.Agent, c.Roles.Judge.Agent, c.Roles.Triage.Agent, c.Roles.Editor.Agent)
}

// resolvePrompts turns every bare prompt name into a concrete file path, stored
// alongside the name so logs keep reporting the name while the loader reads the
// file. Resolution happens for all prompts up front so a missing one fails at
// startup.
func (c *Config) resolvePrompts(r *Resolver, into map[string]string) error {
	resolve := func(name string) (string, error) {
		if name == "" {
			return "", nil
		}
		if p, done := into[name]; done {
			return p, nil
		}
		p, err := r.Prompt(name)
		if err != nil {
			return "", err
		}
		into[name] = p
		return p, nil
	}
	if c.Roles.Coder.Prompt != "" {
		p, err := resolve(c.Roles.Coder.Prompt)
		if err != nil {
			return err
		}
		c.Roles.Coder.PromptPath = p
	}
	for i := range c.Roles.Review.Prompts {
		p, err := resolve(c.Roles.Review.Prompts[i].Prompt)
		if err != nil {
			return err
		}
		c.Roles.Review.Prompts[i].PromptPath = p
	}
	if c.Roles.Judge.Prompt != "" {
		p, err := resolve(c.Roles.Judge.Prompt)
		if err != nil {
			return err
		}
		c.Roles.Judge.PromptPath = p
	}
	if c.Roles.Triage.Prompt != "" {
		p, err := resolve(c.Roles.Triage.Prompt)
		if err != nil {
			return err
		}
		c.Roles.Triage.PromptPath = p
	}
	if c.Roles.Editor.Prompt != "" {
		p, err := resolve(c.Roles.Editor.Prompt)
		if err != nil {
			return err
		}
		c.Roles.Editor.PromptPath = p
	}
	// The create prompts resolve like every other: eagerly, so a missing one
	// fails before any agent process starts.
	for _, pp := range []struct {
		name string
		into *string
	}{
		{c.Create.Propose, &c.Create.ProposePath},
		{c.Create.Critique, &c.Create.CritiquePath},
		{c.Create.Object, &c.Create.ObjectPath},
	} {
		if pp.name == "" {
			continue
		}
		p, err := resolve(pp.name)
		if err != nil {
			return err
		}
		*pp.into = p
	}
	if c.Review.Refute != "" {
		p, err := resolve(c.Review.Refute)
		if err != nil {
			return err
		}
		c.Review.RefutePath = p
	}
	return nil
}

// anchor makes every relative path in the configuration resolve against the
// project root rather than the working directory, so a run behaves identically
// from anywhere inside the project. Without this, `fixpoint fix-code` from a
// subdirectory would review only that subtree and create its artifact directory
// there.
func (c *Config) anchor(projectRoot string) {
	if projectRoot == "" {
		return
	}
	if c.Target.Path == "" || c.Target.Path == "." {
		c.Target.Path = projectRoot
	} else if !filepath.IsAbs(c.Target.Path) {
		c.Target.Path = filepath.Join(projectRoot, c.Target.Path)
	}
	if c.Logs.Dir != "" && !filepath.IsAbs(c.Logs.Dir) {
		c.Logs.Dir = filepath.Join(projectRoot, c.Logs.Dir)
	}
}

// ProjectSuppliedPolicy reports the bundle files that were resolved from INSIDE
// the project under review. It returns nil when none were.
//
// This closes a hole opened by making bundles shadowable per project. Resolution
// prefers <project>/config, so a repository can ship its own copy of any file a
// run is built from -- and every one of them is policy authored by the code being
// reviewed, not data:
//
//   - an agents/<name>.yaml command, or an `agents:` map defined INLINE in the task
//     config, is argv fixpoint executes (in the preflight ping and once per role);
//   - a verify command is argv fixpoint executes;
//   - a prompt is the instruction stream handed verbatim to an agent that can read
//     any file the invoking user can, and whose findings are printed, persisted,
//     and echoed into a fix run's commit body;
//   - the task config chooses all three, plus the target and the trust-relevant
//     shape of the run.
//
// So EVERY resolved file counts, whatever it contains, rather than an enumeration
// of the dangerous keys. Enumerating is what let inline `agents:` maps through: the
// list only covered agents loaded from their own file (resolveAgents records those
// in Source.Agents and skips the lookup entirely for an inline name), so a hostile
// task config carrying its own commands was invisible to the gate. A content-based
// list fails OPEN every time config surface grows; this one fails closed.
//
// Containment is measured against ProjectRoot -- the directory the bundle search
// path is anchored to, and so the directory a shadowing copy has to be planted in
// -- and additionally against target.path. Measuring against target.path ALONE let
// the config choose the boundary it is judged by: a hostile config declaring
// `target: {path: ./src}` put every one of its sibling bundle files outside "the
// target" and straight past the gate.
//
// The consequence is deliberate: pointing fixpoint at a project that carries its
// own bundle requires an explicit trust assertion even for a review-only run. The
// alternative is deciding, per key, which of the reviewed repository's own policy
// is harmless.
//
// The caller gates on this, and either -trusted-bundle or -trusted-target
// satisfies it. -trusted-bundle is the assertion that says only what this list is
// about; -trusted-target says the same and more (see Loop.TrustedBundle), so a
// caller that needs no more than bundle trust must not reach for it -- in pr mode
// the wider flag also disarms guards over content `gh pr checkout` has not written
// yet.
func (l *Loaded) ProjectSuppliedPolicy() []string {
	if l.ProjectRoot == "" {
		return nil
	}
	var out []string
	add := func(kind, name, path string) {
		if path == "" || !l.fromProject(path) {
			return
		}
		if name == "" {
			out = append(out, kind+" "+path)
			return
		}
		out = append(out, fmt.Sprintf("%s %s (%s)", kind, name, path))
	}
	// The config first: it is the file that names all the others, and the one an
	// inline `agents:` map or a verify command lives in.
	add("config", "", l.Source.Config)
	add("extends base", "", l.Source.Extends)
	for name, path := range l.Source.Agents {
		add("agent", name, path)
	}
	for name, path := range l.Source.Prompts {
		add("prompt", name, path)
	}
	sort.Strings(out) // stable message regardless of map iteration order
	return out
}

// rejectProjectSuppliedInheritAll fails when an agent this run is built around
// declares env.inherit_all in a file that came from INSIDE the project under
// review.
//
// env.inherit_all makes buildEnv return nil, which exec reads as "inherit the
// parent environment": the agent process then receives every exported secret
// fixpoint was invoked with, including the bespoke ones whose names neither the
// credential-shape rule nor the redactor recognizes -- and a reviewer can quote
// any of them into a finding that is logged, persisted and echoed into a commit
// body. Declaring the variables under env.pass exposes the same values only when
// the agent genuinely needs them, and leaves the exposure enumerated in a file a
// reader can audit and in the run's own provenance log.
//
// The refusal is unconditional, which is what separates this from
// ProjectSuppliedPolicy's -trusted-bundle gate: that assertion says the target's
// policy may be EXECUTED, not that the target may help itself to secrets it
// cannot even name. It is the same boundary as the one that keeps
// FIXPOINT_KEEP_ENV out of YAML (see internal/agent/env.go) -- the environment
// fixpoint was invoked with is a channel the reviewed code must not write to --
// and the same one that makes the trust keys flag-only. An operator who really
// wants full inheritance still can: declare it in a bundle outside the target,
// which is the copy no reviewed commit can change.
func (l *Loaded) rejectProjectSuppliedInheritAll() error {
	if l.ProjectRoot == "" {
		return nil
	}
	for _, name := range l.Config.referencedAgents() {
		if name == "" || !l.Config.Agents[name].Env.InheritAll {
			continue
		}
		for _, path := range l.agentDefinitionPaths(name) {
			if !l.fromProject(path) {
				continue
			}
			return fmt.Errorf("%s: agents.%s sets env.inherit_all, which cannot be set by a file inside the target: "+
				"the agent would receive fixpoint's ENTIRE environment, so every exported secret (cloud credentials, database passwords, tokens for other services) "+
				"is readable by a process the reviewed code configured, including the ones no denylist or redactor knows by name. "+
				"Declare the variables the agent actually needs under env.pass, or supply the agent from a bundle outside the target "+
				"-- unlike -trusted-target this is not something the assertion can grant", path, name)
		}
	}
	return nil
}

// agentDefinitionPaths names the files that could carry one agent's declaration,
// for provenance checks. An agent loaded from agents/<name>.yaml is judged by that
// file; one defined INLINE has no file of its own, so it is judged by the task
// config -- which named the agent and chose the base -- plus, when the entry came
// from the base, the base as well.
//
// The base is dropped once the config declares the agent itself because the merge
// is per key of the `agents:` map: a child that names an agent replaces the base's
// whole entry, command included, rather than merging into it. So an operator's own
// config that extends a project-shipped base still owns every field of the agents
// it declares, and refusing it for the base's mere existence would close the one
// escape hatch env.inherit_all has (see rejectProjectSuppliedInheritAll) for
// everyone who inherits from the project.
func (l *Loaded) agentDefinitionPaths(name string) []string {
	if p := l.Source.Agents[name]; p != "" {
		return []string{p}
	}
	if l.ownInlineAgents[name] {
		return []string{l.Source.Config}
	}
	return []string{l.Source.Config, l.Source.Extends}
}

// fromProject reports whether a bundle file lies inside the code the run does not
// trust: the project the bundle search path is anchored to, or the review target
// when that points somewhere else. Either is enough -- a file in either tree may
// have been shipped by the code under review.
func (l *Loaded) fromProject(path string) bool {
	// An absent file is not a file the project supplied. Source.Extends is empty
	// whenever a config inherits from nothing, and filepath.Abs("") resolves to the
	// WORKING DIRECTORY -- normally inside the project -- so an unset path measured
	// like a real one would read an operator's own bundle as target-supplied.
	if path == "" {
		return false
	}
	if withinTree(path, l.ProjectRoot) {
		return true
	}
	return l.Config.Target.Path != "" && withinTree(path, l.Config.Target.Path)
}

// withinTree reports whether path lies inside root either lexically or with every
// symlink on both sides resolved.
//
// The resolved comparison is what makes this gate hold, because a bundle DIRECTORY
// is deliberately allowed to be a symlink -- withinBundle resolves both sides for
// exactly that reason, so ~/.fixpoint pointing into a dotfiles checkout keeps
// working. Point that same link at a directory in the repository under review and
// every file resolved from the "user" bundle keeps a lexical path under ~/.fixpoint
// while really living inside the target: a lexical test reads policy the reviewed
// commit can rewrite as operator-owned, so an inline agent or agents/*.yaml command
// runs during the preflight ping with no -trusted-target, and
// rejectProjectSuppliedInheritAll hands it the whole parent environment as well.
//
// Both spellings count rather than the resolved one alone: a lexical hit is already
// proof the file sits in a tree the reviewed code can write, and resolving can only
// move a path OUT of one tree into another.
func withinTree(path, root string) bool {
	if within(path, root) {
		return true
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Every path measured here was resolved from a bundle and read moments ago, so
		// failing to canonicalize one now means the filesystem changed underneath the
		// run. The answer decides whether the reviewed code's own policy may execute,
		// so unknown provenance counts as the project's.
		return true
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		// A root that cannot be canonicalized -- a target.path that does not exist --
		// keeps its lexical form's verdict. Target validation reports that far more
		// clearly than a trust refusal naming an unrelated bundle file would.
		return false
	}
	return within(realPath, realRoot)
}

// within reports whether path lies inside root.
func within(path, root string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
