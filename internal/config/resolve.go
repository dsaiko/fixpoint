package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Layout of a config bundle. A bundle is a directory holding task configs at its
// root plus these two subdirectories, so one name resolves in one category:
//
//	<bundle>/full-review.yaml     a task config, referenced as "full-review"
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
// from dir for a marker: an existing bundle directory, or a git repository root.
// Everything relative in a run -- the review target and the artifact directory --
// resolves against this, NOT against the working directory, so `fixpoint
// full-review` behaves identically from the repository root and from three
// directories down. Anchoring to the working directory instead would silently
// review only the subtree you happened to stand in and scatter artifact
// directories through the project.
//
// It falls back to dir when no marker is found, which is the correct behavior
// for a non-git directory reviewed in place.
func ProjectRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for cur := abs; ; {
		for _, marker := range rootMarkers {
			// Existence, not directory-ness: in a git worktree or submodule ".git"
			// is a FILE pointing at the real git dir, and requiring a directory
			// would walk straight past the root of every worktree.
			if _, err := os.Lstat(filepath.Join(cur, marker)); err == nil {
				return cur, nil
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur { // reached the filesystem root
			return abs, nil
		}
		cur = parent
	}
}

// rootMarkers identify a project root when walking up. Deliberately NOT the
// bundle directory name: "config" is an extremely common package and directory
// name (this repository has internal/config), so treating it as a marker would
// detect internal/ as the project root and review only that subtree. A marker has
// to be something that appears once, at the top.
var rootMarkers = []string{".git", "." + appName}

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

// ListConfigs returns the task configs visible on the search path, keyed by name,
// with the bundle path each resolved from. Shadowed copies are omitted: the value
// is the one a run would actually use. Backs both `--list` and shell completion.
func (r *Resolver) ListConfigs() (map[string]string, error) {
	out := map[string]string{}
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
			if _, shadowed := out[name]; !shadowed {
				out[name] = filepath.Join(b, e.Name())
			}
		}
	}
	return out, errors.Join(errs...)
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
