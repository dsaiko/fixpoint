package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Layout of a config bundle. A bundle is a directory holding task configs at its
// root plus these two subdirectories, so one name resolves in one category:
//
//	<bundle>/fix-code.yaml        a task config, referenced as "fix-code"
//	<bundle>/prompts/fix.md       a prompt, referenced as "fix"
//	<bundle>/agents/claude.yaml   an agent, referenced as "claude"
const (
	promptsDir = "prompts"
	agentsDir  = "agents"

	configExt = ".yaml"
	promptExt = ".md"

	// bundleDirName is the conventional bundle directory inside a project and
	// under a user's home; a project keeps its bundle in <root>/config.
	projectBundleDir = "config"
	userBundleDir    = ".fixpoint"
	// appName is the subdirectory used under the OS config dir and the system
	// data dir.
	appName = "fixpoint"

	// maxBundleFile caps how much of a bundle file is read. Bundle files are
	// hand-written YAML and Markdown -- the largest in the shipped bundle is under
	// 2 KB -- so a megabyte is room the real files will never need.
	//
	// The cap exists because the FIRST bundle searched is <project>/config, inside
	// the repository under review, and these reads happen BEFORE the
	// project-supplied-policy gate can refuse the run: `fixpoint --list` reads every
	// config it finds there just to describe it. Without a ceiling, a target that
	// ships one enormous file makes merely looking at what it offers exhaust memory.
	maxBundleFile = 1 << 20
)

// systemBundleDirs returns the read-only, package-installed bundle locations.
// These are deliberately vendor-owned data directories rather than the
// admin-editable config location: deb and rpm treat files under /etc as
// conffiles and start prompting about local edits on every upgrade, whereas
// files under /usr/share are replaced cleanly. Users override by shadowing in a
// project or home bundle, never by editing the installed copy.
func systemBundleDirs() []string {
	if runtime.GOOS == "windows" {
		// ProgramData is the machine-wide equivalent: writable by installers,
		// readable by everyone, and not wiped per user profile.
		if pd := os.Getenv("ProgramData"); pd != "" {
			return []string{filepath.Join(pd, appName)}
		}
		return nil
	}
	if runtime.GOOS == "darwin" {
		// Homebrew, both Apple-silicon and Intel prefixes.
		return []string{
			"/opt/homebrew/share/" + appName,
			"/usr/local/share/" + appName,
			"/usr/share/" + appName,
		}
	}
	return []string{
		"/usr/local/share/" + appName,
		"/usr/share/" + appName,
	}
}

// ProjectRoot returns the directory a run is anchored to, found by walking up
// from dir for a marker: a git repository root, the project's own config bundle,
// or an artifact directory a previous run left behind. Everything relative in a
// run -- the review target and the artifact directory -- resolves against this,
// NOT against the working directory, so `fixpoint fix-code` behaves identically
// from the repository root and from three directories down. Anchoring to the
// working directory instead would silently review only the subtree you happened
// to stand in and scatter artifact directories through the project.
//
// It falls back to dir itself when no marker is found, which is the correct
// behavior for a non-git directory reviewed in place. The path returned is dir's
// canonical form whenever that can be determined, for the reason below.
func ProjectRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	// Walk the REAL directory, not the lexical one. Walking up is a purely lexical
	// operation, and `cd` through a symlink leaves $PWD pointing at the LINK, which
	// os.Getwd honors -- so with /tmp/cfg -> <repo>/internal/config the walk visits
	// /tmp/cfg, then /tmp, then /, finds no marker, and anchors the run to the linked
	// subtree: a fraction of the repository reviewed, artifacts written beside the
	// link. Resolving also keeps the root comparable to the canonical paths
	// everything downstream measures against (see target.fileScope).
	//
	// A path that will not resolve -- a directory that does not exist, or one whose
	// parent cannot be read -- keeps its lexical form: there is nothing better to
	// walk, and the fallback below still has an answer.
	start := abs
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		start = resolved
	}
	homes := homeDirs()
	for cur := start; ; {
		if isProjectRoot(cur, homes) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur { // reached the filesystem root
			return start, nil
		}
		cur = parent
	}
}

