# create-design: a panel drafts a design, an editor holds the pen

Status: revision 3, declared implementable. Revision 1 was reviewed by
`review-design` (run 20260809-135648: 37 findings) and revision 2 answered them;
the re-review (run 20260809-142133: 46 findings) then found holes revision 2 had
itself introduced, and this revision closes the material ones. Remaining findings
are accepted and recorded under *Inherited properties* — a specification does not
converge to zero findings any more than code does, and the stop rule is an
operator's judgment, not an empty report. This document is the specification for
increment 2 of the design pipeline (`review-design` → **`create-design`** →
`implement-design`).

## Goal

`fixpoint create-design -target assignment.md -out DESIGN.md` turns a short
statement of intent — *"a browser card game of Prší with a modern look"*, *"an
ESP32 temperature and CO2 sensor, including a printable case"* — into a design
document good enough to be reviewed by `review-design` and implemented by
`implement-design`.

The assignment is a local file or directory. The deliverable is one markdown
document written by fixpoint itself.

## What this is not

- **Not a consensus process.** Measured on this project: across 19 runs under 4%
  of findings were reported by two reviewers independently, and the refutation
  round has never once produced a unanimous position. Models do not converge;
  they diverge usefully. Any design that assumes the panel will "agree" is
  assuming a fiction, so this design does not.
- **Not implementation.** The output is a document. `implement-design`
  (increment 3) turns it into a project.
- **Not URL ingestion.** v1 reads local files. A later increment may accept a
  URL, fetched ONCE by fixpoint and snapshotted into the run's artifacts so every
  agent reads the same bytes and the run stays auditable — never fetched
  per-agent.

## Pipeline

```
assignment ─► snapshot ─► PROPOSE (each pool agent, independently)
           ─► CRITIQUE (each agent reads the OTHERS' proposals, anonymized)
           ─► SYNTHESIZE (the editor writes the final document + dissent)
           ─► OBJECT (one bounded pass: blocking objections only)
           ─► REVISE (the editor applies or records each objection)
           ─► fixpoint stamps provenance and writes -out
```

Every phase runs against the SNAPSHOT of the assignment: a verbatim copy — file
or directory — taken into the run's artifacts before anything runs, and used as
the agents' working directory. The copy is what freezes the bytes; an inventory
of hashes only detects drift after the fact, which the re-review of this document
correctly called not-a-snapshot. Copying also makes the exclusions structural
rather than rule-based: `-out`, `.fixpoint/`, and anything else that is not the
assignment simply is not in the copy, so a rerun cannot ingest its own previous
deliverable or the logs of the run that produced it. An assignment too large to
fit the prompt caps below is refused at startup, before any session — the copy
cost is bounded by the same number.

### PROPOSE

Every agent in the reviewer pool receives the assignment — fenced and defanged
like all target-authored text — and produces a complete, independent proposal:
architecture, data, technology choices, failure handling, open questions. No
agent sees another's work; independence is where the panel's value is, per the
corroboration measurement above.

A proposal is prose in a `<design>` envelope, not findings JSON. It is bounded
(stated cap, elision announced — the same rule the conversation and intent clamps
follow), because later phases must carry several of them at once — the SYNTHESIZE
prompt, the largest of the run, carries them all. The cap formula lives under
*Roles and configuration*.

### CRITIQUE

Every agent receives the **other agents' proposals, anonymized** (Proposal A, B,
C — authorship would hand a critic a reason that is not evidence, the same
argument the refutation round already documents) and returns a structured
critique per proposal: what is strong and should survive, what is weak or wrong
and why, and what it would adopt into a combined design. This is the refutation
round's shape pointed at proposals instead of findings.

A critic never receives its own proposal. Anonymization cannot blind an author to
its own text, so "critique everything" quietly mixed self-assessment in with the
judgments the phase exists to collect; excluding it is mechanical and costs
nothing. The label-to-agent mapping is fixpoint's, persisted in the run's
artifacts — anonymous in every prompt, attributable in the audit trail, exactly
like the anonymized refutation before it. The editor is blinded the same way
(resolving open question 1 of the reviewed draft): weighing a proposal by its
author's reputation is the bias the labels exist to remove, and the artifacts
keep the mapping for the human who wants it.

### SYNTHESIZE

A new role, the **editor** — read-only by validation, like the judge — receives
the assignment, every proposal, and every critique, and writes the final
document. The editor is the answer to "how do the models agree": they do not,
and someone must hold the pen.

