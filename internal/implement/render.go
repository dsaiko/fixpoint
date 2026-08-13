package implement

import (
	"fmt"
	"strings"
)

// RenderHeader is what PLAN.md's provenance header states beyond the plan's
// own provenance block: how the coverage rule read the document and what gate
// the run was configured with, in words an operator can audit later.
type RenderHeader struct {
	// Coverage is `"##" (7 headings)` or exactly "unchecked" -- nothing in a
	// run report may imply the check ran when it did not (§4.2 rule 6).
	Coverage string
	// GateCommands are the operator's configured commands verbatim; empty
	// renders as ungated.
	GateCommands []string
}

// RenderMarkdown renders the validated plan as PLAN.md: fixpoint's own
// rendering, committed once in the bootstrap commit and never rewritten
// (§4.3). Every agent-authored string is expected to be flattened and
// redacted by the CALLER before it reaches this renderer -- the renderer is
// layout, not a sanitizer, and sanitizing twice would hide a missed call site.
func RenderMarkdown(p Plan, h RenderHeader) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Plan: %s\n\n", p.Project.Name)
	if p.Project.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Project.Summary)
	}

	b.WriteString("## Provenance\n\n")
	if pr := p.Provenance; pr != nil {
		fmt.Fprintf(&b, "- run: %s\n", pr.RunID)
		fmt.Fprintf(&b, "- planner: %s\n", pr.Planner)
		fmt.Fprintf(&b, "- coder: %s\n", pr.Coder)
		fmt.Fprintf(&b, "- design: %s (sha256 %s)\n", pr.DesignPath, pr.DesignSHA256)
		fmt.Fprintf(&b, "- verify profile: %s\n", pr.VerifyProfile)
	}
	fmt.Fprintf(&b, "- coverage: %s\n", h.Coverage)
	if len(h.GateCommands) == 0 {
		b.WriteString("- gate: ungated (no operator commands configured)\n")
	} else {
		fmt.Fprintf(&b, "- gate: %s\n", strings.Join(h.GateCommands, " && "))
	}
	b.WriteString("\n")

	b.WriteString("## Coverage\n\n")
	for _, c := range p.Coverage {
		switch {
		case len(c.Tasks) > 0:
			fmt.Fprintf(&b, "- %s → %s\n", c.Heading, strings.Join(c.Tasks, ", "))
		default:
			fmt.Fprintf(&b, "- %s → out of scope: %s\n", c.Heading, c.OutOfScope)
		}
	}
	b.WriteString("\n## Tasks\n")
	for _, t := range p.Tasks {
		fmt.Fprintf(&b, "\n### %s — %s\n\n", t.ID, t.Title)
		fmt.Fprintf(&b, "%s\n\n", t.Goal)
		if len(t.DependsOn) > 0 {
			fmt.Fprintf(&b, "- depends on: %s\n", strings.Join(t.DependsOn, ", "))
		}
		if len(t.DesignRefs) > 0 {
			fmt.Fprintf(&b, "- design: %s\n", strings.Join(t.DesignRefs, ", "))
		}
		fmt.Fprintf(&b, "- files: %s\n", strings.Join(t.Files, ", "))
		b.WriteString("- acceptance:\n")
		for _, a := range t.Acceptance {
			fmt.Fprintf(&b, "  - %s\n", a)
		}
	}
	return b.String()
}
