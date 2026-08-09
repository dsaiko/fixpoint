package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/create"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/target"
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

// synthesize runs the editor over everything the run has produced: the
// assignment, every surviving proposal, every carried critique. The editor is
// the answer to "how do the models agree" -- they do not, and someone must hold
// the pen.
//
// It fails CLOSED, and the run with it: nobody else may hold the pen, and
// falling back to "use the best proposal verbatim" would publish an unreviewed
// single voice under a synthesis's name. The one structural demand -- a
// "Decisions and dissent" section -- is checked here, because a synthesis that
// erases disagreement is the failure the whole pipeline exists to avoid.
func (o *Orchestrator) synthesize(ctx context.Context, snapDir, material string, proposals []proposal, critiques []critiqueResult) (string, model.StepStat, error) {
	e := o.cfg.Roles.Editor
	var labeled [][2]string
	for _, p := range proposals {
		if p.err == nil {
			labeled = append(labeled, [2]string{p.label, p.text})
		}
	}
	byCritic := make([][]model.Critique, 0, len(critiques))
	for _, c := range critiques {
		if c.err == nil && len(c.critiques) > 0 {
			byCritic = append(byCritic, c.critiques)
		}
	}
	o.phase("SYNTHESIZE  %s over %d proposal(s) and %d critique set(s)", e.Agent, len(labeled), len(byCritic))
	d := prompt.EditorData{
		Path:           snapDir,
		ModeGuidance:   prompt.DocumentGuidance,
		Target:         material,
		Proposals:      prompt.FormatProposals(labeled),
		Critiques:      prompt.FormatCritiques(byCritic),
		OutputContract: prompt.EditorContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: o.cfg.Target.Mode, Path: snapDir, ModeGuidance: d.ModeGuidance, Target: d.Target,
	})
	doc, step, err := o.editorSession(ctx, snapDir, "synthesize", d)
	if err != nil {
		o.endPhase("SYNTHESIZE  failed; the run fails with it")
		return "", step, err
	}
	o.endPhase("SYNTHESIZE  %d bytes", len(doc))
	return doc, step, nil
}

// objectionResult is one panel member's OBJECT pass.
type objectionResult struct {
	agent      string
	objections []model.Objection
	step       model.StepStat
	err        error
}

// objectAll runs the bounded objection pass: every pool agent reads the final
// draft and may raise blocking defects only. Zero objections is a normal answer.
func (o *Orchestrator) objectAll(ctx context.Context, snapDir, draft string) []objectionResult {
	pool := append([]string(nil), o.cfg.Roles.Review.Agents...)
	sort.Strings(pool)
	o.phase("OBJECT  %d agent(s) reading the final draft", len(pool))
	out := make([]objectionResult, len(pool))
	var wg sync.WaitGroup
	for i, name := range pool {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			out[i] = o.objectWith(ctx, snapDir, draft, name)
		}(i, name)
	}
	wg.Wait()
	total, failed := 0, 0
	for _, r := range out {
		if r.err != nil {
			failed++
			continue
		}
		total += len(r.objections)
	}
	o.endPhase("OBJECT  %d objection(s), %d objector(s) failed", total, failed)
	return out
}

// objectWith runs one agent's objection session.
func (o *Orchestrator) objectWith(ctx context.Context, snapDir, draft, agentName string) objectionResult {
	r := objectionResult{agent: agentName}
	d := prompt.ObjectData{
		Path:           snapDir,
		ModeGuidance:   prompt.DocumentGuidance,
		Draft:          prompt.Quote(draft),
		OutputContract: prompt.ObjectContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: o.cfg.Target.Mode, Path: snapDir, ModeGuidance: d.ModeGuidance,
	})
	tmpl := o.templates[o.cfg.Create.Object]
	if tmpl == nil {
		r.err = fmt.Errorf("object prompt %q was never loaded", o.cfg.Create.Object)
		return r
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		r.err = err
		return r
	}
	label := "object: " + agentName
	res := o.runAgentIn(ctx, snapDir, label, "object", agentName, o.cfg.Create.Object, 1, text)
	var out model.ObjectionOutput
	perr := res.Err
	if perr == nil {
		perr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	r.step = stepStat("object", agentName, o.cfg.Create.Object, len(text), res, perr != nil)
	o.logStep("object", agentName, o.cfg.Create.Object, 1, perr == nil, out,
		renderObjectionsMD(agentName, out.Objections, perr), res, "")
	if perr != nil {
		// Not fatal: objections are a net over the editor, and a net that failed is
		// reported rather than silently absent -- the caller records it.
		o.logf("WARNING: %s failed (%v)", label, perr)
		r.err = perr
		return r
	}
	for _, obj := range out.Objections {
		if !obj.Substantial() {
			o.logf("WARNING: %s raised an objection with no defect stated; ignored -- a passage alone is a pointer with no claim", label)
			continue
		}
		r.objections = append(r.objections, obj)
	}
	o.logf("%s done (%d objection(s), %s)", label, len(r.objections), res.Duration)
	return r
}

