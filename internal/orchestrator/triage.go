package orchestrator

import (
	"context"
	"strings"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/review"
)

// triageConversations decides what the pull request's open comments commission,
// and makes sure every one of them ends with an answer.
//
// Without this step a comment is context and nothing else: the coder is shown the
// threads and told to leave alone anything its own issue does not address, which
// is deliberate -- a comment is text anyone with access to the pull request can
// write, and "fix this" from a stranger must not reach the tree on its own say-so.
// The cost of that rule is that a reviewer's comment can be read by three coder
// sessions and answered by none.
//
// So the decision is made ONCE, by a read-only agent, before any fixing starts:
//
//   - accept turns the comment into an ordinary issue. It goes through the same
//     pipeline as anything the panel found -- one coder session, the project's own
//     verify gate, its own commit -- and the conversation is answered only once
//     that commit lands. Nothing here shortens the path from untrusted text to a
//     change; it only lets the text enter at the front of it.
//   - reject is answered immediately, with the reason triage gave. A rejection
//     claims no work was done, so unlike an accepted one it needs no commit behind
//     it.
//
// Deciding once is also what stops the same conversation being answered by three
// separate sessions, which is the failure the run's own panel reported: a thread
// stayed in the list forever and every session was free to reply to it again.
//
// Failing here is not fatal. A run whose triage died reports it and carries on with
// the conversations as plain context -- the behavior before this step existed.
func (o *Orchestrator) triageConversations(ctx context.Context) {
	t := o.cfg.Roles.Triage
	if t.Agent == "" || len(o.threads) == 0 || ctx.Err() != nil {
		return
	}
	o.phase("TRIAGE  %s deciding %d open conversation(s)", t.Agent, len(o.threads))

	// The change under review, collected here rather than read off o.material: that
	// field is assigned by the review-only path, which never reaches triage (a
	// review-only run has no coder, so readForgeThreads leaves the list empty), so
	// the prompt used to advertise a material block and render it empty. Collection
	// is read-only and the tree is pristine at this point, so this is the same diff
	// round 1 will put in front of the panel.
	material, err := o.collector.Collect(ctx)
	if err != nil {
		// Not fatal here: the round loop collects again and fails the run properly if
		// this is a real collection failure. Deciding what a comment commissions
		// without the diff in front of the agent is the thing worth refusing.
		o.logf("WARNING: triage: could not collect the change under review (%v); the conversations stay context and are not answered", err)
		o.endPhase("TRIAGE  did not finish; the conversations stay context and are not answered")
		return
	}

	me := o.forgeLogin(ctx)
	decisions, ok := o.askTriage(ctx, t, material)
	if !ok {
		o.endPhase("TRIAGE  did not finish; the conversations stay context and are not answered")
		return
	}

	byThread := make(map[string]model.TriageDecision, len(decisions))
	known := make(map[string]forge.Thread, len(o.threads))
	for _, th := range o.threads {
		known[th.ID] = th
	}
	for _, d := range decisions {
		missing := missingAcceptFields(d)
		switch {
		case !model.ValidTriageVerdict(d.Verdict):
			o.logf("WARNING: triage returned an unknown verdict %q on conversation %s; it is left undecided", d.Verdict, d.Thread)
		case known[d.Thread].ID == "":
			// An id nobody was shown. It cannot answer a conversation that exists, but a
			// triage agent inventing them is worth saying out loud.
			o.logf("WARNING: triage decided conversation %q, which was not in the set it was shown; ignored", d.Thread)
		case strings.TrimSpace(d.Reason) == "":
			// The reason IS the reply on a rejection and the justification on an
			// acceptance. A decision nobody can argue with is not a decision, and on the
			// reject side it would post an empty answer to a person.
			o.logf("WARNING: triage decided %s with no reason; it is left undecided", d.Thread)
		case model.NormalizeTriageVerdict(d.Verdict) == model.TriageAccept && missing != "":
			// An acceptance is a work order, and these fields are the whole of it. The
			// comment itself is context the coder is told not to act on by its own say-so,
			// so an accept with no title or no description commissions a session with
			// nothing to do -- and spends a coder pass, a verify gate and a commit
			// subject on it. Left undecided instead: the conversation stays context,
			// exactly as it would have without this step.
			o.logf("WARNING: triage accepted conversation %s with no %s; it is left undecided", d.Thread, missing)
		case byThread[d.Thread].Thread != "":
			o.logf("WARNING: triage decided conversation %s more than once; the first decision stands", d.Thread)
		default:
			// Store the spelling the validator checked, not the one the agent typed: the
			// dispatch below compares against the constants, so a padded "accept\n" would
			// validate as an acceptance and be dispatched as a rejection.
			d.Verdict = model.NormalizeTriageVerdict(d.Verdict)
			byThread[d.Thread] = d
		}
	}

	var accepted, rejected, undecided int
	remaining := make([]forge.Thread, 0, len(o.threads))
	for _, th := range o.threads {
		d, ok := byThread[th.ID]
		if !ok {
			// Silence is not a decision. The thread stays in the list as context, exactly
			// as it would have without this step, and the count is reported: a run that
			// says it answers every conversation must say when it did not.
			undecided++
			remaining = append(remaining, th)
			continue
		}
		if d.Verdict == model.TriageAccept {
			accepted++
			o.commissioned = append(o.commissioned, o.commissionedFinding(d, th, me))
			// Kept: the coder that fixes the issue answers this thread once its work is
			// committed, and postReplies refuses an id that is not in this list.
			// Recorded as commissioned, too, so that it is that ONE session's to answer:
			// every other session is shown the thread as context and would otherwise be
			// free to reply to it as well.
			if o.commissionedThreads == nil {
				o.commissionedThreads = make(map[string]bool)
			}
			o.commissionedThreads[th.ID] = true
			remaining = append(remaining, th)
			continue
		}
		rejected++
		o.declineConversation(ctx, th, d.Reason)
	}
	o.threads = remaining
	o.endPhase("TRIAGE  %d accepted, %d declined, %d left undecided", accepted, rejected, undecided)
}