The final document must contain a **Decisions and dissent** section: which
proposal each major decision came from, what was rejected and why, and where the
panel disagreed without resolution. Dissent is recorded, not erased — the same
principle as `contested` on findings.

### OBJECT

One bounded pass (`create.objections: 1`, `0` disables both this phase and
REVISE): the panel reads the final draft and may raise **blocking objections
only** — a defect in the chosen design, not a preference for the road not taken.
The phase has its own prompt (`create.object`) and a structured output contract —
objection id, the passage it names, the defect, the consequence — so the handoff
to REVISE is machine-checkable rather than prose the editor may miss. Objectors
are labeled the same way critics are: anonymous in the editor's prompt,
attributable in the artifacts.

### REVISE

The consumer the objections were missing in the reviewed draft of this document:
one further editor session that must address each objection — amend the document,
or record it in the dissent section with its reason for standing firm. There is
no second objection pass on the revised text; an objection loop does not converge
for the same reason `max_final_passes` is 1, and the mandatory `review-design`
that follows is the check on the revision itself.

If REVISE fails, the run does not ship the draft as though nothing happened and
does not fail either: fixpoint appends the unapplied objections verbatim to the
**fixpoint-owned appendix** below its provenance footer — never into the editor's
own text, whose internal structure is prose no machine reliably edits — and
stamps the provenance as unrevised. Objections a reader can see and weigh are
worth more than a failed run; objections silently discarded are the defect this
phase exists to close.

## Roles and configuration

```yaml
# create-design.yaml
roles:
  editor: { agent: claude, prompt: design-editor }   # read-only, validated
  review:
    strategy: all
    prompts: []          # no review lenses; the pool is used for propose/critique

create:
  propose: design-propose
  critique: design-critique
  object: design-object
  objections: 1
```

The proposal/critique panel is the existing reviewer pool (`roles.review.agents`
from defaults), so who designs is the same one-edit decision as who reviews.
Editor defaults to claude for the same measured reason the judge does.

With today's two-agent pool a run is 8 sessions (2 propose + 2 critique + 1
synthesize + 2 object + 1 revise). Two proposals is thin; the pool returning to
four after the ollama quota reset doubles the value of the propose phase at the
same shape.

Prompt sizing is a function of POOL SIZE, and the largest prompt is the
SYNTHESIZE — assignment + N proposals + N critiques — which is also the only
phase whose failure fails the run; the REVISE prompt (draft + objections) is
sized under the same rule. The per-proposal and per-critique caps are derived,
not fixed:

    cap = (B − assignment − overhead) / (2 × pool)

where **B is the smallest `prompt_budget` across the pool and the editor** — the
pool is deliberately heterogeneous (900 kB in-house, 400 kB served), and a cap
derived from anything but the minimum overruns exactly the agent least able to
take it. The formula is validated at startup: if the cap falls below a stated
floor (an assignment so large that proposals would be squeezed into uselessness),
the run is refused before any session rather than degraded into one. Announced in
the prompt and elided with a stated count, like every other clamp in this tool. A
pool of four halves the caps a pool of two enjoys; growing the panel must never
be able to push the editor over its budget.

## Output and trust

- **Agents never write files.** Every phase returns text; fixpoint extracts,
  redacts, and writes — the same division `review-body.md` already uses. So
  `create-design` needs **no trust flag** beyond what any run needs
  (`-trusted-bundle` when the target ships a bundle): nothing editable is
  invoked.
- `-out` names the deliverable (default: `DESIGN.md` beside the assignment). It
  must not live under `.fixpoint/` — that is the gitignored log directory, and a
  deliverable is not a log. fixpoint **refuses to overwrite** an existing file,
  checked at startup (before anything is spent) and again at the write;
  regenerating over a reviewed design must be a deliberate `-out` choice, not a
  default. The write is temp-file-then-**hard-link**: the deliverable is written
  beside its destination and linked into place, and `link(2)` fails if the target
  exists — which makes the no-overwrite refusal and the publication one atomic
  operation instead of a check racing a write. (Plain rename would silently
  replace the very file the refusal exists to protect, which the re-review of
  this document caught.) A failed write leaves only a temp file, cleaned up by
  the run or overwritten by the next; it can never leave a truncated deliverable
  squatting on the protected name. Exclusion of `-out` from a directory
  assignment falls out of the snapshot: the copy simply does not contain it.