// revise runs the editor once more, over its own draft and the panel's
// objections. Failure is NOT fatal to the run: the caller ships the draft with
// the unapplied objections appended to the fixpoint-owned appendix and the
// provenance stamped unrevised -- objections a reader can see and weigh are
// worth more than a failed run, and objections silently discarded are the
// defect the phase exists to close.
func (o *Orchestrator) revise(ctx context.Context, snapDir, draft string, objections []model.Objection) (string, model.StepStat, error) {
	e := o.cfg.Roles.Editor
	o.phase("REVISE  %s addressing %d objection(s)", e.Agent, len(objections))
	d := prompt.EditorData{
		Path:           snapDir,
		ModeGuidance:   prompt.DocumentGuidance,
		Draft:          prompt.Quote(draft),
		Objections:     prompt.FormatObjections(objections),
		OutputContract: prompt.ReviseContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: o.cfg.Target.Mode, Path: snapDir, ModeGuidance: d.ModeGuidance,
	})
	doc, step, err := o.editorSession(ctx, snapDir, "revise", d)
	if err != nil {
		o.endPhase("REVISE  failed; the draft ships with the objections appended")
		return "", step, err
	}
	o.endPhase("REVISE  %d bytes", len(doc))
	return doc, step, nil
}

// editorSession is one editor invocation: render, run, extract the document,
// enforce the dissent section, persist the artifact. Shared by SYNTHESIZE and
// REVISE, which differ only in their data and what a failure means.
func (o *Orchestrator) editorSession(ctx context.Context, snapDir, phase string, d prompt.EditorData) (string, model.StepStat, error) {
	e := o.cfg.Roles.Editor
	tmpl := o.templates[e.Prompt]
	if tmpl == nil {
		return "", model.StepStat{}, fmt.Errorf("editor prompt %q was never loaded", e.Prompt)
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		return "", model.StepStat{}, err
	}
	label := phase + ": " + e.Agent
	res := o.runAgentIn(ctx, snapDir, label, phase, e.Agent, e.Prompt, 1, text)
	var doc string
	perr := res.Err
	if perr == nil {
		doc, perr = agent.ExtractText(res.Stdout, "design")
	}
	if perr == nil && !hasDissentSection(doc) {
		// A synthesis that erases disagreement is the failure the pipeline exists to
		// avoid, and the contract names the section verbatim. Checked as a contract
		// violation, not fixed up: fixpoint editing the editor's prose is what the
		// specification's own review ruled out.
		perr = errors.New("the document has no \"Decisions and dissent\" section; the contract requires dissent to be recorded, not erased")
	}
	step := stepStat(phase, e.Agent, e.Prompt, len(text), res, perr != nil)
	o.logStep(phase, e.Agent, e.Prompt, 1, perr == nil, nil, editorArtifact(phase, e.Agent, doc, perr), res, "")
	if perr != nil {
		o.logf("WARNING: %s failed (%v)", label, perr)
		return "", step, perr
	}
	o.logf("%s done (%d bytes, %s)", label, len(doc), res.Duration)
	return doc, step, nil
}

// hasDissentSection looks for the contract's required heading.
func hasDissentSection(doc string) bool {
	return strings.Contains(strings.ToLower(doc), "decisions and dissent")
}

// editorArtifact is the persisted record of one editor session.
func editorArtifact(phase, agentName, doc string, err error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s by %s\n\n", phase, agentName)
	if err != nil {
		fmt.Fprintf(&b, "**failed:** %v\n", err)
		return b.String()
	}
	b.WriteString(doc + "\n")
	return b.String()
}

// renderObjectionsMD is the persisted record of one objector's session.
func renderObjectionsMD(agentName string, objections []model.Objection, err error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# objections by %s\n\n", agentName)
	if err != nil {
		fmt.Fprintf(&b, "**failed:** %v\n", err)
		return b.String()
	}
	if len(objections) == 0 {
		b.WriteString("_None: the draft holds up._\n")
		return b.String()
	}
	for i, obj := range objections {
		fmt.Fprintf(&b, "## objection %d\n\n- passage: %s\n- defect: %s\n- consequence: %s\n\n", i+1, obj.Passage, obj.Defect, obj.Consequence)
	}
	return b.String()
}