// missingAcceptFields names the required fields an acceptance left empty, or ""
// when it carries them all.
//
// Only the two the coder actually works from. Severity and category are left out on
// purpose: both have a documented default and a warning below, because a
// mislabeled issue is still workable, while a nameless one is not -- an empty title
// reaches the fingerprint that groups findings and the subject of the commit, and
// an empty description is an instruction to fix nothing in particular. This mirrors
// validateReviewFindings, which refuses a titleless finding from the panel; work
// commissioned by a comment enters the same pipeline and is held to the same bar.
func missingAcceptFields(d model.TriageDecision) string {
	var missing []string
	if strings.TrimSpace(d.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(d.Description) == "" {
		missing = append(missing, "description")
	}
	return strings.Join(missing, " or ")
}

// askTriage runs the agent over the collected material and returns its decisions.
func (o *Orchestrator) askTriage(ctx context.Context, t config.RoleRef, material string) ([]model.TriageDecision, bool) {
	d := prompt.TriageData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		ModeGuidance:   prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:         material,
		Conversations:  o.conversations(),
		OutputContract: prompt.TriageContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: d.Mode, Path: d.Path, ModeGuidance: d.ModeGuidance, Target: d.Target,
	})
	tmpl, ok := o.templates[t.Prompt]
	if !ok || tmpl == nil {
		// Defensive: New loads this template, so reaching here means a hand-built
		// Orchestrator. Rendering a nil template panics, which would take a run down
		// over a step that is allowed to fail.
		o.logf("WARNING: triage prompt %q was never loaded; the conversations stay context", t.Prompt)
		return nil, false
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		o.logf("WARNING: triage: render failed (%v); the conversations stay context", err)
		return nil, false
	}
	res := o.runAgent(ctx, "triage: "+t.Agent, "triage", t.Agent, t.Prompt, 0, text)
	var out model.TriageOutput
	parseErr := res.Err
	if parseErr == nil {
		parseErr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	o.triageStep = stepStat("triage", t.Agent, t.Prompt, len(text), res, parseErr != nil)
	// Persisted like every other agent pass. The decisions ARE the product here --
	// a rejection is posted to a person and an acceptance becomes a commit, so
	// "what was decided, and on what grounds" has to survive the run.
	o.logStep("triage", t.Agent, t.Prompt, 0, parseErr == nil, out,
		logstore.RenderTriageMD(t.Agent, out.Decisions, parseErr), res, "")
	if parseErr != nil {
		o.logf("WARNING: triage failed (%v); the conversations stay context and are not answered", parseErr)
		return nil, false
	}
	return out.Decisions, true
}

