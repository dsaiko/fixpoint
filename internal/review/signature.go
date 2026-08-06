package review

import (
	"runtime/debug"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/forge"
)

// DefaultSignature is what a review is signed with when a config says nothing.
//
// It answers the three questions a human asks of a machine-authored review they
// did not expect: what wrote this, what did it consult, and where is the rest of
// it. What it deliberately does NOT do is name the tool.
//
// Reviews get posted into other people's repositories, where "fixpoint" means
// nothing to the reader and reads as an unexplained internal name. "An AI panel"
// is the fact that matters -- it tells the reader how much weight to give the
// comment. The run id stays because it is how the operator finds the artifacts;
// the agent names stay because which models looked is the substance of the claim.
const DefaultSignature = "🤖 Reviewed by AI panel · {agents} · run {run}"

// Signature renders the operator's template.
//
// It is rendered by fixpoint from facts fixpoint holds, and it is placed OUTSIDE
// every region carrying agent text -- see RenderBody. That placement is the whole
// security property: a signature composed from a finding's prose could be forged
// by whatever wrote that prose, and a forged signature on a machine review is a
// forged attribution on someone's pull request.
//
// Unknown placeholders are left verbatim rather than blanked, so a typo shows up
// in the output as "{agnets}" instead of silently deleting itself.
func Signature(tmpl string, f SignatureFacts) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = DefaultSignature
	}
	agents := append([]string(nil), f.Agents...)
	sort.Strings(agents)
	r := strings.NewReplacer(
		"{agents}", strings.Join(agents, ", "),
		"{run}", f.Run,
		"{version}", f.Version,
		"{config}", f.Config,
		"{verdict}", f.Verdict,
	)
	// Through the SAME forge funnel as the body's text (mdText), for the same reason:
	// the config this template came from -- and the agent and config names it
	// substitutes -- may have been shipped by the repository under review. An agent
	// named `@victim` or `closer#1`, or a custom template writing `Closes #1`, would
	// otherwise make the posted review notify a stranger or close an issue under the
	// operator's identity, from the one region of the document that is placed after
	// all agent text precisely because it is meant to be fixpoint's own words.
	//
	// One line, after sanitizing: a signature that could carry a newline could append
	// a second, unsigned-looking paragraph to a posted review.
	return strings.ReplaceAll(forge.SanitizeText(r.Replace(tmpl)), "\n", " ")
}

// SignatureFacts are the values a signature may name. Every one is fixpoint's own
// knowledge of the run -- nothing here comes from an agent.
type SignatureFacts struct {
	Agents  []string
	Run     string
	Version string
	Config  string
	Verdict string
}

// Version is the fixpoint build behind a review, for the signature.
//
// It comes from the build info the Go toolchain embeds rather than from a
// variable someone has to remember to set at release time: a signature on someone
// else's pull request should be traceable even when the binary was built with a
// plain `go build`. A checkout with uncommitted changes says so, because "which
// fixpoint wrote this?" has a different answer then.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	switch {
	case rev == "":
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
		return "dev"
	case len(rev) > 12:
		rev = rev[:12]
	}
	if dirty {
		return rev + "-dirty"
	}
	return rev
}