// homeDirs returns the forms of the user's home directory a walked path can show
// up as: the lexical one and, when it differs, the symlink-resolved one. Both are
// needed because ProjectRoot walks resolved paths when the starting directory can
// be resolved and lexical ones when it cannot, and because home itself may be a
// symlink (/home/u -> /mnt/data/u). The exclusion it feeds is a safety rule --
// anchoring a run to $HOME points the review at the whole home directory -- so
// matching either form is the side to err on. Empty when home is unknown.
func homeDirs() []string {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	homes := []string{filepath.Clean(h)}
	if resolved, err := filepath.EvalSymlinks(h); err == nil && resolved != homes[0] {
		homes = append(homes, resolved)
	}
	return homes
}

// isProjectRoot reports whether dir carries a marker that identifies it as the top
// of a project. homes, when known, are excluded from the markers that the user's
// own files legitimately produce there.
//
// A marker has to be something that appears once, at the top. The bare bundle
// directory name is not one: "config" is an extremely common package and directory
// name (this repository has internal/config), so treating it as a marker would
// detect internal/ as the project root and review only that subtree. A project
// bundle therefore counts only when it has a bundle's shape.
func isProjectRoot(dir string, homes []string) bool {
	// Existence, not directory-ness: in a git worktree or submodule ".git" is a
	// FILE pointing at the real git dir, and requiring a directory would walk
	// straight past the root of every worktree.
	if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	if isBundle(filepath.Join(dir, projectBundleDir)) {
		return true
	}
	// The artifact directory a previous run left behind (the default logs.dir
	// base) marks the root of a project that is not a git repository. Never in the
	// home directory though: there "." + appName is the documented USER bundle, so
	// honoring it would anchor every non-git project below home to the whole home
	// directory -- pointing the review at $HOME, writing artifacts there, and
	// classifying the user's own bundle as policy shipped by the code under review.
	if slices.Contains(homes, filepath.Clean(dir)) {
		return false
	}
	// Directory-ness, unlike the ".git" case above: only an artifact DIRECTORY is
	// that marker. A regular file that happens to be named "."+appName -- an
	// editor's dotfile, a stray note -- would otherwise outrank the real .git root
	// above it, anchoring the run to a nested subtree and then failing to create
	// the default artifact directory because the name is already taken by a file.
	st, err := os.Stat(filepath.Join(dir, "."+appName))
	return err == nil && st.IsDir()
}

