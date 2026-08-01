package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
// It falls back to dir when no marker is found, which is the correct behavior
// for a non-git directory reviewed in place.
func ProjectRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	home := ""
	if h, err := os.UserHomeDir(); err == nil {
		home = filepath.Clean(h)
	}
	for cur := abs; ; {
		if isProjectRoot(cur, home) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur { // reached the filesystem root
			return abs, nil
		}
		cur = parent
	}
}

// isProjectRoot reports whether dir carries a marker that identifies it as the top
// of a project. home, when known, is excluded from the markers that the user's own
// files legitimately produce there.
//
// A marker has to be something that appears once, at the top. The bare bundle
// directory name is not one: "config" is an extremely common package and directory
// name (this repository has internal/config), so treating it as a marker would
// detect internal/ as the project root and review only that subtree. A project
// bundle therefore counts only when it has a bundle's shape.
func isProjectRoot(dir, home string) bool {
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
	if home != "" && filepath.Clean(dir) == home {
		return false
	}
	_, err := os.Lstat(filepath.Join(dir, "."+appName))
	return err == nil
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
	return r.find("config", "", nameOrPath+configExt)
}

// Prompt resolves a prompt by bare name to <bundle>/prompts/<name>.md.
func (r *Resolver) Prompt(name string) (string, error) {
	return r.find("prompt", promptsDir, withExt(name, promptExt))
}

// Agent resolves an agent by bare name to <bundle>/agents/<name>.yaml.
func (r *Resolver) Agent(name string) (string, error) {
	return r.find("agent", agentsDir, withExt(name, configExt))
}

func (r *Resolver) find(kind, sub, file string) (string, error) {
	tried := make([]string, 0, len(r.Bundles))
	for _, b := range r.Bundles {
		p := filepath.Join(b, sub, file)
		tried = append(tried, p)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", &NotFoundError{Kind: kind, Name: strings.TrimSuffix(file, filepath.Ext(file)), Tried: tried}
}

// Entry is one task config visible on the search path.
type Entry struct {
	Name string
	Path string
	// Description is the config's own one-line summary, empty if it has none.
	Description string
	// Runnable is false for a base config -- one that defines no review lenses and
	// so exists to be inherited via `extends`, not executed. Determined by shape
	// rather than by name: a bundle may hold several bases under any names, and
	// hardcoding "defaults" would be a rule that only happens to fit this bundle.
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
			if e.IsDir() || filepath.Ext(e.Name()) != configExt {
				continue
			}
			name := strings.TrimSuffix(e.Name(), configExt)
			if seen[name] {
				continue
			}
			seen[name] = true
			path := filepath.Join(b, e.Name())
			runnable, desc := probe(path)
			out = append(out, Entry{Name: name, Path: path, Description: desc, Runnable: runnable})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, errors.Join(errs...)
}

// probe reads the two things a listing needs from a config without loading it
// properly: whether it defines any review lens (which is what makes it runnable
// rather than a base for `extends`) and its description.
//
// Deliberately lenient -- a lone unknown key, which the strict loader rejects,
// must not make a config vanish from the listing. Listing is discovery; the loader
// is where correctness is enforced, with a message that says what is wrong.
func probe(path string) (runnable bool, description string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return true, "" // unreadable: let the loader report it properly
	}
	var p struct {
		Description string `yaml:"description"`
		Roles       struct {
			Review struct {
				Prompts []yaml.Node `yaml:"prompts"`
			} `yaml:"review"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return true, ""
	}
	return len(p.Roles.Review.Prompts) > 0, strings.TrimSpace(p.Description)
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
