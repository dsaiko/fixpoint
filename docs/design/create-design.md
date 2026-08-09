# create-design: a panel drafts a design, an editor holds the pen

Status: draft for review. This document is the input to `review-design` and, once
it survives that, the specification for increment 2 of the design pipeline
(`review-design` → **`create-design`** → `implement-design`).

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
assignment ─► PROPOSE (each pool agent, independently)
           ─► CRITIQUE (each agent reads all proposals, anonymized)
           ─► SYNTHESIZE (the editor writes the final document + dissent)
           ─► OBJECT (one bounded pass: blocking objections only)
           ─► fixpoint writes -out
```

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

Every agent receives **all proposals, anonymized** (Proposal A, B, C — authorship
would hand a critic a reason that is not evidence, the same argument the
refutation round already documents) and returns a structured critique per
proposal: what is strong and should survive, what is weak or wrong and why, and
what it would adopt into a combined design. This is the refutation round's shape
pointed at proposals instead of findings.

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

One bounded pass (`create.objections: 1`, `0` disables): the panel reads the
final draft and may raise **blocking objections only** — a defect in the chosen
design, not a preference for the road not taken. The editor must address each:
amend the document, or record the objection in the dissent section with its
reason for standing firm. There is no second pass; an objection loop does not
converge for the same reason `max_final_passes` is 1.

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

With today's two-agent pool a run is 7 sessions (2 propose + 2 critique + 1
synthesize + 2 object). Two proposals is thin; the pool returning to four after
the ollama quota reset doubles the value of the propose phase at the same shape.

## Output and trust

- **Agents never write files.** Every phase returns text; fixpoint extracts,
  redacts, and writes — the same division `review-body.md` already uses. So
  `create-design` needs **no trust flag** beyond what any run needs
  (`-trusted-bundle` when the target ships a bundle): nothing editable is
  invoked.
- `-out` names the deliverable (default: `DESIGN.md` beside the assignment). It
  must not live under `.fixpoint/` — that is the gitignored log directory, and a
  deliverable is not a log. fixpoint **refuses to overwrite** an existing file;
  regenerating over a reviewed design must be a deliberate `-out` choice, not a
  default.
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
  and the editor works from what exists — the deliverable's header then states it
  is a single-model design, because a reader must be able to tell a synthesis
  from one model's opinion.
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

## Open questions for the reviewing panel

1. Anonymizing proposals in CRITIQUE hides authorship from critics — but the
   editor sees critiques referencing "Proposal B". Should the editor also be
   blinded to which agent wrote what, or does it need authorship to weigh a
   proposer's known strengths?
2. Is one objection pass worth its cost (one session per pool agent), given the
   refutation round's measured record as a filter — or is the editor-error risk
   better covered by the mandatory `review-design` that follows?
3. The two-proposal minimum for critique: right threshold, or should a
   single-proposal run fail outright rather than degrade?