// isBundle reports whether dir has the shape of a config bundle: at least one of
// the two subdirectories the layout above defines. Shape rather than name is what
// keeps an unrelated "config" directory -- a Go package, an application's settings
// folder -- from being mistaken for a project root.
func isBundle(dir string) bool {
	for _, sub := range []string{promptsDir, agentsDir} {
		if st, err := os.Stat(filepath.Join(dir, sub)); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

// Resolver locates config bundle files by name across an ordered search path.
// The first match wins, so a project can shadow one prompt while inheriting
// everything else from the user or system bundle.
type Resolver struct {
	// Bundles is the ordered search path. Earlier entries shadow later ones.
	Bundles []string
}

// NewResolver builds the standard search path for a project root, most specific
// first: the project's own bundle, the user's home bundle, the OS per-user config
// directory, then the package-installed bundles.
//
// There is deliberately no built-in fallback compiled into the binary. fixpoint's
// prompts drive an agent that edits files with permission checks disabled and its
// config holds the trust gates, so "what will this do to my repository?" must be
// answerable by reading files on disk. An embedded default would be unauditable,
// and worse, would let the tool behave in ways the visible files do not explain.
// When nothing resolves, resolution fails loudly instead (see NotFoundError).
func NewResolver(projectRoot string) *Resolver {
	var bundles []string
	if projectRoot != "" {
		bundles = append(bundles, filepath.Join(projectRoot, projectBundleDir))
	}
	if home, err := os.UserHomeDir(); err == nil {
		bundles = append(bundles, filepath.Join(home, userBundleDir))
	}
	if cfgDir, err := os.UserConfigDir(); err == nil {
		bundles = append(bundles, filepath.Join(cfgDir, appName))
	}
	return &Resolver{Bundles: append(bundles, systemBundleDirs()...)}
}

// NotFoundError reports a name that matched nothing. It lists every location
// searched, in order, because that list is the entire usability of a search path:
// without it a user cannot tell a typo from a missing bundle from a shadowing
// surprise.
type NotFoundError struct {
	Kind   string // "config", "prompt", or "agent"
	Name   string
	Tried  []string
	Reason string // optional extra hint
}

func (e *NotFoundError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %q not found. Searched, in order:\n", e.Kind, e.Name)
	for _, p := range e.Tried {
		fmt.Fprintf(&sb, "  %s\n", p)
	}
	if e.Reason != "" {
		fmt.Fprintf(&sb, "%s\n", e.Reason)
	}
	sb.WriteString("Install the fixpoint config bundle, or place one at " +
		filepath.Join("<project>", projectBundleDir) + " or " +
		filepath.Join("~", userBundleDir))
	return sb.String()
}

// Config resolves a task config. A name that looks like a path -- it contains a
// separator or already ends in .yaml -- is used directly, so an ad-hoc config
// outside any bundle still works.
func (r *Resolver) Config(nameOrPath string) (string, error) {
	if isPathLike(nameOrPath) {
		if _, err := os.Stat(nameOrPath); err != nil {
			return "", fmt.Errorf("config %s: %w", nameOrPath, err)
		}
		return nameOrPath, nil
	}
	return r.configByName(nameOrPath)
}

// configByName resolves a task config by bare name only, without Config's
// path-like escape hatch.
//
// `extends` resolves through this rather than through Config: the base is named
// by a config file that may itself have come from the repository under review,
// and a path there would let that file nominate any file on the host -- read, and
// fragments of it echoed back through the YAML decoder's error message, before
// the project-supplied-policy gate can refuse the run. A path is the OPERATOR's
// convenience for an ad-hoc config, so it stays on the command-line entry point.
func (r *Resolver) configByName(name string) (string, error) {
	if err := checkBundleName("config", name); err != nil {
		return "", err
	}
	return r.find("config", "", withExt(name, configExt))
}

// Prompt resolves a prompt by bare name to <bundle>/prompts/<name>.md.
func (r *Resolver) Prompt(name string) (string, error) {
	if err := checkBundleName("prompt", name); err != nil {
		return "", err
	}
	return r.find("prompt", promptsDir, withExt(name, promptExt))
}

// Agent resolves an agent by bare name to <bundle>/agents/<name>.yaml.
func (r *Resolver) Agent(name string) (string, error) {
	if err := checkBundleName("agent", name); err != nil {
		return "", err
	}
	return r.find("agent", agentsDir, withExt(name, configExt))
}

// checkBundleName rejects a reference that would resolve outside its bundle.
//
// The name is joined onto a bundle directory, so a separator or a dot segment
// walks the lookup OUT of the bundle: a task config asking for prompt
// "../../../../etc/passwd" would otherwise be handed that file's contents as an
// agent's instruction stream. The names are chosen by the task config, and the
// first bundle searched is <project>/config, which the repository under review
// may ship -- so this is untrusted input, not a typo check.
//
// It constrains only what containment needs. A name is otherwise free text that
// reaches an operator's terminal, and keeping THAT safe is the escaping layer's
// job (see agent.EscapeTerminal); duplicating the judgment here would leave two
// places to disagree about which names exist.
func checkBundleName(kind, name string) error {
	switch {
	case name == "":
		return fmt.Errorf("%s name is empty; a bundle reference names a file inside the bundle", kind)
	// ':' along with the separators: on Windows it introduces a drive-relative
	// path, which Join does not clean away either.
	case strings.ContainsAny(name, `/\:`):
		return fmt.Errorf("%s %q must not contain a path separator: it is joined onto the bundle directory, so the lookup would resolve outside the bundle -- "+
			"and the first bundle searched is the one the repository under review may ship", kind, name)
	case name == "." || name == "..":
		return fmt.Errorf("%s %q must not be a dot segment: it is joined onto the bundle directory, and the joined path is cleaned before it is used, so the lookup would resolve outside the bundle", kind, name)
	}
	return nil
}

func (r *Resolver) find(kind, sub, file string) (string, error) {
	tried := make([]string, 0, len(r.Bundles))
	for _, b := range r.Bundles {
		p := filepath.Join(b, sub, file)
		tried = append(tried, p)
		// Regular files only. A directory was never a match, and a device or a fifo
		// is worse than a non-match: reading /dev/zero through a planted symlink
		// never returns, and this lookup runs before the project-supplied-policy
		// gate has had a chance to refuse the target's bundle at all.
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() {
			continue
		}
		if !withinBundle(b, p) {
			return "", fmt.Errorf("%s %q resolves to %s, which lies outside its bundle %s: a bundle file must not be a symlink pointing out of the bundle. "+
				"The project's own bundle is searched first, so following one would let the repository under review nominate any file on the host as fixpoint's policy",
				kind, strings.TrimSuffix(file, filepath.Ext(file)), p, b)
		}
		return p, nil
	}
	return "", &NotFoundError{Kind: kind, Name: strings.TrimSuffix(file, filepath.Ext(file)), Tried: tried}
}

// withinBundle reports whether path, with every symlink along it resolved, still
// lies inside bundle -- itself resolved, so a bundle that IS a symlink (a
// ~/.fixpoint pointing into a dotfiles checkout, a Homebrew prefix) is compared
// against where it really lives and keeps working.
//
// What it rejects is the per-file escape: a single entry inside the bundle
// pointing somewhere else entirely. Only the resolved target can be checked here;
// a lexical test would see nothing wrong with a symlink at all.
func withinBundle(bundle, path string) bool {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	realBundle, err := filepath.EvalSymlinks(bundle)
	if err != nil {
		return false
	}
	return within(realPath, realBundle)
}

// readBundleFile reads a bundle file under the two limits every read of one
// needs: it must be an ordinary file, and it must be bounded.
//
// Both matter because <project>/config is searched first and belongs to the
// repository under review, and because the reads happen BEFORE the
// project-supplied-policy gate can refuse the run. A config that is a symlink to
// /dev/zero or to a fifo would otherwise hang `fixpoint --list` forever, and an
// enormous one would exhaust memory -- a target could deny the operator even the
// ability to look at what it ships. Stat before open, because opening a fifo is
// itself what blocks.
func readBundleFile(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file (%s); a bundle file must be an ordinary file", path, st.Mode().Type())
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // read-only: nothing to report on close
	// One byte past the cap, so hitting the limit is distinguishable from a file
	// that happens to be exactly maxBundleFile long.
	data, err := io.ReadAll(io.LimitReader(f, maxBundleFile+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(data) > maxBundleFile {
		return nil, fmt.Errorf("%s: bundle file is larger than %d bytes; configs, agents and prompts are hand-written files and this cap is what keeps reading an untrusted bundle bounded", path, maxBundleFile)
	}
	return data, nil
}

// Entry is one task config visible on the search path.
type Entry struct {
	Name string
	Path string
	// Description is the config's one-line summary, inherited from its `extends`
	// base when it sets none of its own; empty if neither has one.
	Description string
	// Runnable is false for a base config -- one that defines no review lenses, its
	// own or inherited, and so exists to be inherited via `extends`, not executed.
	// Determined by shape rather than by name: a bundle may hold several bases under
	// any names, and hardcoding "defaults" would be a rule that only happens to fit
	// this bundle.
	Runnable bool
}

// ListConfigs returns the configs visible on the search path, sorted by name, each
// with the file it resolved from. Shadowed copies are omitted: the entry is the one
// a run would actually use. Backs both `--list` and shell completion -- which
// should offer only the runnable ones.
func (r *Resolver) ListConfigs() ([]Entry, error) {
	seen := map[string]bool{}
	var out []Entry
	var errs []error
	for _, b := range r.Bundles {
		entries, err := os.ReadDir(b)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, err)
			}
			continue // a missing bundle is normal, not an error
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != configExt {
				continue
			}
			name := strings.TrimSuffix(e.Name(), configExt)
			if seen[name] {
				continue
			}
			path := filepath.Join(b, e.Name())
			// From here the entry is judged exactly as find judges it, so the listing
			// shows what a run would actually resolve -- including WHICH bundle gets to
			// answer for the name.
			//
			// Regular files only, and for find's reason: a directory was never a match,
			// and a device or a fifo planted through a symlink is worse than a miss.
			// Like find, this leaves the name unclaimed and lets a lower bundle answer.
			st, err := os.Stat(path)
			if err != nil || !st.Mode().IsRegular() {
				continue
			}
			// A symlink out of the bundle is the opposite case: find refuses the name
			// outright there rather than falling through, so a copy in a lower bundle is
			// NOT what a run would use. Claim the name so the listing omits it entirely
			// -- offering the lower copy would advertise a path every run refuses with
			// the containment error. Omitted rather than fatal, unlike the named lookup:
			// listing walks every bundle, so one hostile entry must not be able to take
			// `--list` down for all of them, and nothing is read from it, so the
			// description line of a file elsewhere on the host cannot leak into a
			// listing printed before any trust gate.
			if !withinBundle(b, path) {
				seen[name] = true
				continue
			}
			seen[name] = true
			runnable, desc := r.probeEffective(path)
			out = append(out, Entry{Name: name, Path: path, Description: desc, Runnable: runnable})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, errors.Join(errs...)
}

// probeEffective answers what a listing needs about a config as a RUN would see
// it, which means accounting for `extends`: the child is decoded over the base, so
// a config that names no lenses of its own still inherits the base's and runs
// perfectly (see loadWithExtends). Judging it on its own text alone would print a
// working config as a non-runnable base and drop it from shell completion. The
// description is inherited the same way, for the same reason.
//
// ONE level, matching the loader -- a base that itself extends is an error there,
// so there is no chain to follow here.
func (r *Resolver) probeEffective(path string) (runnable bool, description string) {
	runnable, desc, extends := probe(path)
	if extends == "" || (runnable && desc != "") {
		return runnable, desc
	}
	basePath, err := r.configByName(extends)
	if err != nil {
		// The base does not resolve, so this config cannot run -- but that is the
		// loader's error to report, naming the base it could not find. Reporting it
		// here as a base for others to inherit would be the one thing it is certainly
		// not: it asked to inherit.
		return true, desc
	}
	baseRunnable, baseDesc, _ := probe(basePath)
	if desc == "" {
		desc = baseDesc
	}
	return runnable || baseRunnable, desc
}

// probe reads the three things a listing needs from a config without loading it
// properly: whether it defines any review lens (which is what makes it runnable
// rather than a base for `extends`), its description, and the base it inherits
// from -- because either of the first two may come from that base instead.
//
// Deliberately lenient -- a lone unknown key, which the strict loader rejects,
// must not make a config vanish from the listing. Listing is discovery; the loader
// is where correctness is enforced, with a message that says what is wrong.
func probe(path string) (runnable bool, description, extends string) {
	data, err := readBundleFile(path)
	if err != nil {
		return true, "", "" // unreadable, oversized, or not a regular file: the loader reports it properly
	}
	var p struct {
		Description string `yaml:"description"`
		Extends     string `yaml:"extends"`
		Roles       struct {
			Review struct {
				Prompts []yaml.Node `yaml:"prompts"`
			} `yaml:"review"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return true, "", ""
	}
	return len(p.Roles.Review.Prompts) > 0, strings.TrimSpace(p.Description), strings.TrimSpace(p.Extends)
}

// isPathLike reports whether an argument should be treated as a filesystem path
// rather than a bundle name.
func isPathLike(s string) bool {
	return strings.ContainsRune(s, '/') || strings.ContainsRune(s, filepath.Separator) ||
		filepath.Ext(s) == configExt
}

// withExt appends ext unless the name already carries it, so both "review-bugs"
// and "review-bugs.md" resolve. Bare names are the documented form; tolerating
// the suffix avoids a pointless error for an understandable habit.
func withExt(name, ext string) string {
	if filepath.Ext(name) == ext {
		return name
	}
	return name + ext
}
