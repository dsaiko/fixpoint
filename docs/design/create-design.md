# create-design: a panel drafts a design, an editor holds the pen

Status: revision 2. Revision 1 was reviewed by `review-design` itself (run
20260809-135648: 37 findings, 13 surviving highs); this revision answers them —
most materially, the pipeline was collecting blocking objections with no phase
able to apply them. This document remains the specification for increment 2 of
the design pipeline (`review-design` → **`create-design`** → `implement-design`).

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

Every phase reads the SNAPSHOT of the assignment, taken once into the run's
artifacts before anything runs — a file is copied verbatim; a directory is pinned
as an inventory (paths and content hashes) that every phase receives, with the
files themselves read from disk and a closing re-hash recording any mid-run drift
in the deliverable's provenance. Without this, phases of one run can read
different bytes of "the same" assignment, and nothing in the artifacts can prove
what was actually designed against.

### PROPOSE

Every agent in the reviewer pool receives the assignment — fenced and defanged
like all target-authored text — and produces a complete, independent proposal:
architecture, data, technology choices, failure handling, open questions. No
agent sees another's work; independence is where the panel's value is, per the
corroboration measurement above.

A proposal is prose in a `<design>` envelope, not findings JSON. It is bounded
(stated cap, elision announced — the same rule the conversation and intent
clamps follow), because the critique prompt must later carry all of them.

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

### REVISE

The consumer the objections were missing in the reviewed draft of this document:
one further editor session that must address each objection — amend the document,
or record it in the dissent section with its reason for standing firm. There is
no second objection pass on the revised text; an objection loop does not converge
for the same reason `max_final_passes` is 1, and the mandatory `review-design`
that follows is the check on the revision itself.

If REVISE fails, the run does not ship the draft as though nothing happened and
does not fail either: fixpoint appends the unapplied objections verbatim to the
dissent section — a mechanical append, no model holds the pen — and stamps the
provenance as unrevised. Objections a reader can see and weigh are worth more
than a failed run; objections silently discarded are the defect this phase
exists to close.

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
  objections: 1
```

The proposal/critique panel is the existing reviewer pool (`roles.review.agents`
from defaults), so who designs is the same one-edit decision as who reviews.
Editor defaults to claude for the same measured reason the judge does.

With today's two-agent pool a run is 8 sessions (2 propose + 2 critique + 1
synthesize + 2 object + 1 revise). Two proposals is thin; the pool returning to
four after the ollama quota reset doubles the value of the propose phase at the
same shape.

Prompt sizing is a function of POOL SIZE, and the largest prompt is not the
critique but the SYNTHESIZE — assignment + N proposals + N critiques — which is
also the only phase whose failure fails the run. The per-proposal and
per-critique caps are therefore derived, not fixed: cap = (agent prompt_budget −
assignment size − contract overhead) / (2 × pool), announced in the prompt and
elided with a stated count like every other clamp in this tool. A pool of four
halves the caps a pool of two enjoys; growing the panel must never be able to
push the editor over its budget.

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
  default. The write is temp-file-plus-rename, so a failure mid-write cannot
  leave a truncated file that the overwrite refusal would then protect forever.
  When the assignment is a directory, the `-out` path is excluded from its
  inventory — otherwise the second run of the same command reads its own
  deliverable as part of the assignment and designs against its previous answer.
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

## Open questions for the reviewing panel

1. Is one objection pass plus REVISE worth its cost (pool + 1 sessions), given
   the refutation round's measured record as a filter — or is the editor-error
   risk better covered by the mandatory `review-design` that follows?
2. The two-proposal minimum for critique: right threshold, or should a
   single-proposal run fail outright rather than degrade?
