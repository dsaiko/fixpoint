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
}

// Overrides are the run's command-line assertions. They are applied while the
// configuration is compiled, not to a Config that already exists -- see Loaded.
//
// These are deliberately NOT config keys. The bundle may itself come from the
// repository under review, and code being reviewed must not be able to declare
// itself trustworthy; a trust gate is therefore an assertion the operator makes
// per invocation, which only a flag can express.
type Overrides struct {
	ReviewOnly        bool
	MaxIterations     int
	AllowUntrustedFix bool
	TrustedTarget     bool
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
	if o.TrustedTarget {
		c.Loop.TrustedTarget = true
	}
	// Any nonzero value applies, including a negative one: an explicit
	// -max-iterations -1 must reach Validate so its must-not-be-negative rule
	// rejects the bad input rather than being swallowed as "no flag supplied".
	// Zero stays "use config", per the flag help.
	if o.MaxIterations != 0 {
		c.Loop.MaxIterations = o.MaxIterations
	}
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
	if o.TrustedTarget {
		out = append(out, "trusted_target=true")
	}
	if o.MaxIterations != 0 {
		out = append(out, "max_iterations="+strconv.Itoa(o.MaxIterations))
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
// It does not call Validate. The caller runs the project-supplied-exec trust gate
// (which reads the now-final TrustedTarget) and then Loaded.Validate, so a
// configuration that is both untrusted and invalid still reports the trust refusal
// first, and a validation failure is still preceded by the resolved-source listing
// that usually explains it.
func LoadBundle(r *Resolver, nameOrPath, projectRoot string, ov Overrides) (*Loaded, error) {
	path, err := r.Config(nameOrPath)
	if err != nil {
		return nil, err
	}
	cfg, extendsPath, err := loadWithExtends(r, path)
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
	return &Loaded{Config: cfg, Source: src, ProjectRoot: projectRoot, Overrides: ov}, nil
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
func loadWithExtends(r *Resolver, path string) (*Config, string, error) {
	if err := rejectTrustKeys(path); err != nil {
		return nil, "", err
	}
	cfg, err := decodeFile(path)
	if err != nil {
		return nil, "", err
	}
	if cfg.Extends == "" {
		return cfg, "", nil
	}
	basePath, err := r.Config(cfg.Extends)
	if err != nil {
		return nil, "", fmt.Errorf("%s: extends: %w", path, err)
	}
	// The base is checked too: inheritance would otherwise be the way around the
	// rule, since a hostile bundle can ship both files.
	if err := rejectTrustKeys(basePath); err != nil {
		return nil, "", err
	}
	base, err := decodeFile(basePath)
	if err != nil {
		return nil, "", err
	}
	if base.Extends != "" {
		return nil, "", fmt.Errorf("%s: extends %s, which itself extends %s; inheritance is one level deep so the effective configuration stays readable from two files",
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
		return nil, "", err
	}
	merged.Extends = cfg.Extends
	return merged, basePath, nil
}

// trustKeys are the authorization keys a task config may not set. They are
// deliberately absent from the Loop struct (see Loop.TrustedTarget for why), so
// the decoder would already reject them as unknown fields -- but as "field
// trusted_target not found in type config.Loop", which reads like a schema
// mismatch to fix rather than a boundary being enforced. Naming them here is what
// turns the refusal into an explanation.
var trustKeys = []struct {
	key, flag string
}{
	{"trusted_target", "-trusted-target"},
	{"allow_untrusted_fix", "-allow-untrusted-fix"},
}

// rejectTrustKeys fails when a task config tries to assert its own trust.
//
// Configs are resolved from <project>/config first, so this file may well have
// come from the repository being reviewed: a config that could grant trust would
// let the code under review authorize executing its own agent definitions and
// running the write-capable coder against itself.
func rejectTrustKeys(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// A permissive probe: this runs BEFORE the strict decode, so it must not fail
	// on unrelated keys and steal the better error message the real decode gives.
	var probe struct {
		Loop map[string]yaml.Node `yaml:"loop"`
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
	for _, tk := range trustKeys {
		if _, ok := probe.Loop[tk.key]; ok {
			return fmt.Errorf("%s: loop.%s cannot be set in a configuration file; pass %s on the command line instead. "+
				"Configs are searched in the target's own directory first, so a config that could grant trust would let reviewed code authorize fixpoint to execute its agent definitions and run the coder against it -- the very thing that assertion is meant to gate",
				path, tk.key, tk.flag)
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
	data, err := os.ReadFile(path)
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
	names := append(c.Roles.Review.ActiveAgents(), c.Roles.Coder.Agent)
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
	p, err := resolve(c.Roles.Coder.Prompt)
	if err != nil {
		return err
	}
	c.Roles.Coder.PromptPath = p
	for i := range c.Roles.Review.Prompts {
		p, err := resolve(c.Roles.Review.Prompts[i].Prompt)
		if err != nil {
			return err
		}
		c.Roles.Review.Prompts[i].PromptPath = p
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

// ProjectSuppliedExec reports the bundle files that were resolved from INSIDE the
// review target and that fixpoint executes: agent commands and verification
// commands. It returns nil when none were.
//
// This closes a hole opened by making bundles shadowable per project. Resolution
// prefers <project>/config, so a repository can ship its own agents/*.yaml or a
// verify command -- and those are argv that fixpoint runs. That is direct code
// execution from a file in the target, with no model and no prompt injection
// involved, and it would otherwise happen even in a review-only run that promises
// to change nothing.
//
// The caller gates on this: executing a target's own definitions requires the same
// explicit trust assertion as letting the coder edit it.
func (l *Loaded) ProjectSuppliedExec() []string {
	if l.ProjectRoot == "" {
		return nil
	}
	target := l.Config.Target.Path
	if target == "" {
		target = l.ProjectRoot
	}
	var out []string
	for name, path := range l.Source.Agents {
		if within(path, target) {
			out = append(out, fmt.Sprintf("agent %s (%s)", name, path))
		}
	}
	if l.Config.Verify.Enabled() && within(l.Source.Config, target) {
		out = append(out, "verify commands in "+l.Source.Config)
	}
	if l.Config.Verify.Enabled() && l.Source.Extends != "" && within(l.Source.Extends, target) {
		out = append(out, "verify commands in "+l.Source.Extends)
	}
	sort.Strings(out) // stable message regardless of map iteration order
	return out
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