- **Provenance is stamped by fixpoint, not written by the editor.** The
  deliverable opens with a header fixpoint composes from its own facts — run id,
  pool, which phases degraded (single-model, uncritiqued, unrevised) — followed
  by the editor's document. A degradation notice the degraded model writes about
  itself is not a notice; this is the same rule that keeps the review signature
  outside every region carrying agent text.
- The assignment is untrusted input. It can steer what the design SAYS — that is
  its purpose — but through the same fencing as every diff, it cannot address
  the agents directly, and no phase executes anything the assignment names.
  `verify` does not run at all in create-design.

## Artifacts and failure

Every phase logs per-step artifacts (`propose-<agent>-*`, `critique-<agent>-*`,
`synthesize-*`, `object-*`) and journal events, exactly like review steps, so a
run is auditable and its cost attributable per phase.

- A failed proposer drops out; the run continues with the rest. Below **two**
  surviving proposals the critique phase is skipped (there is nothing to compare)
  and the editor works from what exists — the provenance header then states it is
  a single-model design, because a reader must be able to tell a synthesis from
  one model's opinion.
- A failed critic drops out the same way; its missing critiques simply do not
  reach the editor. Zero surviving critics degrades the run to an uncritiqued
  synthesis, stamped as such in the provenance — the same rule as single-model,
  one step earlier. No critic failure fails the run: critiques are judgment about
  proposals, and the mandatory review-design that follows re-covers that ground.
- Zero proposals fails the run. A failed **editor** fails the run: the phases
  before it are preserved as artifacts, but nobody else may hold the pen —
  falling back to "use the best proposal verbatim" would publish an unreviewed
  single voice under a synthesis's name.
- A failed objection pass does not fail the run; the deliverable's dissent
  section records that the pass did not complete. Objections are a safety net
  over the editor, and a net that failed is reported, not silently absent.
- Prompt budgets apply as everywhere; the critique prompt is the largest
  (assignment + N proposals) and the proposal cap above is what keeps it inside
  the measured budgets.

## Exit and workflow

Exit 0 with the deliverable written; nonzero otherwise. No verdict — a created
design has not been reviewed yet, and pretending otherwise would launder the
editor's own output into an approval. The intended workflow is explicit:

```sh
fixpoint create-design -target assignment.md -out DESIGN.md
fixpoint review-design -target DESIGN.md          # the panel judges it
# fix findings by editing the document (or a later fix-design), re-review
fixpoint implement-design -target DESIGN.md       # increment 3
```

## Inherited properties worth stating

Agents run in the directory holding the assignment and may read any file it
references — raw, outside the prompt's fencing. That is the posture of every
fixpoint run (a reviewer reading the repository reads unfenced bytes by design),
not a new surface this feature opens; it is stated here because a design
assignment is more likely than code to say "see this other document".

The proposal/critique pool reuses `roles.review.agents` deliberately: who designs
should be the same one-edit decision as who reviews. If create-design ever needs
a different panel, an explicit `create.agents` override is the extension point —
a second pool by default would drift exactly the way the per-config panels did
before the pool was centralized.

Findings from this document's own reviews that are ACCEPTED as tool-wide
properties rather than answered here:

- **Timeouts and liveness** are the agent layer's: every session runs under its
  agent's `timeout` with the heartbeat the whole tool uses. "Failed" means what
  it means everywhere else — a non-zero exit, an unparseable reply, or that
  timeout.
- **Concurrent runs** are excluded by the existing per-project run lock; a
  second create-design on the same project does not start, so the publication
  step never races another fixpoint.
- **No resume.** A run that dies in SYNTHESIZE spends its PROPOSE and CRITIQUE
  sessions; their artifacts survive for a human but no machinery replays them.
  True of every fixpoint run and accepted here for the same reason: resume is a
  tool-wide feature with tool-wide complexity, not something one config should
  grow privately.
- **Fan-out has no admission control** beyond pool size — the same property as
  the review panel's fan-out, bounded by the same number.

## Open questions for the reviewing panel

1. Is one objection pass plus REVISE worth its cost (pool + 1 sessions), given
   the refutation round's measured record as a filter — or is the editor-error
   risk better covered by the mandatory `review-design` that follows?
2. The two-proposal minimum for critique: right threshold, or should a
   single-proposal run fail outright rather than degrade?