// runCreate drives the whole create-design pipeline. It is dispatched from
// run() before any of the loop's machinery: a create run has no coder, no
// verify gate, no forge, and no git requirement -- its only write to the world
// is the deliverable, published atomically at the very end.
//
// Phase order and every failure rule are docs/design/create-design.md's; the
// comments below cite the decisions rather than restating the document.
func (o *Orchestrator) runCreate(ctx context.Context, sum *model.RunSummary) error {
	out, snap, capBytes, material, err := o.prepareCreate(ctx, sum)
	if err != nil {
		return err
	}

	rec := model.RoundRecord{Round: 1}
	defer func() {
		// Whatever happened, the steps are billed: a failed pipeline still spent
		// its sessions, and the scoreboard must say so.
		sum.Rounds = append(sum.Rounds, rec)
	}()

	proposals := o.proposeAll(ctx, snap.Dir, material, capBytes)
	alive := 0
	for _, p := range proposals {
		rec.Steps = append(rec.Steps, p.step)
		if p.err == nil {
			alive++
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if alive == 0 {
		return errors.New("create: every proposal failed; there is nothing to synthesize")
	}

	// Below two proposals there is nothing to compare: the critique phase is
	// skipped, and the deliverable's provenance -- not the editor -- says so.
	var critiques []critiqueResult
	carried := 0
	if alive >= 2 {
		critiques = o.critiqueAll(ctx, snap.Dir, proposals)
		for _, c := range critiques {
			rec.Steps = append(rec.Steps, c.step)
			if c.err == nil && len(c.critiques) > 0 {
				carried++
			}
		}
	} else {
		o.logf("create: only %d proposal survived, so there is nothing to critique; the deliverable is stamped single-model", alive)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	doc, step, err := o.synthesize(ctx, snap.Dir, material, proposals, critiques)
	rec.Steps = append(rec.Steps, step)
	if err != nil {
		return fmt.Errorf("create: the editor failed and nobody else may hold the pen: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	doc, unapplied, unrevised, err := o.objectAndRevise(ctx, snap.Dir, doc, &rec)
	if err != nil {
		return err
	}

	deliverable := create.Deliverable(doc, create.Provenance{
		RunID:        o.logs.RunID(),
		Editor:       o.cfg.Roles.Editor.Agent,
		Pool:         len(o.cfg.Roles.Review.Agents),
		Proposals:    alive,
		Critiques:    carried,
		SingleModel:  alive == 1,
		Uncritiqued:  alive >= 2 && carried == 0,
		Unrevised:    unrevised,
		SkippedLinks: snap.SkippedLinks,
	}, unapplied)
	// Through the same funnel as every text that leaves fixpoint: the document is
	// agent-authored, and a credential quoted into it must not reach disk unmasked.
	if err := create.Publish(out, agent.RedactSecrets(deliverable)); err != nil {
		return err
	}
	sum.Termination = model.TermCreated
	o.logf("deliverable: %s", out)
	return nil
}

// prepareCreate is everything before the first agent phase: resolve the
// deliverable path and refuse an existing one (a courtesy -- fail in seconds,
// not after eight sessions; the link(2) at publish is the guarantee), ping,
// snapshot, derive the caps, render the material.
func (o *Orchestrator) prepareCreate(ctx context.Context, sum *model.RunSummary) (out string, snap create.Snapshotted, capBytes int, material string, err error) {
	assignment, isDir, err := o.assignmentPath()
	if err != nil {
		return "", snap, 0, "", err
	}
	out = o.cfg.Create.Out
	if out == "" {
		out = create.DefaultOut(assignment, isDir)
	}
	if _, serr := os.Stat(out); serr == nil {
		return "", snap, 0, "", fmt.Errorf("%s already exists; a regenerated design must not silently replace a reviewed one -- name a different -out", out)
	}
	sum.Deliverable = out

	if o.cfg.Ping() {
		o.phase("PREFLIGHT  pinging %d agent(s)", len(o.activeAgentNames()))
		if err := o.Ping(ctx); err != nil {
			o.endPhase("PREFLIGHT  failed")
			return "", snap, 0, "", err
		}
		o.endPhase("PREFLIGHT  every agent responded")
	}

	// The snapshot is what every phase runs against; the caps bound what the
	// phases may say to each other. Both derive from the same budgets, and both
	// refuse at startup rather than degrade midway.
	budgets := o.createBudgets()
	snapBudget := int64(1) << 30
	if m := minPositive(budgets); m > 0 {
		snapBudget = int64(m)
	}
	scratch, err := o.logs.ScratchDir("assignment")
	if err != nil {
		return "", snap, 0, "", err
	}
	snap, err = create.Snapshot(assignment, scratch, []string{out}, snapBudget)
	if err != nil {
		return "", snap, 0, "", err
	}
	if snap.SkippedLinks > 0 {
		o.logf("WARNING: %d symlink(s) in the assignment were not followed; the deliverable's provenance says so", snap.SkippedLinks)
	}
	capBytes, err = create.Caps(budgets, snap.Bytes, len(o.cfg.Roles.Review.Agents))
	if err != nil {
		return "", snap, 0, "", err
	}
	o.logf("assignment: %d file(s), %d byte(s); per-proposal cap %s", snap.Files, snap.Bytes, capDesc(capBytes))

	material, err = o.snapshotMaterial(ctx, snap)
	if err != nil {
		return "", snap, 0, "", err
	}
	return out, snap, capBytes, material, nil
}

// objectAndRevise runs the bounded objection pass and the editor's revision.
// A failed revision does not fail the run: the draft ships with the objections
// appended by fixpoint and the provenance stamped unrevised -- objections a
// reader can see and weigh are worth more than a failed run.
func (o *Orchestrator) objectAndRevise(ctx context.Context, snapDir, doc string, rec *model.RoundRecord) (string, []model.Objection, bool, error) {
	if o.cfg.Create.Objections < 1 {
		return doc, nil, false, ctx.Err()
	}
	results := o.objectAll(ctx, snapDir, doc)
	var objections []model.Objection
	for _, r := range results {
		rec.Steps = append(rec.Steps, r.step)
		objections = append(objections, r.objections...)
	}
	if ctx.Err() != nil {
		return doc, nil, false, ctx.Err()
	}
	if len(objections) == 0 {
		return doc, nil, false, nil
	}
	revised, rstep, rerr := o.revise(ctx, snapDir, doc, objections)
	rec.Steps = append(rec.Steps, rstep)
	if ctx.Err() != nil {
		return doc, nil, false, ctx.Err()
	}
	if rerr != nil {
		// Deliberately NOT an error: the run continues, the draft ships with the
		// objections in fixpoint's appendix and the provenance stamped unrevised.
		// See the specification's REVISE failure rule.
		return doc, objections, true, nil //nolint:nilerr // shipping unrevised is the designed outcome
	}
	return revised, nil, false, nil
}

// assignmentPath resolves what the run designs against, and whether it is a
// directory.
func (o *Orchestrator) assignmentPath() (string, bool, error) {
	if o.cfg.Target.Document != "" {
		doc := o.cfg.Target.Document
		if !filepath.IsAbs(doc) {
			doc = filepath.Join(o.cfg.Target.Path, doc)
		}
		return doc, false, nil
	}
	info, err := os.Stat(o.cfg.Target.Path)
	if err != nil {
		return "", false, fmt.Errorf("assignment: %w", err)
	}
	return o.cfg.Target.Path, info.IsDir(), nil
}

// createBudgets collects the prompt budgets the caps derive from: the pool's and
// the editor's, since the SYNTHESIZE prompt is the largest of the run.
func (o *Orchestrator) createBudgets() []int {
	out := make([]int, 0, len(o.cfg.Roles.Review.Agents)+1)
	for _, name := range o.cfg.Roles.Review.Agents {
		out = append(out, o.cfg.Agents[name].PromptBudget)
	}
	out = append(out, o.cfg.Agents[o.cfg.Roles.Editor.Agent].PromptBudget)
	return out
}

// snapshotMaterial renders the assignment for the prompts, through the same
// collector every review uses -- a file becomes the material whole, a directory
// a listing -- so the fencing and truncation rules are shared, not reimplemented.
func (o *Orchestrator) snapshotMaterial(ctx context.Context, snap create.Snapshotted) (string, error) {
	c := target.New(config.Target{Mode: config.ModeDirectory, Path: snap.Dir, Document: snap.Entry})
	if err := c.Prepare(ctx); err != nil {
		return "", err
	}
	return c.Collect(ctx)
}

func minPositive(ns []int) int {
	out := 0
	for _, n := range ns {
		if n <= 0 {
			continue
		}
		if out == 0 || n < out {
			out = n
		}
	}
	return out
}

// capDesc renders a cap for the log, where 0 means unbounded.
func capDesc(capBytes int) string {
	if capBytes <= 0 {
		return "unbounded (no prompt budgets declared)"
	}
	return fmt.Sprintf("%d bytes", capBytes)
}
