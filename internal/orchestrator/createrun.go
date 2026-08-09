package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/create"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
)

// The create-design pipeline's agent phases. The mechanics that are not agent
// scheduling -- snapshot, labels, caps, clamp -- live in internal/create; the
// pipeline itself is specified in docs/design/create-design.md.

// proposal is one PROPOSE session's product.
type proposal struct {
	agent string
	label string
	text  string
	step  model.StepStat
	err   error
}

// critique is one CRITIQUE session's product.
type critiqueResult struct {
	agent     string
	critiques []model.Critique
	step      model.StepStat
	err       error
}

// proposeAll runs the PROPOSE phase: every pool agent, independently, in the
// snapshot directory. No agent sees another's work -- independence is where the
// panel's value is, measured at under 4% corroboration -- so this is a plain
// fan-out with nothing shared but the assignment.
//
// A failed proposer is returned with its error rather than dropped here: the
// caller decides what the survivor count means (below two, the critique phase is
// skipped and the deliverable says so), and a step that spent tokens is billed
// whatever happened.
func (o *Orchestrator) proposeAll(ctx context.Context, snapDir, material string, capBytes int) []proposal {
	pool := append([]string(nil), o.cfg.Roles.Review.Agents...)
	sort.Strings(pool)
	labels := create.Labels(pool)

	o.phase("PROPOSE  %d agent(s), independently", len(pool))
	out := make([]proposal, len(pool))
	var wg sync.WaitGroup
	for i, name := range pool {
		wg.Add(1)
		// i and name are passed in rather than captured, matching every fan-out
		// here: the slot decides whose proposal is whose.
		go func(i int, name string) {
			defer wg.Done()
			out[i] = o.proposeWith(ctx, snapDir, material, name, labels[name], capBytes)
		}(i, name)
	}
	wg.Wait()

	ok := 0
	for _, p := range out {
		if p.err == nil {
			ok++
		}
	}
	o.endPhase("PROPOSE  %d of %d proposal(s)", ok, len(pool))
	return out
}

// proposeWith runs one agent's proposal session.
func (o *Orchestrator) proposeWith(ctx context.Context, snapDir, material, agentName, label string, capBytes int) proposal {
	p := proposal{agent: agentName, label: label}
	d := prompt.ProposeData{
		Path:           snapDir,
		ModeGuidance:   prompt.DocumentGuidance,
		Target:         material,
		Cap:            capNote(capBytes),
		OutputContract: prompt.ProposeContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: o.cfg.Target.Mode, Path: snapDir, ModeGuidance: d.ModeGuidance, Target: d.Target,
	})
	tmpl := o.templates[o.cfg.Create.Propose]
	if tmpl == nil {
		p.err = fmt.Errorf("propose prompt %q was never loaded", o.cfg.Create.Propose)
		return p
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		p.err = err
		return p
	}
	label2 := "propose: " + agentName
	res := o.runAgentIn(ctx, snapDir, label2, "propose", agentName, o.cfg.Create.Propose, 1, text)
	var doc string
	perr := res.Err
	if perr == nil {
		doc, perr = agent.ExtractText(res.Stdout, "design")
	}
	p.step = stepStat("propose", agentName, o.cfg.Create.Propose, len(text), res, perr != nil)
	// The artifact is the proposal itself, under the agent's REAL name: anonymous
	// in every prompt, attributable in the audit trail. This file is also where the
	// label mapping is persisted -- the header pairs them.
	md := fmt.Sprintf("# proposal %s (by %s)\n\n%s\n", p.label, agentName, doc)
	o.logStep("propose", agentName, o.cfg.Create.Propose, 1, perr == nil, nil, md, res, "")
	if perr != nil {
		o.logf("WARNING: %s failed (%v); the pipeline continues with the remaining proposals", label2, perr)
		p.err = perr
		return p
	}
	clamped := create.Clamp(doc, capBytes)
	if clamped != doc {
		o.logf("%s: the proposal exceeded the %d-byte cap and was elided (stated in place)", label2, capBytes)
	}
	p.text = clamped
	o.logf("%s done (proposal %s, %d bytes, %s)", label2, p.label, len(p.text), res.Duration)
	return p
}