// commissionedFinding turns an accepted decision into an ordinary finding.
//
// Triage's OWN words, not the comment's: the coder acts on this text, and "this
// looks wrong to me" is not something anyone can fix. What the comment said is
// still in front of the coder -- the thread stays in the conversation list -- but
// what it is asked to do comes from an agent that went and read the code.
//
// The severity is validated rather than trusted. It orders the round and, in a
// review run, decides what blocks a merge; a value invented by an agent reading
// attacker-controlled text must not be able to jump that queue.
func (o *Orchestrator) commissionedFinding(d model.TriageDecision, th forge.Thread, me string) model.Finding {
	sev := model.NormalizeSeverity(d.Severity)
	if !model.ValidSeverity(sev) {
		o.logf("WARNING: triage gave conversation %s an unknown severity %q; recorded as medium", th.ID, d.Severity)
		sev = "medium"
	}
	cat := strings.TrimSpace(d.Category)
	if cat == "" {
		cat = "bug"
	}
	file := strings.TrimSpace(d.File)
	if file == "" {
		// The conversation names the file even when the decision forgot to -- a review
		// comment is anchored to a path. Taking it from there keeps the issue locatable
		// for the coder and for the history, rather than refusing an acceptance whose
		// location was in front of the agent all along.
		file = th.Path
	}
	external := me == "" || !strings.EqualFold(me, th.Author)
	return model.Finding{
		Agent:       o.cfg.Roles.Triage.Agent,
		Lens:        o.cfg.Roles.Triage.Prompt,
		Category:    cat,
		Severity:    sev,
		File:        file,
		Line:        d.Line,
		Title:       strings.TrimSpace(d.Title),
		Description: strings.TrimSpace(d.Description),
		Origin:      model.Origin{Thread: th.ID, Author: th.Author, External: external},
	}
}

// declineConversation answers a comment triage did not accept.
//
// Posted immediately, unlike an accepted one: a rejection asserts that nothing was
// changed, so there is no commit for it to wait behind -- the rule that holds
// replies until their fix lands exists to stop a claim about work that does not
// exist. Signed like every other machine-authored reply.
func (o *Orchestrator) declineConversation(ctx context.Context, th forge.Thread, reason string) {
	o.logf("triage: conversation %s declined -- %s", th.ID, firstLineOf(reason))
	if !o.cfg.Review.Post {
		o.logf("conversation %s was not answered (-post was not given)", th.ID)
		return
	}
	r := readerFor(ctx, o.cfg.Target.Path)
	if r == nil {
		return
	}
	body := publishedText(forge.SanitizeText(reason) + "\n\n" + o.replySignature())
	if err := r.Reply(ctx, o.cfg.Target.Path, o.cfg.Target.PR, th.ID, body); err != nil {
		o.logf("ERROR: answering conversation %s failed: %v", th.ID, err)
		return
	}
	o.logf("answered conversation %s", th.ID)
}

// replySignature is the attribution on any machine-authored reply, plus the
// invisible marker that lets a later run recognize it.
//
// Built here rather than at each call site so a decline and a fix report are
// signed identically -- both are this tool answering a person -- and so neither
// can be posted without the marker. A reply that lost it would be indistinguishable
// from a person's, and the next run would read it as a live question and answer it
// again.
func (o *Orchestrator) replySignature() string {
	sig := review.ReplySignature(o.cfg.Review.ReplySignature, review.SignatureFacts{
		Agents:  []string{o.cfg.Roles.Coder.Agent},
		Run:     o.logs.RunID(),
		Version: review.Version(),
		Config:  configBaseName(o.source.Config),
	})
	return sig + "\n" + forge.ReplyMarker(o.logs.RunID())
}

// forgeLogin is the account the forge CLI is authenticated as, or "".
//
// Used only to decide whether a comment's author is somebody else, which is a
// LABEL and never a permission: an external request is acted on exactly like any
// other. "" makes every author external, which is the direction that says more.
func (o *Orchestrator) forgeLogin(ctx context.Context) string {
	r := readerFor(ctx, o.cfg.Target.Path)
	if r == nil {
		return ""
	}
	return r.Login(ctx, o.cfg.Target.Path)
}