// critiqueAll runs the CRITIQUE phase: every surviving proposer critiques the
// OTHERS' proposals. A critic never receives its own -- anonymization cannot
// blind an author to its own text -- and below two surviving proposals the
// caller skips this phase entirely, since there is nothing to compare.
func (o *Orchestrator) critiqueAll(ctx context.Context, snapDir string, proposals []proposal) []critiqueResult {
	alive := make([]proposal, 0, len(proposals))
	for _, p := range proposals {
		if p.err == nil {
			alive = append(alive, p)
		}
	}
	o.phase("CRITIQUE  %d critic(s), each reading the others' proposals", len(alive))
	out := make([]critiqueResult, len(alive))
	var wg sync.WaitGroup
	for i, p := range alive {
		wg.Add(1)
		go func(i int, self proposal) {
			defer wg.Done()
			out[i] = o.critiqueWith(ctx, snapDir, self, alive)
		}(i, p)
	}
	wg.Wait()

	ok := 0
	for _, c := range out {
		if c.err == nil {
			ok++
		}
	}
	o.endPhase("CRITIQUE  %d of %d critique(s)", ok, len(alive))
	return out
}

// critiqueWith runs one critic's session over everybody else's proposals.
func (o *Orchestrator) critiqueWith(ctx context.Context, snapDir string, self proposal, all []proposal) critiqueResult {
	c := critiqueResult{agent: self.agent}
	labeled := make([][2]string, 0, len(all)-1)
	shown := map[string]bool{}
	for _, p := range all {
		if p.agent == self.agent {
			continue
		}
		labeled = append(labeled, [2]string{p.label, p.text})
		shown[p.label] = true
	}
	d := prompt.CritiqueData{
		Path:           snapDir,
		ModeGuidance:   prompt.DocumentGuidance,
		Proposals:      prompt.FormatProposals(labeled),
		OutputContract: prompt.CritiqueContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: o.cfg.Target.Mode, Path: snapDir, ModeGuidance: d.ModeGuidance,
	})
	tmpl := o.templates[o.cfg.Create.Critique]
	if tmpl == nil {
		c.err = fmt.Errorf("critique prompt %q was never loaded", o.cfg.Create.Critique)
		return c
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		c.err = err
		return c
	}
	label := "critique: " + self.agent
	res := o.runAgentIn(ctx, snapDir, label, "critique", self.agent, o.cfg.Create.Critique, 1, text)
	var out model.CritiqueOutput
	perr := res.Err
	if perr == nil {
		perr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	c.step = stepStat("critique", self.agent, o.cfg.Create.Critique, len(text), res, perr != nil)
	o.logStep("critique", self.agent, o.cfg.Create.Critique, 1, perr == nil, out,
		renderCritiqueMD(self.agent, out.Critiques, perr), res, "")
	if perr != nil {
		o.logf("WARNING: %s failed (%v); its critiques do not reach the editor", label, perr)
		c.err = perr
		return c
	}
	// A critique of a proposal nobody showed this critic is either a hallucinated
	// label or its OWN proposal recognized -- both are judgments the phase must not
	// carry. An empty critique is indistinguishable from not having read the text.
	for _, cr := range out.Critiques {
		switch {
		case !shown[cr.Proposal]:
			o.logf("WARNING: %s critiqued proposal %q, which it was not shown; ignored", label, cr.Proposal)
		case cr.Empty():
			o.logf("WARNING: %s returned an empty critique of proposal %s; ignored", label, cr.Proposal)
		default:
			c.critiques = append(c.critiques, cr)
		}
	}
	o.logf("%s done (%d critique(s), %s)", label, len(c.critiques), res.Duration)
	return c
}

// capNote states the proposal cap inside the prompt, or nothing when unbounded.
// Stated because the clamp that enforces it announces its elision -- a cap the
// writer was never told about would read as the tool cutting text arbitrarily.
func capNote(capBytes int) string {
	if capBytes <= 0 {
		return ""
	}
	return fmt.Sprintf("Keep the proposal under %d bytes; text past that limit is elided with a notice.", capBytes)
}

// renderCritiqueMD is the human-readable artifact of one critic's session.
func renderCritiqueMD(agentName string, critiques []model.Critique, err error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# critiques by %s\n\n", agentName)
	if err != nil {
		fmt.Fprintf(&b, "**failed:** %v\n", err)
		return b.String()
	}
	for _, c := range critiques {
		fmt.Fprintf(&b, "## proposal %s\n\n", c.Proposal)
		writeList(&b, "Strengths", c.Strengths)
		writeList(&b, "Weaknesses", c.Weaknesses)
		writeList(&b, "Would adopt", c.Adopt)
	}
	return b.String()
}

func writeList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "**%s**\n\n", title)
	for _, it := range items {
		fmt.Fprintf(b, "- %s\n", it)
	}
	b.WriteString("\n")
}
