> Drafted by an AI panel and synthesized by a single editor · run 20260809-175113
> 2 proposal(s) from a pool of 2, 2 critique set(s) · this header is stamped by the tool, not written by the editor
> Revision 2 · the operator folded in the 18 high findings of review-design run 20260812-183115 (the first review with repository access) · what changed and why is recorded at the end of §11
> Revision 3 · the operator folded the mechanism-level findings of review-design run 20260813-003817 (the first four-reviewer round) and recorded the -continue trust cluster as open questions · §11 and §12

# implement-design: plan first, one task per commit, two targets

## 0. Reading note on provenance of this document

This synthesis was written without repository access — the review sandbox exposed
only the assignment file, and the same was true for this revision round (the
working directory contains `implement-design-assignment.md` and nothing else).
Where the design names an existing symbol (`verify.Run`, `target.Collector`,
`create.Snapshot`, `agent.ExtractJSON`, `create.Publish`, `RenderRunTable`,
`flattenField`, `RedactSecrets`, `config.mandatoryExcludes`, `relWithin`,
`discardFailedFix`, `validPathIdent`, `guardUntrustedGitConfig`), it is repeating a
claim made by a proposal, not a verified fact. Each such name is a **reuse
hypothesis the implementer must confirm first**; if a named helper does not exist or
does not have the property claimed, the surrounding decision still stands but its
cost changes. Section 11 lists the places where a failed hypothesis would change a
decision rather than just its price. The same caveat applies to the exit-code
allocation in §8.

Three hypotheses added in this revision are load-bearing enough to name here:

- **`Collector.StashDirty` may not include untracked files.** The whole discard
  path in §5.3 depends on an untracked-inclusive discard. If the existing helper is
  tracked-only, extending it (or adding a sibling) is **in scope for this
  increment**, not assumed. §5.3's post-discard clean-tree assertion exists so a
  wrong guess here fails loudly on the first retry instead of silently mixing two
  sessions' output.
- **git's repo-local metadata guards.** §5.2's repository-invariant check needs a
  cheap way to enumerate refs and hash `.git/config`; plain `git for-each-ref` and a
  file digest suffice, but if `guardUntrustedGitConfig` already encodes the list of
  dangerous config keys, reuse it rather than re-deriving it.
- **`verify.Run`'s working-directory and timeout surface.** §4.2's feasibility rule
  needs the configured session and gate timeouts as numbers at plan time. If they
  are not reachable as values, the rule degrades to a stated skip (§4.2 rule 7),
  not to a silent one.

## 1. The contract, stated before anything else

> `implement-design` takes a design document and produces a new git repository in
> which **every commit after the bootstrap commit is one plan task, authored by one
> coder session, that passed the operator's configured gate** — or, where no task
> commit exists, the run's report names the task and why it does not.

It makes no claim that the result is correct, complete against the design, secure,
or good. Nothing reviews code inside this run. Quality is the job of the
`review-code` / `fix-code` loops that follow. §9 draws that line precisely,
including the part of it this pipeline genuinely cannot cover.

## 2. The one structural insight: two targets, not one

Every fixpoint run so far has had one tree. `target.path` is simultaneously the
thing read, the thing committed to, the thing the lock is claimed on, the thing the
clean-tree check protects, the thing `.fixpoint/` is excluded from, and the thing
`-trusted-target` vouches for. All six meanings coincide, so the code never had to
separate them.

`implement-design` separates them, and most of this design falls out of that:

| | **read-target** | **write-target** |
|---|---|---|
| what | the design document (markdown file) | a new project directory |
| trust | untrusted content; the operator asserts it is safe to act on | *created* by fixpoint, and re-checked after every coder session — see below |
| git | may live anywhere; may not be a repo at all | a fresh repo fixpoint initialises |
| written to | never — **except the run's artifact root `.fixpoint/`**, which lives here and keeps its hardening (below) | the only tree that is committed to |
| clean-tree check | irrelevant | guaranteed at start (it did not exist) and **re-asserted before every task** |
| `.fixpoint/` logs | here (existing behaviour, unchanged) | never — its history is only the implementation |
| run lock | not claimed | claimed |

Naming the split is what lets the *first-contact* hardening — `ensureCleanTree` over
an operator's dirty tree and the "this repository arrived carrying a hostile
`.git/config`, a bundle, or hooks" guard — drop out of this pipeline: there is no
pre-existing tree to harden, because fixpoint made it. **Scoped to the committed
tree only** (review run 20260813-003817): the artifact root `.fixpoint/` did NOT
move — it lives beside the design, in a pre-existing directory the operator named —
so `checkLogsNotSymlinked` and the git-exclude dance stay, on whichever tree
receives the artifacts, on every path including a fresh run. Newness excuses the
tree fixpoint made; it excuses nothing about where the logs land.

**What does not drop out is the same hardening applied *between* sessions.** The
write-target is trusted at the moment it is created and not one minute longer: the
coder that runs in it is a full agent with permission checks disabled and can write
`.git/config`, drop a file in `.git/hooks/`, stage a poisoned index, create a nested
repository, or move a ref that is not HEAD — none of which the HEAD comparison in an
earlier draft of this design would have caught. So the guard is not deleted, it is
**relocated**: §5.1 hardens the repository at `git init`, §5.2 step 4 re-verifies the
repository's own metadata after every coder session and stops the run on any
mismatch, and §5.2 step 8 rebuilds the index from fixpoint's own census rather than
trusting whatever the session left there. "Trusted by construction" is true of the
tree fixpoint creates; it is a claim about an instant, and this design says so
rather than leaning on it for the length of a forty-session run.

The read/write split is Proposal A's central move and it is correct.

## 3. Shape

```
fixpoint implement-go -target DESIGN.md -out ~/src/prsi -trusted-target

  PREFLIGHT  config, trust, gate policy, -out absent and not nested in any repo,
     │       design readable, within the planner's budget, and carrying an
     │       outline the coverage rule can read                          (cmd)
     │
  SNAPSHOT   the design is copied into the run's artifacts;
     │       every later phase reads the copy                            (internal/create.Snapshot)
     │
  PLAN       one read-only planner session over the snapshot →
     │       JSON plan → deterministic validation (including fit against
     │       the run deadline), or refusal                               (internal/implement/plan.go)
     │       ── with -plan-only, the run stops here, having created nothing
     │
  SCAFFOLD   mkdir -out, git init (hardened), write DESIGN.md (byte-exact),
     │       PLAN.md, PLAN.json, .gitignore → the BOOTSTRAP COMMIT       (internal/implement/scaffold.go)
     │
  BUILD      for each task, in plan order, serially:
     │         clean-tree precondition → coder session → repository invariant
     │         → tree reconciliation → gate → commit
     │         a failed task is discarded; its dependents are skipped
     │         every processed task leaves exactly one commit: code, or an
     │         empty outcome marker (§5.4) — skipped tasks are derived, not recorded
     │
  REPORT     scoreboard, run summary, journal; exit 0 or 2               (orchestrator)
```

With `-continue <project>` (§5.5) the run skips SCAFFOLD, reads the plan from the
project's own bootstrap commit, replays every recorded outcome — built or not —
from the commit trailers, and enters BUILD at the first task with no commit of
either kind.

**Planning happens before the write-target exists.** Proposal B's ordering, and it
is right for two reasons: a planning failure leaves no orphan directory to explain,
and it makes `-plan-only` coherent (§7.4) — a defect Critic 2 correctly found in
Proposal A, which scaffolded first and then could never resume into the directory
it had already claimed. `-out`'s *absence* is still checked at preflight so a
doomed run does not spend a planner session; the `os.Mkdir` after planning is the
authoritative claim.

### Parts, drawn where ownership changes

- **`internal/implement/plan.go`** — the plan as a value: schema, parser,
  deterministic validator, markdown renderer, and the coverage/feasibility rules.
  No agent scheduling, no git. This is the part with the interesting unit tests, and
  it is the part a fixer can test without an agent.
- **`internal/implement/scaffold.go`** — the only code in the tool that creates a
  repository. `os.Mkdir` (fails if the path exists — check and claim in one
  syscall, the argument `create.Publish` already makes for `link(2)`),
  `git init` with the hardening flags of §5.1, the four files, one commit.
- **`internal/implement/repostate.go`** — the repository-invariant snapshot and
  comparison (§5.2 step 4) and the census (§5.2 steps 0, 6, 7). Deterministic, git
  plumbing only, no agents: the second part a fixer can test without a model.
- **`internal/orchestrator/implementrun.go`** — the phases, dispatched from `run()`
  before the loop's machinery, exactly as the create run is dispatched. Owns the
  planner session, the per-task coder sessions, the gate call, the commit, and skip
  propagation.
- **`internal/verify`** — reused verbatim. The gate is the only signal in this tool
  no model authored; it needs no change. The *policy* around it changes (§7.2).
- **`internal/target.Collector`** — reused, pointed at the write-target. `Commit`,
  `GitClean`, `HeadSHA`, `StashDirty`, `ChangedSince` already exist and are already
  hardened against trailer forging, closing-keyword injection and secret leakage.
  Three additions: `Init(ctx)`, an untracked-inclusive discard (§0), and a
  staged/empty commit primitive — `Commit` today makes no commit on a clean tree
  and stages its own idea of the changes, while §5.2 step 8 needs to stage a
  computed path set and §5.4's outcome markers need `--allow-empty`; the new
  primitive (say `CommitStaged(msg, paths, allowEmpty)`) must preserve the
  existing trailer, redaction and hardening behaviour, and it is **in scope for
  this increment**, not assumed (review run 20260813-003817).
- **config + prompts** — one new read-only role, one new config section, four
  shipped configs, two prompts.

### Boundaries deliberately not drawn

- **No reviewer panel during implementation.** The assignment's rule, and the right
  one: a panel over code that is still being written produces critiques of
  intermediate states — the measured lesson behind `max_final_passes`.
- **No panel over the plan either.** A plan is a list with checkable structure. The
  deterministic validator (§4.2) catches what matters; one operator glance at
  `-plan-only` output catches the rest for free. A plan needs one author.
- **No parallel task sessions.** Parallel sessions share a base, so a later patch is
  written against a tree without the earlier fix in it — measured, and worse on a
  fresh project, where task N+1 would be written against a tree that does not
  contain task N's *files*. Serial, permanently.

## 4. Data

### 4.1 The plan

The planner returns JSON inside a `<plan>` envelope, extracted with the same
mechanism the review, critique and objection contracts already use. Not YAML: the
producer is a model, and one proven envelope with one proven reformat-retry path
beats a second syntax.

```json
{
  "schema_version": 1,
  "project":  { "name": "prsi", "summary": "a browser card game of Prší" },
  "coverage": [
    { "heading": "## Rendering",  "tasks": ["T03", "T04"] },
    { "heading": "## Networking", "tasks": [], "out_of_scope": "the design defers multiplayer to a later phase" }
  ],
  "tasks": [
    { "id": "T01",
      "title": "Project skeleton: index.html, main.js, and a page that loads",
      "goal": "prose: what must be true when this task is done",
      "acceptance": ["opening index.html shows an empty table", "no console errors"],
      "files": ["index.html", "src/main.js"],
      "depends_on": [],
      "design_refs": ["## Shape", "## Rendering"] }
  ]
}
```

**Ownership.** The planner owns `project`, `coverage`, and each task's `title`,
`goal`, `acceptance`, `files`, `depends_on`, `design_refs`, and the array order.
fixpoint owns everything else and never asks the planner for it: outcome, commit
SHA, attempt count, gate results, durations, tokens, and all provenance. Provenance
is injected **after** parsing, so the model cannot author or overwrite it
(Proposal B; the correct ownership split for the assignment's provenance rule).

That injection has a name and a shape, because §7.4's `-plan` flag has to check it:

```json
"provenance": {
  "schema_version": 1,
  "run_id": "20260809-175113",
  "design_path": "/home/me/DESIGN.md",
  "design_sha256": "9f2c…",
  "planner": "claude/<configured-model-label>",
  "coder": "claude-coder/<configured-model-label>",
  "verify_profile": "<digest of the EFFECTIVE gate: canonical serialization of commands, policy, per-command timeouts and gate_generated, or \"ungated\">"
}
```

A `provenance` key present in the planner's reply is **discarded before
validation**, with a `contract_deviation` journal event — not merged, not trusted,
not an error. The canonical `plan.json` fixpoint writes always carries the block;
so does the `PLAN.json` committed in the bootstrap commit (§4.3). Which fields of it
are *required* when a plan is fed back in, and what happens when they are absent, is
§7.4, and the answer there is a refusal with the digest printed — not silent
acceptance.

**Nothing in the plan is ever executed.** Every field is prompt data. An
acceptance criterion reading "run `scripts/check.sh`" causes fixpoint to run
nothing; it is text shown to a coder. This is stated in the design, not merely
true, so that no later change "helpfully" starts executing acceptance criteria.
(Proposal B; both critics singled it out.)

### 4.2 Deterministic validation — every refusal before the first coder session

1. `1 ≤ len(tasks) ≤ implement.max_tasks` (default 40). Over the cap is a
   **refusal, not a truncation**: "split the design." A design needing 200 sessions
   is a programme, not a run.
2. Ids unique, non-empty, `^[A-Za-z0-9._-]{1,16}$` — they land in log filenames and
   commit subjects, so they get path-identifier discipline.
3. `depends_on` may name **only earlier ids**. Forward references, self-references
   and unknown ids are refused. This one rule makes the array a valid execution
   order with no topological sort, makes "skip dependents" a transitive walk over an
   already-sorted DAG, and makes a cyclic plan *impossible* rather than *detected*.
4. `files` non-empty, `≤ implement.max_files_per_task` (default 12), each a
   relative slash path with no `..`, no leading `/`, not under `.git/` or
   `.fixpoint/`, not one of the four control artifacts (`DESIGN.md`, `PLAN.md`,
   `PLAN.json`, `.gitignore` — §4.3), and not matching the mandatory credential
   patterns the config layer already enforces. The list is advisory to the coder — coders are not
   confined to it, which is an existing property of the tool — but a plan naming
   `~/.ssh/authorized_keys` was written by something that misunderstood or was
   steered, and refusing it costs nothing.
5. `acceptance` non-empty. A task nobody can state acceptance criteria for is two
   tasks or none, and it is also a task whose commit body would say nothing.
6. **Coverage.** fixpoint extracts the design's section headings from the snapshot
   itself — not from the planner — and requires each to appear in `coverage`, mapped
   either to at least one task id (which must exist) or to a non-empty
   `out_of_scope` reason. Unreferenced headings are a refusal naming them. The rule
   is defined against documents as they actually arrive, not against a convention
   nothing enforces:
   - **Which headings.** ATX (`#`…`######`) and setext (`===` / `---` underlines,
     mapped to levels 1 and 2) headings are collected, skipping fenced code blocks.
     The **outline level** is the shallowest level that occurs **more than once** in
     the document. A design with a single `#` title and `##` sections outlines at
     `##`; a design whose sections are `#` outlines at `#`; a design with `# title`
     and `###` subsections outlines at `###`. The chosen level and the extracted
     headings are printed at preflight and recorded in `PLAN.md`'s provenance, so
     the operator can see what the rule read before a session is spent. **Duplicate
     headings at the outline level are a preflight refusal** naming them: heading
     text is the join key shared by the extractor, `coverage` and `design_refs`,
     and two sections with one name collapse into one identity — a plan could
     cover one and silently drop the other (review run 20260813-003817). Renaming
     a section costs the author a word; disambiguating a collision downstream
     costs a schema.
   - **When it cannot read the document.** If no level qualifies — no headings at
     all, or a single heading and nothing else — the coverage rule has nothing to
     check, and an unchecked rule that reports success is worse than no rule. This
     is a **preflight refusal**, before the planner session: *"the design has no
     section outline fixpoint can use for coverage checking (headings found: 1).
     Add `##` sections, or pass `-no-coverage-check` to run without this check."*
   - **When it is switched off.** `-no-coverage-check` is a flag, never a config key
     (a config can ship inside the thing being reviewed). A run with the check off
     says `coverage: unchecked` in `PLAN.md`'s provenance header, in the scoreboard
     header, in the run summary, and in `plan_finished{coverage:"unchecked"}` — and
     the summary line reads *"no automated check ruled out the plan dropping part of
     the design."* Nothing in a run report may imply the check ran when it did not.
7. **Fit.** The plan must be executable inside the run's deadline **at the worst
   case the configuration actually permits**, not at an optimistic one attempt:
   `len(tasks) × max_task_attempts × (session_timeout + gate_worst + 1 min)`
   must not exceed `implement.max_run_duration`, where **`gate_worst` is the sum
   of every configured command's timeout** — the verify executor applies the
   timeout per command and runs them sequentially, so a three-command gate under a
   10-minute timeout is a 30-minute worst case, not a 10-minute one (review run
   20260813-003817; the earlier formula undercounted exactly this). Over it is a
   refusal naming every number — *"this plan needs at least 50h50m at 2 attempts
   per task and a 30m worst-case gate; max_run_duration is 32h. Raise it, cut the
   plan, lower max_task_attempts, or tighten verify.timeout."* — costing one
   planner session and no coder session. The number the refusal quotes and the
   number the run can spend are the same number; an expected-case model here was
   rejected in review (run 20260812-183115) because the common run — two or three
   tasks needing their second attempt — would still hit the deadline mid-run,
   which is the exact failure this rule exists to refuse. If either timeout is unbounded or not reachable as a value
   (§0), the check is **skipped and says so** in the same places coverage says so.
   This rule exists because the alternative, discovered in review, is a plan the
   validator calls legal that is arithmetically guaranteed to hit the deadline
   mid-run, after the money is spent (§11).

Rules 6 and 7 are in neither proposal. Rule 6 is the cheapest deterministic answer
to Critic 2's strongest finding, that a plan can silently drop a whole section of
the design and nothing downstream would ever notice (§11 records the reasoning and
its limits). Rule 7 is what keeps §12.2's promise — that every guessed number
refuses at startup rather than truncating late — true of the deadline as well.

A plan that fails validation costs exactly **one** planner session and zero coder
sessions. The raw reply is persisted before validation, so the operator sees what
was actually returned.

### 4.3 Where each fact lives

| fact | home | owner | lifetime |
|---|---|---|---|
| the design as every phase reads it | `.fixpoint/<ts>/assignment/` (a copy) | fixpoint | the run |
| the design for the built project | `<project>/DESIGN.md`, **byte-exact**, committed | the project | forever |
| the design's sha256 | commit trailers + `PLAN.md` header + `PLAN.json` provenance | fixpoint | forever |
| the planner's raw reply | `.fixpoint/<ts>/round-0/plan-<agent>-*.json` | fixpoint | the run's artifacts |
| the validated plan, canonical | `.fixpoint/<ts>/plan.json` | fixpoint | the run's artifacts |
| the validated plan, in the project | `<project>/PLAN.json`, committed **once**, never edited | the project | forever |
| the plan as a human artifact | `<project>/PLAN.md`, committed **once**, never edited | the project | forever |
| **every task's outcome, built or not** | the write-target's commit trailers (`Fixpoint-Task` + `Fixpoint-Outcome`, §5.4) | the project | forever |
| live per-task status | `.fixpoint/<ts>/status.json` (rewritten per task) + journal | fixpoint | the run's artifacts |
| **a continued run's artifacts** | `<project>/.fixpoint/<ts>/` — `-continue` has no read-target, so the project's own ignored scratch root is the only home; exempt from every ignored-path rule (§5.2) | fixpoint | the run's artifacts |
| per-task outcome, timing, tokens | `journal.jsonl` + `RunSummary.Tasks` | fixpoint | the run's artifacts |
| the repository-invariant baseline, when it MOVED | `.fixpoint/<ts>/repostate.json` — before, after, and the named diff | fixpoint | the run's artifacts |
| the implementation | the write-target's commits | the project | forever |

**Three of these homes were named here and written by nothing** until review
runs 20260813-180828 and 20260813-222753 (`plan.json`, `status.json`,
`repostate.json`). They exist now, and the reason to build them rather than
delete the promise is that each answers a question an operator actually asks:
`status.json` is rewritten atomically after every task and is the only live view
of a run that may last thirty hours unattended; `repostate.json` is written when
an invariant moves, because §8 sends an operator diagnosing the loudest failure
in the tool to exactly this file and a one-line journal string is not the
pre-image; `plan.json` is the validated plan WITH its provenance, which the
planner step artifact is not — that one is logged before provenance is injected.
`repostate.json` is at the run root rather than under `round-<n>/` because an
implement run has one round.

Snapshotting the design earns its keep twice here: the run is long (tens of
sessions), so the odds that someone edits `DESIGN.md` mid-run are real, and every
task must implement the same document the planner decomposed.

**`DESIGN.md` in the project is byte-identical to the input.** No banner, no
header, no rewriting — altering the bytes would break the hash match with the
document review-design approved. Provenance lives in the commit trailers, in
`PLAN.md` and in `PLAN.json`, not in the reviewed artifact's bytes. (Proposal B.)

**`PLAN.md` and `PLAN.json` are committed once, in the bootstrap commit, and never
rewritten.** `PLAN.md` is fixpoint's own rendering of the validated plan, with a
provenance header: run id, planner agent, coder agent, the design's path and
sha256, the coverage level (or `unchecked`), the configured gate commands, and
whether the run is **ungated**. `PLAN.json` is the canonical validated plan with its
provenance block — the machine-readable twin, and the thing that makes a repository
fixpoint built self-describing enough to be *continued* (§5.5) instead of only
rerun. Progress does *not* live in either file. Proposal A rewrote and re-committed
`PLAN.md` at the end of the run; that is rejected (§11) — it inserts a tool-authored,
un-gated commit into a history whose entire value is one-task-one-revert, and Critic
2 showed it also leaves HEAD in a state the gate never saw. What was and was not
built lives in the commit trailers, the run summary and the scoreboard.

**`DESIGN.md`, `PLAN.md`, `PLAN.json` and `.gitignore` are control artifacts, and
the protection is mechanical, not declarative.** Declaring them immutable while
they sit as ordinary tracked files in the coder's worktree protects nothing — a
coder could edit the design of record and pass an unrelated build gate (review
run 20260812-183115). So: the plan validator refuses a task whose `files` name one
(§4.2 rule 4); after every coder session their blob hashes are compared against the
bootstrap commit's, and a change is restored from the bootstrap blob with the task
failed as a contract violation (§5.2 step 5); the commit step never stages them
(§5.2 step 8); and `-continue` reads them from the bootstrap commit's blobs, never
from the worktree, so orchestration state cannot be altered by anything a session
wrote. `.gitignore` is on the list because it decides what fixpoint's own census
can see: it is written once by fixpoint from the operator's `gitignore_seed`
(§5.1), and no agent may author or amend the exemption set that governs the
tool's own invariants.

**The plan is not written into the project's `.fixpoint/`.** Proposal B put it
there; Critic 1 flagged that `.fixpoint/` is fixpoint's own artifact root and
doubles as the pathspec excluded from round commits, so a later `fix-code` run
would write its artifacts alongside a committed file, exclude that file from its
own commits, and could remove it during cleanup. The bootstrap commit's
`.gitignore` contains `.fixpoint/` precisely so later runs have scratch space; a
durable artifact must not live in scratch space. `PLAN.md` and `PLAN.json` sit at
the project root.

### 4.4 What the coder is given, and what it reads for itself

The coder does **not** receive the whole design in each of forty prompts. It
receives the design's heading outline, the sections its task's `design_refs` name
(quoted and defanged, clamped at 32 kB with any elision stated), and the sentence
*"the full design is at `DESIGN.md` in your working directory."*

Agents are agentic and read files. The consequence is stated rather than hidden:
the coder reads those bytes **unfenced**, which is the posture of every fixpoint
agent reading a repository, and is exactly what `-trusted-target` is asserted for.

## 5. The run, task by task

One task = one coder session = one commit = one round directory
(`.fixpoint/<ts>/round-<n>/`, so log-directory templating needs no change and
artifacts stay navigable).

### 5.1 The bootstrap commit, the hardened init, and the honest exemption

The repository is created with hardening applied at `git init`, not assumed from
its youth:

- `git -c init.defaultBranch=main init` — the operator's global default branch is
  not this tool's business to guess.
- `core.hooksPath` is set, repo-locally, to an **empty directory inside the
  repository's own `.git/`** — `.git/fixpoint-hooks/`, created empty at init — so a
  hook file dropped into `.git/hooks/` by anything never runs. Inside `.git/`, not
  inside the run's artifacts: an earlier draft pointed it at the artifact tree, and
  review run 20260812-183115 named the consequences — the delivered repository would
  carry an absolute path into a directory §5.5 itself calls transient, the operator's
  hooks would be silently disabled by a dangling reference, a recreated directory
  would execute for whoever owned it, and the §5.2 step 4 invariant would be checking
  the emptiness of a directory outside the repository it claims to validate. The
  setting is durable on purpose (later `fix-code` runs want the same posture) and is
  named in the run report at handoff, so it is a disclosed property, not a silent
  one. Every fixpoint-owned git invocation in this pipeline additionally passes
  `-c core.hooksPath=.git/fixpoint-hooks` and commits with `--no-verify`, so the
  protection survives a coder that edits the repo config back.
- An explicit fixpoint committer identity is written repo-locally rather than
  inheriting ambient global git config.
- `.gitignore` in the bootstrap commit contains `.fixpoint/` plus the stack config's
  `implement.gitignore_seed` entries (e.g. `node_modules/`, `dist/`, `target/`),
  because a build's output being ignored from commit one is what keeps §5.2 step 7
  from having to delete anything in the common case. It is a **control artifact**
  (§4.3): fixpoint writes it once, no session may change it, and a missing entry is
  the operator's to add via `gitignore_seed` — the exemption set that decides what
  the census can see is never agent-authored.

Then one tool-authored commit containing `DESIGN.md`, `PLAN.md`, `PLAN.json` and
`.gitignore`, and no source code:

```
fixpoint: initialize implementation of "a browser card game of Prší"

Fixpoint-Run: 20260809-175113
Fixpoint-Producer: fixpoint implement-design
Fixpoint-Phase: bootstrap
Design-SHA256: <digest>
Planner: claude/<configured-model-label>
Coder: claude-coder/<configured-model-label>
Verify-Profile: <digest of the effective gate: commands, policy, timeouts, gate_generated -- or "ungated">
Coverage-Check: "##" (7 headings) | unchecked
```

It exists because HEAD must exist before anything can commit against it, because
the reviewed artifact belongs in the history of the thing it produced, and because
a repository that carries its own plan and design hash can be continued by a later
run without a side-channel. Critic 2 correctly observed that Proposal A's equivalent
root commit is not gated and therefore dents "the project builds after every
commit." The honest answer is an exemption, stated rather than glossed:

> The bootstrap commit contains no source code. There is no build to run in a tree
> with no sources; running the configured gate there would fail for every stack and
> prove nothing. The build-after-every-commit property is asserted **from the first
> task commit onward**, and it is task 1's job (§6) to establish it.

The commit's own subject and the scoreboard both label it `bootstrap (not gated,
no sources)`. An implementation that quietly gated it, or quietly did not mention
it, would be worse than either.

Immediately after the bootstrap commit, fixpoint records the **repository-invariant
baseline** (§5.2 step 4): the digest of `.git/config`, the effective `core.hooksPath`
and the emptiness of the hooks directory it points at, the full ref list, the
absence of any nested `.git` in the worktree — and the git metadata that controls
what the census itself can see: the digests of `.git/info/exclude` and
`.git/info/attributes` (both written empty at init), and the absence of
`.git/objects/info/alternates` and of a `shallow` marker. Review run
20260812-183115 showed why the list must include them: one line in
`.git/info/exclude` makes a coder's source path invisible to `git status`, so the
gate passes on a file no commit contains — hardening that guards `.git/config` but
not the exclude file guards the lock and leaves the door open.

### 5.2 The per-task sequence

For each task in plan order:

0. **Clean-tree precondition.** The write-target must be clean relative to HEAD:
   no modified tracked files, no staged changes, and no un-ignored untracked files.
   This is an invariant every other step in this section relies on — it is what
   makes "the tree is dirty" mean "this session did something" — so a violation is
   not a task failure but a **repository-invariant failure that stops the run**,
   naming the offending paths. Steps 3, 5, 7 and §5.3 are each written to leave the
   tree clean; if one of them did not, the design is wrong somewhere and continuing
   would attribute one task's residue to the next task's coder.
1. **Skip check.** If any task this one transitively `depends_on` failed or was
   blocked, skip without spending a session and record *which* dependency stopped
   it.
2. **Record the base.** `HeadSHA`, the repository-invariant snapshot, and the
   **ignored-path census**: every ignored path, minus fixpoint's own artifact
   root (`.fixpoint/` is the run's scratch on a continued run, and a census that
   reads the run's own journal writes would fail every task on the tool's own
   bookkeeping — review run 20260813-003817). A stat walk (path, size, mtime),
   not a digest walk — `node_modules/` makes hashing the ignored tree
   unaffordable — and honestly scoped: it reliably detects **creation**, and its
   modification diff is a journal signal, not an enforcement mechanism, because
   stat metadata can be restored by anything that can write the file (§5.2
   step 6, §7.2). All before the session starts.
3. **Coder session.** Prompt in §6. Working directory is the write-target. Attempt
   number 1.
4. **Repository invariant.** After the session, HEAD first:
   - HEAD == base → normal.
   - base is an ancestor of HEAD (the coder committed despite being told not to) →
     `git reset --soft <base>`, which turns its commits into staged changes;
     proceed, and record a contract-deviation event in the journal. The work is
     kept; fixpoint's ownership of the commit is kept.
   - anything else — HEAD not a descendant of base, refs rewritten, base missing →
     **stop the whole run** with a repository-invariant failure. Continuing from a
     HEAD nobody gated is exactly what this pipeline exists to prevent.

   Then the rest of the repository, because a session can change what fixpoint is
   about to commit without touching HEAD at all. Compared against the snapshot from
   step 2: the digest of `.git/config`; the effective `core.hooksPath` and the
   emptiness of the directory it names; the full ref list (`git for-each-ref` plus
   packed-refs digest) other than the branch HEAD is on; the absence of any
   nested `.git` file or directory anywhere in the worktree; and the census-bearing
   metadata of §5.1's baseline — `.git/info/exclude`, `.git/info/attributes`,
   no alternates, no shallow marker. **Any mismatch stops the run** with the same
   repository-invariant failure, naming what changed. This
   is not a task failure: a session that rewrote the repository's own configuration
   is a session whose other output cannot be reasoned about either.
5. **Reconcile the report against the tree.** "Dirty" here means dirty relative to
   HEAD, which step 0 guaranteed was also the state at session start.
   - **control artifacts first**: the blob hashes of `DESIGN.md`, `PLAN.md`,
     `PLAN.json` and `.gitignore` are compared against the bootstrap commit's. A
     change is restored from the bootstrap blob and the task fails as a contract
     violation (§4.3) — session-scoped, like a coder that commits, because the
     authoritative bytes were never lost; the run continues.
   - reported `implemented`, tree clean → contract violation, task failed
     ("the coder reported implementing T07 but left the working tree unchanged").
   - reported `already_satisfied` or `blocked`, tree dirty → edits discarded
     (§5.3's untracked-inclusive discard, followed by the same clean assertion),
     task failed. A session that disclaims work must not leave work behind.
   - reported `already_satisfied` → the coder must name the earlier task ids that
     cover it; fixpoint checks each names an *earlier* task that this run or an
     earlier run of this plan actually implemented — a code commit, not an outcome
     marker (§5.4). Uncorroborated → contract violation, task failed.
   - reported `implemented`, tree dirty → on to the gate.
6. **Pre-gate census, and the ignored-path reconciliation.** Record, for the
   un-ignored working tree: the set of untracked paths **with a sha256 for each**,
   and the set of modified tracked paths with their digests. Digests, not just
   paths, because a task's new files are untracked by definition and a gate that
   rewrites one of them changes no path set at all. During the same walk, the
   **byte bound** is enforced: if the session's un-ignored changes exceed
   `implement.max_task_bytes`, the attempt fails before anything is hashed or
   gated, and the oversized tree goes through a discard that **avoids the object
   database** — checkout-and-clean semantics rather than a stash, since stashing
   is exactly the cost the ceiling exists to refuse (§5.3; review run
   20260813-003817 caught the earlier text skipping the cleanup along with the
   hashing, which left a dirty tree that stopped the whole run at the next task's
   step 0).

   Then the ignored tree is compared against step 2's stat census — minus the
   artifact root — and ignored paths the **session created** are deleted before
   the gate runs, with a journal note: a coder that ran the build to test its
   work loses nothing the gate does not recreate from committed sources.
   Session-*modified* ignored paths are recorded in the journal and the task
   report but fail nothing: an earlier revision made them a `hidden-state` task
   failure, and the four-reviewer round killed it twice over — ordinary build
   churn rewrites caches in place from task 2 onward (the coder running
   `npm test` is behaviour §6 encourages), and stat metadata cannot carry an
   integrity claim anyway, because a same-size write can restore its own mtime.
   The property the rule was protecting — **the commits alone satisfy the
   gate** — is enforced where it can actually be checked, by the clean-clone
   check in §7.2, and the census diff remains as the pointer that tells an
   operator *where* to look when that check fails.
7. **Gate.** `verify.Run` over the write-target. Its output is target-authored
   text and is fenced, quoted and defanged before it reaches any prompt. Afterwards,
   the tree is compared against the step 6 census, and every difference is
   classified:
   - **Gate mutation of sources**: the gate modified or deleted any **tracked** file,
     **or** modified or deleted any untracked file that existed in the pre-gate
     census. The task fails with a distinct `gate-mutated-sources` outcome naming
     the paths. Including the coder's brand-new untracked files in this rule is the
     point: on a greenfield project almost everything task 1 writes is untracked,
     and a formatter or codegen step in the gate would otherwise rewrite the
     session's source and have those bytes committed under the coder's attribution
     without any verification pass over them.
   - **Gate-maintained committed files**: paths named in the stack config's
     `implement.gate_generated` list (`go.sum`, `package-lock.json`, `Cargo.lock`,
     …) that the gate created or rewrote. These are **staged into the task commit**
     and attributed to the gate in a trailer (`Fixpoint-Gate-Wrote: go.sum`) —
     neither a mutation failure nor removable output. Without this third,
     config-owned category the two-bucket rule gets lockfiles exactly wrong (review
     run 20260812-183115): a lockfile the gate creates would be deleted from every
     commit forever, and a lockfile the gate updates would fail the task — while the
     log recommended a `.gitignore` entry, the one remedy that is always wrong for a
     file clones need. Coder-written changes to these paths remain ordinary
     sources; the list is the operator's, like the gate itself.
   - **Gate output**: un-ignored paths that did not exist before the gate ran. These
     are excluded from the commit, **and removed from the write-target after every
     gate run, before any commit** — not deferred until after a successful one,
     because a failed gate's droppings would otherwise sit in the tree for the next
     attempt and a crash between commit and a deferred cleanup would leave a dirty
     tree `-continue` refuses. They are named in the task's log line with the note
     that the operator's `gitignore_seed` needs an entry for them. Removal is what
     makes the exclusion mean anything: a build artifact left in place is simply a
     file that the *next* task's census sees as pre-existing, and therefore commits,
     attributed to the next task's coder — an invariant honoured for exactly one
     task. Removal is safe because the gate produced these bytes from committed
     sources and re-running the gate reproduces them, and it is usually a no-op
     because `gitignore_seed` (§5.1) covers the normal cases: **ignored paths are
     never committed, and outside step 6's reconciliation, never touched.**
   - On gate failure the **attempt fails** — there is no in-session correction
     pass. An earlier draft allowed one; review run 20260812-183115 showed it
     cannot be had honestly: the agent interface is a one-shot process, so a
     "correction" is a second session whose edits land in a commit the trailers
     attribute to one — the exact rule the assignment calls hard-won — and the
     failed gate's output contaminates the correction's census. The retry budget
     lives in one place, §5.3, where each attempt is one session, one gate, one
     census, discarded whole or committed whole.
8. **Commit.** The index is not trusted: fixpoint first resets the index to HEAD
   (worktree untouched), then stages exactly the path set it computed in step 7 —
   coder-authored additions and modifications, plus gate-maintained committed
   files, minus gate output, and **never a control artifact** (§4.3). A gitlink can
   never be staged (step 4 already refuses a nested `.git`). Subject
   `fixpoint: <id> — <title>`; body carrying the task goal and acceptance criteria;
   trailers:

   ```
   Fixpoint-Run: 20260809-175113
   Fixpoint-Task: T03
   Fixpoint-Outcome: implemented
   Fixpoint-Attempt: 1
   Design-SHA256: <digest>
   Coder: claude-coder/<configured-model-label>
   Verification: passed: build, vet, test
   ```

   For an ungated run the last line reads exactly
   `Verification: skipped (no operator gates configured)` — **never** "passed"
   (Proposal B; both critics). Every agent-authored field goes through
   flatten + redact before it lands anywhere fixpoint publishes. Gate output was
   already removed before the commit (step 7), so the tree is asserted clean
   immediately after it, which is step 0 for the next task.

   A task that ends **without** code — `already_satisfied`, `blocked`, or `failed`
   after its attempts — gets an **empty outcome marker commit** instead (§5.4):
   same trailers, `Fixpoint-Outcome` naming the outcome and its reason
   (`Fixpoint-Covered-By: T04`, `Fixpoint-Blocked-By: T02`, or the failing check),
   subject `fixpoint: <id> — <title> [already_satisfied]`, no tree change, not
   gated — the bootstrap commit's exemption argument (§5.1) verbatim: there are no
   bytes to gate.
9. **Journal + log line**, then the next task.

### 5.3 Attempts: a bounded second session, not a bounded human

`implement.max_task_attempts`, default **2**.

- An attempt is **exactly one coder session and one gate run** — no correction
  pass inside it (§5.2 step 7).
- **Every attempt that does not reach a commit exits through this discard** —
  gate failure, dead session, contract violation, control-artifact edit,
  gate-mutated-sources, byte-bound breach, all of them. Review run
  20260813-003817 found the earlier text triggering the discard on only two of
  the failure paths; each of the others left the failed attempt's residue in the
  tree, where the next task's step 0 reported it as a fixpoint bug and stopped a
  30-hour run over one failed task. One variant: the byte-bound breach discards
  via checkout-and-clean instead of a stash (§5.2 step 6), because writing the
  oversized tree into the object database is the cost the bound refuses.
- On discard, **all of the attempt's changes are
  discarded** and the next attempt is a *fresh* session from the last accepted commit,
  carrying the prior failure diagnostic as prompt text and **none** of the prior
  file changes. One commit stays attributable to one session; several sessions'
  partial output never collapses into one ambiguously attributed commit.
  (Proposal B's retry rule; Critic 1 named it as the thing most designs get wrong.)
- After the last attempt, the task is failed and its dependents are skipped.

**The discard is untracked-inclusive, and this is not a detail.** In the fix loop,
"the changes" are edits to files that already exist, so a plain stash is a complete
discard. Here almost everything a task produces is a **new file**, and a
tracked-only stash would leave every one of them sitting in the tree — so attempt 2
would start on attempt 1's rejected output, its commit would mix two sessions, and
in the worse variant attempt 1's half-written files would combine with attempt 2's
to satisfy the gate, producing a "passing" task no single session authored. So:

- the discard stashes **tracked and untracked** un-ignored changes together
  (`git stash push --include-untracked` semantics), under a stash message naming
  the run, task and attempt;
- ignored paths the session **created** are deleted, using step 2's ignored-path
  census — a dead session may never have reached step 6's reconciliation, and an
  attempt's residue surviving under an ignore rule is the same
  two-sessions-one-commit bug in slow motion. Pre-existing ignored paths are left
  alone;
- and immediately afterwards fixpoint **asserts the tree is clean relative to
  HEAD** (§5.2 step 0's check). If anything remains, the run stops with a
  repository-invariant failure rather than starting attempt 2 on a polluted base.
  That assertion is deliberately redundant with the stash: it is the thing that
  turns §0's unverified `StashDirty` hypothesis into a loud failure instead of a
  silent attribution bug.

The same discard-then-assert path is used by §5.2 step 5 for a session that
disclaims work while leaving files behind, and by the interrupted-session path.

**Failed work is discarded, not salvaged.** This diverges from the fix loop on
purpose: salvage there is justified because the *next round re-reviews* the partial
work. Implement-design has no round after a task. Committing half a task would put
every later task on a base nothing has looked at, in a repository whose whole claim
is "every commit passed the gate." Discarded changes are recoverable from the
attempt's artifacts and from `git stash list` (fixpoint stashes rather than hard-
resets, so nothing is destroyed), but they never reach a commit.

### 5.4 Task outcomes

| outcome | commit | dependents | run exit |
|---|---|---|---|
| `implemented` | code commit | proceed | 0 |
| `already_satisfied` — corroborated by named earlier implemented tasks | empty marker | proceed | 0 |
| `carried` — recorded by an earlier run of this plan, found by trailer on `-continue` | already there | per its outcome | 0 |
| `blocked` — cannot be done as specified, **confirmed by two independent sessions** | empty marker | skipped | 2 |
| `failed` — gate, contract violation, or gate mutation, after all attempts | empty marker | skipped | 2 |
| `skipped` — a dependency did not land | **none — derived** | skipped | 2 |
| **infrastructure** — provider refusal (the 402/429/5xx `error_status` path), dead session with no output, missing gate executable, or a gate command the operator declared `infra: true` exiting non-zero | **none — not terminal** | untouched | 2 |

**Infrastructure failures are not task outcomes** (review run 20260813-003817).
"The coder's work failed the gate" is a fact about the code; "fixpoint could not
obtain a coder session" is a fact about the morning — and the earlier text let a
two-minute provider outage burn both attempts, write a permanent `failed` marker,
and kill the task's whole dependent subtree on every future `-continue`. An
attempt that dies from infrastructure does not count against
`max_task_attempts`, and only results about the work — gate failures, contract
violations, gate mutation — reach the immutable history.

**The gate's dependencies count too** (review run 20260813-222753). The rule
above covered the AGENT's provider and stopped there, so a gate command that
*ran* and exited non-zero because its own dependency was unreachable — `npm ci`
against a registry returning 503 — was recorded as a verdict on the code. The
shipped node config's first gate command is exactly that. A command the operator
marks `infra: true` in `verify.commands` is therefore classified like a provider
refusal: no attempt consumed, no marker, backed off and fed to the breaker.
Opt-in and operator-declared, because fixpoint cannot tell a network failure
from a real one by reading an exit code — only the person who wrote the command
knows what it reaches.

**Waiting comes before the breaker.** An infrastructure failure is retried with
an increasing wait — 1m, 5m, 15m, 30m — and the run stops incomplete (exit 2)
only when `implement.max_infra_tries` (default 4) is spent, with no marker for
the task in flight. A strike is one try, not one exhausted ladder. The waits are
journalled (`infra_backoff`) and charged against `max_run_duration`, so the
deadline stays honest about time actually spent. `402` skips the ladder
outright: the account is out of credit, and every retry buys the same answer
more slowly. Without any wait at all — the first shape this shipped in — the
second call landed inside the same rate-limit window as the first, so an
overnight run could be dead four minutes after a routine 429 (review run
20260813-180828).

**Every processed task leaves exactly one commit**, because the commit trailers
are the resume state (§5.5) and a state that cannot represent the outcomes §5.4
defines breaks `-continue` on ordinary runs — review run 20260812-183115's most
corroborated finding. A task that produced no code gets an empty, tool-authored
**outcome marker commit** (§5.2 step 8) with a **versioned trailer schema** —
`Fixpoint-Marker: 1`, `Fixpoint-Task`, `Fixpoint-Outcome`, a normalized
`Fixpoint-Reason` code (`covered_by`, `design_conflict`, `gate_failed:<check>`,
`contract_violation`, `gate_mutated_sources`, `byte_bound`), the related ids
(`Fixpoint-Covered-By: T04`, `Fixpoint-Blocked-On: §3,§5`), and one flattened,
redacted, ≤200-character `Fixpoint-Detail` line — so the full outcome vector
*and its reasons* are reconstructible from HEAD alone, on a machine where the
run's artifacts never existed; a continued run's report derives from these
durable fields and only decorates them from artifacts when it has any (review
run 20260813-003817 — the earlier schema promised reasons the trailers could
not carry). The unabridged detail lives in the journal. Reverting a
marker is a no-op, so one-task-one-revert survives; a marker changes no tree
bytes, so the rule §8 protects — no tool-authored *diff* enters the history
between coder tasks — survives too, and the bootstrap commit already establishes
the ungated tool-commit precedent. `skipped` is the one outcome **derived, not
recorded**: it is a pure function of the plan's `depends_on` graph and the
recorded `failed`/`blocked` markers, and recording it would bury a 40-task run's
history under empty commits the moment task 2 fails.

`already_satisfied` exists because planners over-decompose, and a run that failed
because task 9 turned out to be part of task 8 would be failing on a taxonomy
question — but Proposal A let it pass unchecked, which is how an ungated run exits
0 with functionality absent (Critic 2). The corroboration requirement in §5.2 step 5
is the fix, plus one aggregate guard: if `already_satisfied` exceeds
`implement.max_vacuous_frac` of the plan (default `0.34`), the run ends
**incomplete (exit 2)** with "the plan was decomposed badly: N of M tasks were
already covered by earlier work." That threshold is a guess and is a config key for
that reason; it refuses loudly rather than truncating silently. `carried` tasks
count toward the plan's denominator but never toward the vacuous numerator.

`blocked` exists because a design can contradict itself, and the coder is the first
thing in the pipeline positioned to notice. That is a real finding about the
design and it must be loud. It is also a model's opinion with the largest blast
radius in the taxonomy — one hallucinated "the design contradicts itself" on an
early task permanently skips every transitive dependent — so it gets the same
treatment `already_satisfied` got and for the same reason (review run
20260813-003817): the report must **cite the design sections in conflict**
(refused as a contract violation without them), and a first `blocked` is
confirmed by the second attempt's fresh session before the marker is written.
Two independent sessions agreeing that the design is self-contradictory, with
citations, is a finding; one session saying so is a guess that would have cost
the rest of a 40-task plan.

### 5.5 The run has a deadline, and the deadline has a continuation

`implement.max_run_duration`, default **32h**, checked between tasks. On expiry the
run stops cleanly, reports incomplete (exit 2), and names the task it stopped
before. Per-session and per-gate timeouts alone leave a hundred-task plan free to
occupy a machine for days with no deadline at which it fails loudly (Critic 1).

Two things make the deadline honest rather than a trap:

**It is checked against the plan before any coder session runs.** §4.2 rule 7
refuses a plan that cannot fit, naming the arithmetic — and the arithmetic is the
worst case the configuration permits, not the happy path:

```
tasks × attempts × (session_timeout + Σ per-command gate timeouts + 1 min)
      + the clone gate runs clean_check buys (one per task at `every`, one at `last`)
```

Both correction terms were learned the hard way. The per-command sum replaced a
single gate timeout in revision 3 (review run 20260812-183115); the clean-check
term was missing until review run 20260813-180828 found that a 15-task plan the
validator called legal needed ~38h against a 32h deadline at `clean_check:
every`, hitting the deadline around task 12 — the exact trap rule 7 exists to
refuse, re-opened by the check §8 tells operators to switch on.

The same arithmetic produces the cap quoted to the planner, because it is
literally the same function (`Rules.Admitted`). It has to be: when the two were
computed separately the prompt asked for "8–25 tasks" beside a computed "at most
15", and the planner — the only agent that decides task size — resolved the
contradiction by making tasks too big for one session.

At the shipped 32h and `clean_check: last`, that admits **20** tasks for
implement-go (30m session, 3 × 5m gate), **17** for implement-node (3 × 8m), and
**30** for the ungated implement-web; three fewer each at `clean_check: every`.
`max_tasks: 40` is the absolute ceiling, reachable only by raising the deadline
or shortening the gate. An operator running smaller plans lowers the deadline;
the fit rule keeps whichever number is set honest.

**An expired run can be continued, not only rerun — specified here, NOT YET
IMPLEMENTED.** Everything in the rest of this subsection describes `-continue`,
which this version does not ship: `cmd/fixpoint` defines no such flag and
nothing reads the trailers back. It is specified because the *write* side is
built to it — the committed `PLAN.json`, the bootstrap trailers and §5.4's
outcome markers all exist to be replayed — and because the read side is a
self-contained increment on top of them.

Until it lands, the honest recovery for every stop below is: **the committed
tasks stand and were gated; the remainder must be rebuilt by a fresh run into a
fresh directory**, or finished by hand with `fix-code` over the half-built
project. §8's recovery column says so, and the messages the run prints when it
stops say so. Nothing in the tool claims re-entry it cannot perform.

`-continue <project>` will resume into a repository fixpoint itself built:

```sh
fixpoint implement-go -continue ~/src/prsi -trusted-target
```

The repository is self-describing, so no side-channel state is needed:

- the plan is read from the committed `PLAN.json` (not from run artifacts, which
  may be on another machine or long deleted);
- the design is read from the committed `DESIGN.md` and must hash to the
  `design_sha256` in `PLAN.json`'s provenance, and to the bootstrap commit's
  `Design-SHA256` trailer; any mismatch is a refusal;
- **every recorded outcome is replayed from the commit trailers**: each commit
  reachable from HEAD carrying `Fixpoint-Task: <id>` — a code commit or an outcome
  marker (§5.4) — marks that task `carried` with its recorded
  `Fixpoint-Outcome`; skips are re-derived from the plan and the carried
  `failed`/`blocked` outcomes; the run enters BUILD at the first task with no
  commit of either kind. Carried outcomes are **final**: a carried `failed` or
  `blocked` task is not retried — retrying against a corrected design is a fresh
  run, and a `-retry-failed` variant is an open question (§12), not a silent
  behaviour;
- the run refuses unless HEAD is the bootstrap commit or a fixpoint task commit,
  the tree is clean (§5.2 step 0), the recorded tasks — commits and markers,
  plus the derived skips — form a prefix of the plan order, and the configured
  `Verify-Profile` digest matches the bootstrap trailer — a project half-built
  under one gate must not be finished under another and reported as one thing;
- **the trust boundary is re-established rather than inherited.** Between runs the
  tree was outside fixpoint's lock and anyone could have touched it, so `-continue`
  runs the *first-contact* hardening §2 says drops out for a fresh directory:
  the untrusted-git-config guard, a hooks check, a symlink check on `.fixpoint/`,
  and the repository-invariant baseline is taken fresh. `-trusted-target` is still
  required.
- the deadline restarts; the report covers the whole plan, with `carried` tasks
  shown with their recorded outcome and reason (read from the marker trailers, so
  a `blocked` task's finding survives the first run's artifacts) and attributed to
  the run id in their trailers. What the markers cannot carry — the first run's
  timings and token counts — is reported as unavailable, not as zero.

`-continue` is deliberately narrow: same plan, same design, same gate, clean tree,
prefix history. It is not a general resume-anything facility, and it is not a
substitute for `fix-code`. What it buys is that the deliberate deadline path — and
Ctrl-C, and a machine reboot — no longer discard every gated commit and force the
operator to spend the same hours reaching the same task.

## 6. Prompts

**`config/prompts/implement-plan.md`** (planner, read-only). Carries the design
whole, fenced and defanged, plus the operator's configured gate commands verbatim,
and demands:

- Each task is **one agent session with a ~30 minute budget, producing one commit**.
  Stated in those numbers, because "small" is not a unit.
- **Task 1 must leave the project passing these commands** — the commands are
  quoted in the prompt. The skeleton, the manifest, the entry point, and whatever
  minimum makes `go build ./... && go vet ./... && go test ./...` (or the operator's
  actual list) succeed. Every later task must keep them passing.
- **`.gitignore` is fixpoint's, not a task.** The bootstrap commit wrote it from
  the operator's `gitignore_seed`, and no task may modify it (§4.3). A plan that
  schedules ignore-file work was written against a different tool.
- **You are shown the gate; you do not choose it.** Do not name build, test or
  install commands in your output — there is no field for them and anything you
  write elsewhere is ignored. The operator configured the gate; your job is to make
  task 1 satisfy it.
- Dependencies point **backwards only**; the order you emit is the order it runs.
- Every task needs acceptance criteria a reader could check.
- Account for every section of the design in `coverage`, either by task or by an
  explicit out-of-scope reason. The prompt quotes the exact heading list fixpoint
  extracted (§4.2 rule 6), so the planner is matching against the same strings the
  validator will.
- Prefer 8–25 tasks. **At most N**, where N is what the run deadline admits (§4.2
  rule 7) — the number is computed and quoted, not left as a cap the planner
  discovers by being refused.

Showing the planner the actual commands is Critic 1's best catch against Proposal
B, which passed only gate *names*: a planner told "test" and "build" cannot know it
is targeting Go, can emit a Python plan for a Go gate, and then every task fails
from T01. The flow is config → prompt, which is safe; only prompt → command is
forbidden. For an ungated run the prompt says so explicitly, and asks for a plan
whose first task still produces a runnable artifact.

**`config/prompts/implement-task.md`** (coder). Carries: this task only; the plan's
shape (ids, titles, statuses and commit SHAs of everything already done, so the
coder knows what exists); the design excerpt and the pointer to `DESIGN.md`; the
prior attempt's gate diagnostic when this is attempt ≥ 2; and verbatim reuse of the
fix prompt's **Quality gate** paragraph — fixpoint runs the project's own checks
itself and will not commit work that fails them; never silence a checker. Plus
four rules the fix prompt does not need:

- **Do not implement future tasks.** It breaks one-task-one-commit, spends the next
  session's budget, and makes the gate unable to name the culprit.
- **Do not commit, amend, reset, or move refs.** fixpoint owns the history.
- **Do not touch `.git` — not the config, not the hooks, not the index, not any
  ref.** fixpoint checks this after every session and stops the whole run on a
  mismatch, so a stray `git config` costs the operator the rest of the run.
- **Do not edit `DESIGN.md`, `PLAN.md`, `PLAN.json` or `.gitignore`.** They are
  restored and the task fails (§4.3). Running the build or the tests to check
  your work is fine — but ignored files you create are deleted before the gate
  runs (§5.2 step 6), and nothing you place under an ignore rule can make the
  gate pass: the run's final check clones HEAD and gates the clone (§7.2), so
  work hidden from the commit is work that does not exist.
- **`blocked` must cite the design sections in conflict.** A blocked report
  without citations is a contract violation, and a first blocked report is
  re-checked by a fresh session before it is believed (§5.4).
- **Say so if this task is already satisfied** by earlier work, name the task ids
  that cover it, and change nothing. That is a legitimate answer, not a failure.

Output contract, in an `<implement>` envelope:

```json
{ "task": "T07",
  "status": "implemented | already_satisfied | blocked",
  "covered_by": ["T05"],
  "notes": "...",
  "files_touched": ["src/game.js"] }
```

## 7. Configuration, trust, and the gate on a project that does not exist yet

### 7.1 A stack is a config, not a new schema

The gate's commands must come from the operator and never from the design — but the
operator is configuring a gate for a project whose language the design has not been
read to discover. The answer is `extends` — with one constraint this design must
respect rather than wish away: **the loader's inheritance is one level deep, on
purpose** (`loadWithExtends` refuses a base that itself extends another config, so
the effective configuration stays readable from two files). An earlier draft
stacked `implement-go → implement-design → defaults`; verified against the loader,
every one of those configs fails to load (review run 20260812-183115 — the one
reuse hypothesis whose failure invalidated a section outright). So there is no
`implement-design.yaml` base file at all:

```
config/implement-go.yaml       extends defaults; roles + verify: go build / vet / test
config/implement-node.yaml     extends defaults; roles + verify: npm ci / build / test
config/implement-web.yaml      extends defaults; roles + verify.policy: off
```

Each stack config is runnable and self-contained; the shared shape numbers live in
code (`applyDefaults`), exactly as the create pipeline's do — a config carries only
what an operator would edit:

```yaml
# config/implement-go.yaml
description: Implement a reviewed design as a new Go project, one task per commit. Needs -trusted-target.
extends: defaults

roles:
  planner: { agent: claude, prompt: implement-plan }   # read-only
  coder:   { agent: claude-coder, prompt: implement-task }

implement:
  gitignore_seed: []          # go builds in-tree artifacts rarely; node: node_modules/, dist/
  gate_generated: ["go.sum"]  # files the gate maintains and the commit must keep (§5.2 step 7)

verify:
  commands: ["go build ./...", "go vet ./...", "go test ./..."]
```

Code-level defaults (config keys so a wrong guess is the operator's to correct,
§12.2): `max_tasks: 40`, `max_files_per_task: 12`, `max_task_attempts: 2`,
`max_vacuous_frac: 0.34`, `max_run_duration: 32h`, `max_task_bytes: 64MB`,
`min_free_disk: 2GB`, `clean_check: last` (§7.2).

The existing `verify` block is reused as-is. Proposal B introduced a parallel
`implement_design.verification` block; Critic 1 killed it and is right — operators
would define build and test twice, the definitions drift, and a project can pass
implement-design's gate and immediately fail `fix-code`'s gate on the identical
commit.

### 7.2 Gate policy

- **`verify.commands: []` is refused unless `verify.policy: off` is set
  explicitly.** A deliberate divergence from the rest of the tool, where an empty
  list silently disables the gate. Silence is tolerable on an existing repository
  the operator has seen; it is not tolerable here, where the project does not exist
  and an empty gate is far more likely a half-finished config than a statement
  about a browser game. The ungated case the assignment requires therefore becomes
  an **assertion** (`implement-web`, one line), recorded in `PLAN.md`'s provenance,
  in every commit trailer, and in every log line as `ungated`. Commits land ungated,
  exactly as the fix loop does on such projects. Nothing else changes — including
  §5.2 steps 6 and 7, which still run: an ungated run has no gate to create output
  or mutate sources, so the post-gate diff against the census is trivially empty,
  and the code path is the same one rather than a second one. The byte bound and
  the ignored-path reconciliation in step 6 are about the coder, not the gate, and
  apply unchanged.
- **`implement.clean_check: last | every | off`, default `last`.** At the chosen
  cadence, fixpoint clones HEAD into a run-owned temporary directory and runs the
  gate in the clone. This is the enforcement of the design's central claim — the
  committed bytes alone satisfy the gate — put where it can actually be checked
  (review run 20260813-003817). The working tree cannot carry that claim: gates
  legitimately read ignored caches the commits do not contain, coder-written
  ignored bytes cannot be reliably distinguished from build churn by stat
  metadata, and gate-created ignored state survives across tasks by design. The
  clone check ignores all of it and asks the only question that matters at
  handoff: *does a fresh checkout build?* `last` (one extra gate run per run)
  catches accumulated hidden state at the end and cannot name the culprit task —
  it says so, and `every` exists for when attribution is worth one gate run per
  task. A `last` failure ends the run **incomplete (exit 2)**: "HEAD does not
  pass the gate in a clean clone; the working tree carries state the commits do
  not." Ungated runs force it `off` — there is nothing to run in the clone. The
  per-task ignored-path census (§5.2) remains as the *pointer* — when the clone
  check fails, its journal diffs say where the hidden state accumulated.
- **`verify.policy: no_regressions` is refused.** A no-regressions baseline is
  captured on the pristine tree — and the pristine tree here is *empty*, so
  `go build ./...` fails in it, and that failure would exempt the build check for
  the whole run. A gate that can never block is worse than no gate, because the
  summary claims it ran. `must_pass`, no baseline, from task 1 onward. This is also
  what enforces build-after-every-commit: task 1's gate failing is the immediate,
  loud signal that the plan's first task was not a skeleton — one session spent,
  not twenty.

### 7.3 Trust, and other refusals

`-trusted-target` is reused, not replaced. Its existing meaning — "this target
holds only content you trust, permitting a coder that edits files with permission
checks disabled" — is exactly the assertion needed, with `target.path` being the
design. The refusal names the document:

> implement-design hands a design document to a coder that edits files with
> permission checks disabled and is not confined to the project it creates, so the
> document could steer it by prompt injection; pass -trusted-target to assert that
> DESIGN.md is a design you wrote or reviewed.

Proposal B's `--trust-config` is refused. This repository already decided that a
create run needs no trust flag (commit f424d9f), and the reasoning transfers:
implement-design has no pre-existing target that could ship a hostile config, the
gates come from the operator's own bundle, and a flag that every real invocation
must carry trains operators to always pass it — destroying the signal the existing
flag depends on. `-allow-untrusted-fix` is refused as belonging to pr mode.

New config validation, each refusal carrying its reason:

- `roles.planner.agent` required, and must be `can_edit: false` — the same rule as
  the judge and the editor, for the same reason: the thing that decides must not be
  the thing that can act.
- `roles.coder.agent` required and `can_edit: true` (existing rule).
- `roles.review.prompts` refused — a lens list means someone expected a panel; the
  lenses belong to the `review-code` run that follows. (`roles.review.agents` is
  inherited from defaults and is simply unused; refusing it would break
  `extends: defaults`, and pretending otherwise would be a lie in the error text.)
- **The run's active agent set is the planner and the coder, explicitly** —
  never the inherited reviewer pool. "Simply unused" is not enough: the preflight
  ping walks the active set, so an implement run that inherited the pool would
  ping and bill four reviewers that never run, and die on an ollama quota
  blackout before its first useful session (review run 20260813-003817 — the
  same bug the create run shipped with and fixed in f424d9f; this time it is in
  the design instead of the postmortem).
- `roles.judge`, `roles.editor`, `roles.triage`, `review.refute`, every `create.*`
  key: refused as inert.
- `loop.commit_policy` must be `per_fix`. Squashing per round or per run collapses
  the project into one commit and destroys the property the assignment calls
  hard-won. An operator who wants a single clean import has `git` for that,
  afterwards.
- **`-out` must not lie inside any git repository.** The check walks `-out`'s
  parent chain to the filesystem root looking for a `.git` file or directory, and
  refuses on the first one it finds, naming it. It is scoped to no particular
  repository, because the harm — the enclosing repo records the new project as a
  gitlink or absorbs its files, the outer tree is left dirty by a run that never
  checked it, and every later `review-code`/`fix-code` run inside the project
  operates in a nested history — is a property of *any* enclosing repository. An
  earlier draft scoped this to the read-target's repository, which is exactly the
  repository that §2 says may not exist: `-target /tmp/DESIGN.md -out ./prsi` run
  from inside any checkout would have passed. `-out` must also not be under any
  `.fixpoint/` directory. The refusal text says "inside a git repository" and now
  means it.
- Config discovery for this command must not load files from the design's directory
  or from `-out`. A repository cannot exist yet, and later-created repository files
  must not be able to alter the active run. (Proposal B; this is the "a config can
  ship inside the thing being reviewed" rule, applied to a thing that does not
  exist yet.) The same holds for `-continue`: the config comes from the operator's
  bundle, never from the project fixpoint built, because a coder session wrote
  files in that project.

  **This rule needs its own flag, and finding out why took a review** (run
  20260813-222753). Bundle discovery is anchored on the working directory, so
  the natural invocation — `cd` into the repository holding the design, run the
  implement config — makes that repository the first bundle searched. There is a
  general guard for target-supplied policy, and it is cleared by *either*
  `-trusted-bundle` or `-trusted-target` — while this section makes
  `-trusted-target` **mandatory** for every implement run. The one flag an
  operator cannot avoid was therefore admitting the design repository's own
  `verify.commands` (argv fixpoint executes itself), agent definitions and
  prompts, silently. The operator's assertion was that *this DESIGN.md is a
  design you wrote or reviewed*; it was being spent on a second, unrelated claim
  about arbitrary YAML beside it — and the gate is what §3 calls the only signal
  in this tool no model authored. So an implement run gates policy resolved from
  inside `target.path` **separately**, and only `-trusted-bundle` clears it.
- **The gate executes model-authored build code and whatever it depends on**, and
  that belongs here beside the other disclosures rather than being left implicit.
  §5.2 step 7 makes a lockfile a designed happy path: the coder writes a manifest
  from a document fixpoint did not author, the gate then resolves and installs
  what that manifest names — up to `max_tasks × max_task_attempts` times in one
  run — and fixpoint commits the resulting lockfile into the delivered project.
  Nothing reviews that code inside this run: §9 is explicit that `review-code`
  comes afterwards. One hallucinated or typosquatted dependency is therefore
  code running as the operator, on the host, during the gate.

  The stance, rather than only the disclosure: the shipped node stack installs
  with `--ignore-scripts`, so a package's install hooks do not run by default and
  a project that genuinely needs them fails the gate loudly instead of executing
  quietly. And this pipeline is the strongest case in the tool for the
  container/VM operation `docs/security.md` already recommends for untrusted
  agent CLIs — it is the only pipeline that runs model-chosen third-party code.
- Inherited-but-inert loop keys (`max_iterations`, `clean_rounds_to_stop`,
  `max_findings_per_round`, `max_final_passes`, `final_skip_run_edits`) are printed
  by `--check` as ignored. Refusing them is impossible for the same `extends`
  reason; saying nothing would let an operator tune a dial that does nothing.

### 7.4 Flags: three about the plan, one about the tree

**None of the four flags in this subsection is implemented in this version.**
The binary defines `-config`, `-target`, `-out`, `-trusted-target`,
`-trusted-bundle`, `-allow-untrusted-fix` and the loop flags, and nothing else;
`-plan-only`, `-plan`, `-no-coverage-check` and `-continue` (§5.5) are all
specified here and unbuilt. What that costs today is stated where it lands: the
coverage refusal in `internal/implement/outline.go` tells the operator in the
error text that there is no flag to bypass it, and §8's recovery column names
recoveries the tool can actually perform. This block stays because the rules
below — above all the `design_sha256` refusal — are the part that must not be
re-derived casually when the flags are built.

- **`-plan-only`** — preflight, snapshot, plan, validate; write `plan.json` and a
  rendered `PLAN.md` **into the run's artifacts**; create no directory, claim no
  `-out`, spend no coder session; exit 0.
- **`-plan <file>`** — skip the planner; load and validate an operator-supplied or
  previously-generated plan, against every rule in §4.2. Provenance is checked, and
  the rule is stated rather than implied:
  - The file must carry `provenance.design_sha256`, and it must equal the sha256 of
    the design passed on this invocation. A plan for design X must never be
    implemented against design Y.
  - **A missing or empty `design_sha256` is a refusal, not a waiver.** An
    accept-if-absent rule would make the guard bypassable by deleting one key —
    precisely the key an operator editing `plan.json` by hand is most likely to
    disturb.
  - So that this does not make the hand-written-plan recovery path of §8 unusable,
    the refusal prints what to paste: *"the design at /home/me/DESIGN.md hashes to
    9f2c…; add `"provenance": {"design_sha256": "9f2c…"}` to the plan to assert it
    was written for this design."* The same digest is printed by preflight on every
    run and by `--check`. Asserting the pairing costs one line; omitting it never
    silently succeeds.
  - Every *execution* provenance field is re-injected by fixpoint for this run
    (run id, coder, verify profile). The **planner field is never forged**: a
    plan supplied via `-plan` is stamped `planner: operator-supplied (-plan)`
    plus the file's sha256 — recording the configured planner agent as the
    author of a plan it never saw would make the audit trail state a falsehood
    (review run 20260813-003817). Only the design hash is an assertion the file
    is allowed to carry, because only that one is a claim about the file itself.
- **`-no-coverage-check`** — proceed with a design whose outline the coverage rule
  cannot read (§4.2 rule 6), with the run's every report saying `coverage:
  unchecked`.
- **`-continue <project>`** — resume a project fixpoint built (§5.5). Mutually
  exclusive with `-out`, `-plan` and `-plan-only`: the plan and the design come from
  the repository.

Because planning precedes scaffolding, these compose:

```sh
fixpoint implement-go -target DESIGN.md -plan-only
$EDITOR .fixpoint/<ts>/plan.json
fixpoint implement-go -target DESIGN.md -out ~/src/prsi -plan .fixpoint/<ts>/plan.json -trusted-target
fixpoint implement-go -continue ~/src/prsi -trusted-target     # if the deadline ran out
```

This is the cheapest human-in-the-loop at the highest-leverage point in the run: a
bad decomposition wrecks forty sessions, and reading twenty lines of markdown costs
a minute. It also composes with the previous increment —
`-plan-only` → `review-design -target PLAN.md` → `-plan` — for an operator who wants
the panel on the plan after all.

## 8. Failure and operation

| what breaks | when | what the user sees | what an operator can observe | recovery |
|---|---|---|---|---|
| `-out` exists, or is nested in any git repo or a `.fixpoint/` | preflight | refusal naming the enclosing repo | — | choose another path |
| no gate configured and `policy` not `off` | preflight | refusal listing the shipped stack configs | — | pick a stack, or assert `off` |
| trust flag absent | preflight | the §7.3 refusal | — | pass `-trusted-target` |
| design exceeds the planner's budget | preflight | refusal naming bytes and budget | — | split the design, or raise the budget |
| design has no readable outline | preflight | refusal naming the heading count | — | add sections, or pass `-no-coverage-check` |
| planner session fails | after 1 session | run fails, exit 1, no directory created | raw reply in `round-0/` | rerun; the design and the write-target are untouched (`-plan` is specified but not shipped, §7.4) |
| plan invalid (cycle, 200 tasks, absolute path, uncovered section) | after 1 session | refusal naming the offending task and rule | `plan.json` at the run root holds what was returned verbatim | edit the design (or raise the limit the refusal names) and rerun; hand-correcting the plan needs `-plan`, which is not shipped (§7.4) |
| plan cannot fit the deadline | after 1 session | refusal naming tasks, per-task budget, and `max_run_duration` | the plan | raise the deadline, or cut the plan |
| `-plan` file lacks or mismatches `design_sha256` | preflight | refusal quoting the design's actual digest | — | paste the digest, or point at the right design |
| `-out` appeared between preflight and `os.Mkdir` | scaffold | refusal; the planner session is already spent and its plan is at the run root | `plan.json` | name a different `-out` and rerun -- the planner session is paid again (`-plan` would avoid it, §7.4) |
| task 1 does not leave the project building | task 1's gate | task 1 fails after its attempts; everything depends on it, so everything is skipped; exit 2 | verify output per attempt; summary names the failing check | fix the plan's first task, rerun into a fresh directory |
| a mid-run task fails the gate on every attempt | that task | task failed, changes stashed, dependents skipped, run continues, exit 2 | scoreboard and summary name the task, the check, and each skipped dependent | rerun that task by hand, or hand the project to `fix-code` |
| the gate rewrites tracked sources, or the coder's new untracked sources | that task | `gate-mutated-sources`, task failed, paths named | the diff the gate produced | fix the gate config; a formatter belongs in a task, not a gate |
| the gate creates un-ignored output | that task | output excluded from the commit, deleted before it, named in the log with a `gitignore_seed` recommendation | the log line | add the seed entry so the next run stops paying for it |
| the gate rewrites a `gate_generated` file (lockfile) | that task | staged into the commit, attributed to the gate in a trailer | the trailer | none needed — this is the designed path |
| the coder edits a control artifact (`DESIGN.md`, `PLAN.md`, `PLAN.json`, `.gitignore`) | that task | restored from the bootstrap blob, task failed as contract violation | the journal event and the restored file | rerun the task |
| the coder's changes exceed `max_task_bytes` | that task | attempt failed before hashing or gating; tree discarded via checkout-and-clean (§5.3) | the census walk's report and the journal | raise the bound, or fix the plan's task |
| the coder creates a credential-shaped file (`.env`, `id_rsa`, `*.pem`, …) | that task | attempt failed, tree discarded via checkout-and-clean so nothing reaches the object database — not even a stash — paths named | the `credential_shaped_paths` journal event | if the project genuinely needs one, add it after the run: fixpoint will not commit a path its own mandatory exclude patterns would then hide from every reviewer |
| the clean-clone check fails (`clean_check`, §7.2) | end of run (`last`) or that task (`every`) | run incomplete (exit 2): "HEAD does not pass the gate in a clean clone" | the clone's gate output; the per-task ignored-census diffs point at the accumulation | at `last`: rerun with `clean_check: every` to find the culprit task |
| the agent or gate cannot run at all (provider refusal, dead session, missing executable), **or a gate command declared `infra: true` exits non-zero** | that attempt | attempt not counted, no marker; retried with an increasing wait (1m, 5m, 15m, 30m) until `implement.max_infra_tries` is spent, then the circuit breaker stops the run incomplete (exit 2). A 402 skips the ladder: waiting cannot fix payment | the `error_status` in the step record and the `infra_backoff` journal events | wait out the outage; the built tasks stand, the remainder needs a fresh run (no `-continue` yet, §5.5) |
| free disk below `min_free_disk` | preflight or between tasks | refusal / run stops incomplete (exit 2), threshold and free figure named; every unreached task is reported | `df`, the `disk_exhausted` journal event | free space, then a fresh run (no `-continue` yet, §5.5) |
| the coder dies mid-task | that task | changes discarded (tracked **and** untracked); an infrastructure death does not count against the attempts (§5.4), a mid-work crash with output does | `git stash list`, attempt artifacts, `error_status` | the built tasks stand |
| the coder claims a task it did not do | that task | contract violation, task failed | the session artifact and the clean tree | rerun that task |
| the coder committed on its own | that task | soft-reset to base, journal deviation, run continues | the journal event | none needed |
| the coder rewrote history below base | that task | **run stops**, exit 1, repository-invariant failure with expected and actual SHAs | reflog | inspect by hand; fixpoint never resets user-visible history for you |
| the coder touched `.git/config`, hooks, other refs, or created a nested repo **in the un-ignored tree** | that task | **run stops**, exit 1, repository-invariant failure naming what changed | `repostate.json` at the run root: the before, the after, and the named diff | inspect by hand; the committed tasks stand and were gated |
| **the GATE** touched any of the same | that task | **run stops**, exit 1, before anything is staged: a gate runs coder-authored code, and a test that appends to `.git/info/exclude` would hide a source file from the census and the commit | the `gate_mutated_repository` journal event and `repostate.json` | fix the gate config; the committed tasks stand |
| a nested repo appears under an IGNORED path (`node_modules/`, a vendored fixture) | nothing | ignored: only the un-ignored tree can reach a commit as a gitlink | — | none needed — installing a dependency from a git URL is ordinary |
| the tree is dirty at the start of a task | that task | **run stops**, exit 1, repository-invariant failure naming the paths | the paths | a fixpoint bug; report it — the residue is fixpoint's, not the coder's |
| `max_run_duration` reached | between tasks | run stops, exit 2, names the next unbuilt task; every task after it is reported `unreached` | summary | a fresh run for the remainder (no `-continue` yet, §5.5) |
| Ctrl-C | between sessions | run stops; every committed task stands and was gated; **nothing further is committed**; the in-flight attempt is discarded so the tree is left clean | summary written to artifacts | leave it — the repository is consistent as-is; the remainder needs a fresh run (no `-continue` yet, §5.5) |

Three of these deserve their reasons stated.

**Failure degrades loudly and per-task.** A whole-project run is expensive, and one
failed leaf task ("add the sound effects") must not throw away twenty verified
commits. So the run continues — but *never* onto a broken base: a failed task's
dependents are skipped rather than attempted. Proposal B halted the entire run on
any task failure and never used the `depends_on` graph it validated; Critic 1
called that dead metadata implying a capability the design did not have, and both
critics preferred Proposal A's skip propagation. Adopted from A.

**Repository-invariant failures stop everything; task failures do not.** The line
is whether fixpoint can still reason about the tree. A task that fails its gate
leaves a repository whose every commit still passed the gate; a session that moved a
ref, rewrote `.git/config`, or left the tree dirty in a way this design does not
account for leaves a repository where fixpoint's next commit could contain bytes
nothing verified. The first is a result; the second is a lost premise.

**Nothing is committed after a stop request.** Proposal A made an exception for its
final `PLAN.md` rewrite, on a fresh context, arguing that fixpoint's own
bookkeeping is not agent work. Since `PLAN.md` is now immutable and progress lives
in the commit trailers and run artifacts (§4.3), the exception is unnecessary and is
dropped — the fix loop's rule holds without a carve-out.

**Exit codes.** `TermImplemented` → 0; `TermIncomplete` → 2; `TermError` /
`TermInterrupted` → 1; preflight and config refusals keep whatever code the tool
already uses for configuration errors. Proposal B's 0/2/3/4/5/6 allocation is
rejected: Critic 1 reports that 3 is already taken (`all-rejected`), and a
per-pipeline exit taxonomy is a cross-cutting change that does not belong in this
increment. **The implementer must confirm the existing allocation before wiring
this up** — I could not read the source (§0). The distinction that matters is the
one already drawn elsewhere in the tool: "finished with holes" (2) versus "could
not run" (1).

**The scoreboard.** The run table gains an implement shape, exactly as it gained a
create one: the header reads
`implement (a project is built; 18 of 20 task(s) implemented; coverage "##" | unchecked)`
and, where the create summary puts the deliverable path, this one puts the project
path and branch. Below it, one row per task: id, truncated title, outcome, short
SHA, attempts, duration, tokens, and the gate verdict. An ungated run says `ungated`
in every row — which is the point. A continued run marks `carried` rows with the run
that built them. `sum.Deliverable` is the project path.

**Journal events.** `plan_finished{tasks,refused,coverage,coverage_gaps,fit}`,
`task_started{id,title,attempt}`, `task_finished{id,status,notes}`,
`verify_finished` (exists), `gate_artifacts{id,excluded,removed}`,
`task_committed{id,sha}`, `task_skipped{id,blocked_by}`,
`contract_deviation{id,kind}`, `repo_invariant_failed{id,what}`,
`ignored_paths_diff{id,created,modified}`, `infra_failure{id,status,strike}`,
`clean_check_finished{cadence,passed}`.

Run artifacts are created owner-only; environment values are never serialised, and
secret-valued overrides appear only as key names with a redacted digest
(Proposal B). Every published string that an agent authored — task titles, notes,
blocked reasons — passes through flatten + redact + markdown defanging before it
reaches a commit message, `PLAN.md`, `PLAN.json`, the scoreboard or the summary.
Proposal A sanitised only commit messages, which Critic 2 correctly identified as a
hole: the published artifacts are published too.

## 9. Where implement-design ends — including what it cannot cover

Its contract, and nothing more:

> Every task in the plan either has a commit that passed the configured gate, or is
> recorded in the run's report as not built, with a reason.

Completion means *the plan was executed*, not *the design was correctly
implemented*. Nothing reviews code inside this run.

```sh
fixpoint implement-web -target DESIGN.md -out ~/src/prsi -trusted-target
cd ~/src/prsi
fixpoint review-code                        # read it before letting it edit
fixpoint fix-code -trusted-target           # review → fix → verify → commit, to convergence
```

**implement-design makes it exist; fix-code makes it good.** implement-design never
re-enters a task to improve it — that is what the loop after it is for, with its
panel, its rotation, its refutation, its closing coverage round, and its own
convergence rule. `fix-code` handles only concrete `review-code` findings; it does
not rescue tasks that failed the implementation gate.

**The gap this pipeline does not close.** Critic 2 is right that nothing here or
downstream asks "does the code implement the design?" `review-design` reviews a
document; `review-code` finds code defects and will not notice that a whole
requirement was never built. This design narrows the gap at both ends — the
coverage rule (§4.2 rule 6) catches a *plan* that drops a design section, and the
report enumerates every task that did not land — but it does not close it: a task
can be planned, committed, gated, and still not do what the design asked. And the
coverage rule's own floor is conditional: it is a heading match, it proves mention
rather than implementation, and on a design with no usable outline it does not run
at all — in which case every report says so instead of implying otherwise. Proposal
A suggested `review-design -target .` over the built project; that is
review-design's config pointed at something it was not built for, and it is not
adopted. The honest statement is that **`review-implementation` — a panel that
reads a design and a repository and reports requirements the repository does not
satisfy — is the next increment**, and until it exists this pipeline's output
should be read by a human against the design. Saying so is better than shipping a
handoff that quietly loses scope.

## 10. Technology

- **Go, stdlib, and the existing internal packages.** Nothing here needs a
  dependency. The parts that look new — snapshot, clamp, atomic claim, JSON
  envelope, redaction, process-group discipline — are claimed by the proposals to
  exist already (§0).
- **JSON in an envelope**, extracted by the existing mechanism, because a model is
  the producer and the tool already has one contract mechanism with one
  reformat-retry path.
- **`os.Mkdir` for the project claim**, because it fails when the directory exists:
  the check and the claim are one syscall.
- **`git` through the existing `Collector`**, whose commit path already carries the
  trailer-forging, closing-keyword and redaction defences any agent-influenced
  commit message needs. Committer is fixpoint; author is a fixpoint-controlled
  synthetic identity naming the coder, so the trailers distinguish agent-produced
  content from tool-controlled publication. The repository gets an explicit
  fixpoint committer identity rather than inheriting ambient global git config.
- **`init.defaultBranch=main` passed explicitly on `git init`**, because the
  operator's global setting is not this tool's business to guess.
- **Hooks disabled repo-locally and re-asserted per invocation**: `core.hooksPath`
  points at the empty `.git/fixpoint-hooks/` inside the repository itself (§5.1),
  every fixpoint git call passes `-c core.hooksPath=.git/fixpoint-hooks`, and
  commits use `--no-verify`. Belt and braces on purpose: the config is the default
  posture, the per-invocation flag is what survives a session that edited the
  config, and §5.2 step 4 is what notices it did.
- **Census by digest, not by path set** (`git status --porcelain` for the path
  classification, sha256 for content), because on a greenfield project the files
  that matter are untracked and a path set cannot see a rewrite. The **ignored**
  tree is censused by stat (size + mtime), not digest — `node_modules/` makes
  hashing it unaffordable, and creation/modification detection is all §5.2 step 6
  asks of it.
- **The index is rebuilt, never inherited**: reset to HEAD, then stage fixpoint's
  own computed pathspec.
- **Gate commands executed by the existing `verify` executor**, with its existing
  timeout and process-group handling. Proposal B's argv-array-no-shell rule is
  right in principle but is a change to `verify`'s configuration surface shared with
  every other pipeline; if `verify` today takes shell strings, this increment uses
  them unchanged and does not fork the schema (Critic 1's drift argument).
- **sha256 from stdlib** for the design fingerprint, in trailers, `PLAN.md` and
  `PLAN.json`.
- **No database and no new state format.** The run's facts are an append-only
  journal flushed as it happens, plus `status.json` rewritten atomically after each
  task — and, for continuation, the git history itself: `PLAN.json` in the bootstrap
  commit plus the `Fixpoint-Task`/`Fixpoint-Outcome` trailers of code commits and
  outcome markers (§5.4) are the entire resume state, so a killed run leaves an
  accurate record of what it had done — and what it decided without committing —
  in the one place that cannot drift from the tree it describes.

## 11. Decisions and dissent

### Taken from Proposal A (the coherent core)

The read/write target split; snapshot-then-read; the JSON plan with backward-only
dependencies and deterministic pre-flight validation; serial execution; per-task
failure with transitive dependent-skipping; `must_pass` with no baseline and the
refusal of `no_regressions` on an empty tree; empty gate commands refused unless
`policy: off` is asserted; reuse of the existing `verify` block and `-trusted-target`;
`extends`-based stack configs; the read-only `planner` role held to the same
`can_edit: false` rule as the judge and editor; exit 0 / 2 / 1; the scoreboard and
journal shape; `-plan-only` and `-plan`; the coder reading `DESIGN.md` unfenced with
that posture stated rather than hidden; refusal of parallel task sessions and of
run-level squashing.

Proposal A is the core because it is the only one fitted to this codebase: it
reuses the verify gate, the collector, the envelope extractor, the snapshot, the
trust flag and the exit-code vocabulary, rather than proposing parallel versions of
each. Both critics converged on its plan-validation and dependency-skip model.

### Grafted from Proposal B

- **Coordinator-injected provenance** after parsing, so no model can author or
  overwrite run id, design hash, agent labels or verify-profile digest; emitted as
  trailers on the bootstrap commit and every task commit.
- **The bootstrap commit** with a byte-exact `DESIGN.md` and no banner — altering
  the bytes would break the hash match with what review-design approved.
- **Plan fields are prompt data, never executable**, stated explicitly so a later
  change cannot start running acceptance criteria.
- **`Verification: skipped (no operator gates configured)`**, never "passed", for
  ungated runs.
- **Gate-mutation as a distinct failure class**, and the census, so tool-written
  output never enters a coder-attributed commit.
- **Retry from a clean candidate**, carrying the diagnostic but not the prior
  changes.
- **The committed plan is never edited to record progress.**
- **Config discovery may not read the design's directory or `-out`.**
- **Plan before scaffold**, which makes `-plan-only` coherent.
- **Owner-only artifacts and key-name-only secret recording.**

### Rejected from Proposal B, with reasons

- **`--trust-config`.** This repository already decided a create run needs no trust
  flag (f424d9f). A flag every real invocation must carry stops being a signal.
  (Critic 1.)
- **A separate `implement_design.verification` block.** Forks the gate config;
  operators define build and test twice and the definitions drift, so a project can
  pass this gate and fail `fix-code`'s on the identical commit. (Critic 1.)
- **Exit codes 3/4/5/6.** Collide with the existing allocation, and a per-pipeline
  exit taxonomy is a cross-cutting change that does not belong here. (Critic 1;
  unverified — §0, §8.)
- **Halting the whole run on any task failure**, and validating `depends_on`
  without ever using it. Dead metadata implying capability the design lacked; the
  skip-dependents model uses the same field for the thing it was validated for.
- **Empty diff as a fatal error.** Turns an expected outcome — a planner
  over-decomposing — into a dead run.
- **The replan sub-command and its tool metadata commit.** It violates B's own rule
  against tool-authored diffs between coder tasks, and gating a plan-JSON edit
  spends up to a full gate timeout proving something a JSON edit cannot break.
  Re-planning is `-plan-only` plus a new run into a fresh directory. (Critic 1.)
- **The plan committed under `.fixpoint/` in the produced project.** That path is
  fixpoint's scratch and commit-exclusion root; a later `fix-code` run would write
  around it, exclude it, and could clean it away. (Critic 1, who flagged that they
  could not confirm the prefix handling; neither could I — the argument stands on
  the convention either way, and moving the files to the project root costs
  nothing.)
- **Candidate worktrees on private refs with compare-and-swap fast-forward.** This
  is the closest call in the document, and Critic 1 endorsed it. Rejected because
  the run holds an exclusive lock on a directory it created moments earlier, so
  there is no concurrent writer for CAS to defend against; the failure mode that
  *is* real — the coder moving HEAD or the repository's metadata — is caught by the
  invariant ladder in §5.2 step 4. Worktrees would multiply the trees in a design
  whose central move is *reducing* the tree count to two. What CAS bought that the
  SHA check does not is crash-resume of a half-finished acceptance; `-continue`
  (§5.5) now recovers everything up to the last *commit*, which leaves only the
  window between gate-pass and commit — one task's work — as the residual loss.
  That narrowing is what makes this rejection comfortable rather than close.

### Rejected from Proposal A, with reasons

- **Rewriting and re-committing `PLAN.md` at the end of the run.** Critic 2 showed
  it leaves HEAD in a state the gate never saw, and it inserts a tool-authored
  commit into a history whose entire value is one-task-one-revert. Progress lives in
  the commit trailers and run artifacts; `PLAN.md` and `PLAN.json` are written once
  and are immutable. This also removes Proposal A's need to commit after a stop
  request.
- **Unchecked `already_satisfied`.** Critic 2 is right that an ungated run could
  exit 0 with functionality absent. It now requires corroboration against earlier
  committed tasks, and an aggregate ceiling.
- **`target.mode: directory`.** Critic 2 caught the contradiction with a markdown-file
  read-target. The read-target here is a file; if `target.mode` governs it, the
  constraint is `file` — and the implementer must check which meaning the key
  actually has before writing either.
- **Sanitising only commit messages.** `PLAN.md`, `PLAN.json`, the scoreboard and
  the summary are published artifacts too; agent-authored strings pass the same
  flatten + redact + defang path into all of them. (Critic 2.)
- **Scaffold-then-plan ordering**, which made `-plan-only` unresumable. (Critic 2.)
- **"One in-session correction and no second session."** Proposal A declined a
  second session for want of measurement; Critic 1 showed the opposite extreme
  makes unattended runs impossible. `max_task_attempts: 2` is the bounded middle,
  and `1` restores A's posture.

### Decisions nobody proposed

- **The coverage rule (§4.2 rule 6).** fixpoint extracts the design's outline
  itself and requires the plan to account for each heading, by task or by an
  explicit out-of-scope reason. Scrutiny it deserves: it is cheap (a heading scan),
  it is deterministic, it cannot be gamed by the planner because the headings come
  from the snapshot and not from the model, and it fails at plan time — before any
  coder session. Two limits are real and stated in the rule itself: it proves a
  heading was *mentioned*, not that it was *implemented*; and it depends on the
  document having an outline at all, which is why an unusable outline is a refusal
  and `-no-coverage-check` stamps `unchecked` on every report rather than letting a
  clean-looking summary imply a check that never ran.
- **The fit rule (§4.2 rule 7).** The deadline is checked against the plan before
  any coder session, so an impossible run is a refusal costing one planner session
  rather than a truncation costing sixteen hours.
- **The repository-invariant ladder** — HEAD equal, HEAD descendant (soft-reset and
  continue, journaled), anything else about HEAD, and separately `.git/config`,
  hooks, refs, nested repos and the clean-tree precondition (stop the run).
  Proposal A did not consider a coder that commits; Proposal B rejected the attempt
  outright; neither considered a coder that writes `.git`.
- **The census by digest, and the removal of un-ignored gate output.** Excluding a
  file from one commit is not the same as it not being there; the removal is what
  makes the exclusion hold for more than one task.
- **`-continue`**, and the `PLAN.json` in the bootstrap commit that makes it
  possible. The repository fixpoint builds now carries everything needed to finish
  building it.
- **`max_vacuous_frac`, `max_run_duration`, `gitignore_seed`.** The numbers are
  guesses, they are config keys for that reason, and — with rule 7 — they refuse
  loudly rather than truncating silently.

### Objections from review round 0, and what changed

Every objection in round 0 found a real defect and every one is amended. Recorded
here because three of them were amended more narrowly than the objection implied,
and the narrowing is a decision, not an oversight:

1. **Gate output is not sticky (objections 1 and 8).** Amended: un-ignored gate
   output is now **removed** after the commit, not merely excluded (§5.2 step 7),
   `gitignore_seed` and a task-1 prompt rule keep the normal case from producing
   any, and a clean-tree precondition (§5.2 step 0) turns a leak into a loud stop
   rather than a misattributed commit two tasks later. Ignored paths are exempt
   throughout: deleting `node_modules/` between tasks would be a cure worse than
   the disease, and an ignored path can never enter a commit or a census anyway.
2. **Deadline versus task count (objection 2).** Amended twice: §4.2 rule 7 refuses
   an infeasible plan at plan time, and §5.5's `-continue` makes an expired run
   resumable from the commit trailers instead of discarding every gated commit.
   The narrowing: `-continue` is not a general resume. It demands the same plan,
   design, gate, a clean tree and a prefix history, and it re-runs first-contact
   hardening on a tree that has been outside fixpoint's lock. A looser resume would
   re-open the trust argument §2 depends on for a tree fixpoint no longer knows the
   provenance of.
3. **Untracked-inclusive discard (objection 3).** Amended: §5.3 specifies the
   discard covers untracked files, and — because §0 cannot confirm what
   `StashDirty` does today — pairs it with a clean-tree assertion that stops the
   run if the discard was incomplete. The assertion is the load-bearing half: it
   converts an unverifiable reuse hypothesis into a loud failure instead of a
   silently mixed commit.
4. **`design_sha256` (objection 4).** Amended: the provenance block is now in the
   schema (§4.1), fixpoint-owned, stripped if the planner emits it, and §7.4 states
   the answer the draft left open — **missing is a refusal**, with the digest
   printed so a hand-written plan costs one pasted line.
5. **Nesting check scoped to the wrong repository (objection 5).** Amended: the
   check now walks `-out`'s parent chain for any `.git`, independent of the
   read-target and of the invocation's cwd. No escape hatch in v1; a legitimate
   monorepo subproject is refused, which is friction I would rather have than the
   silent gitlink. §12 records the escape hatch as an open question.
6. **Coverage rule as a conditional no-op (objection 6).** Amended: the outline
   level is derived from the document rather than assumed to be `##`, a document
   with no usable outline is a **preflight refusal**, and a run with the check
   disabled says `unchecked` in `PLAN.md`, the scoreboard, the summary and the
   journal. The false assurance the objection identified — a report asserting a
   check that never ran — is the specific thing fixed.
7. **Gate mutating the coder's untracked sources (objection 7).** Amended: the
   census records content digests, and `gate-mutated-sources` now covers modified or
   deleted untracked files that existed before the gate, not only tracked ones. The
   tracked/untracked distinction survives only where it was actually load-bearing —
   *new* files created by the gate are output, not mutation.
8. **Construction-time trust (objection 9).** Amended, and §2 rewritten rather than
   patched: hooks are disabled at init and re-asserted on every invocation, the
   repository's config, hooks, refs and nested-repo status are re-verified after
   every session, and the index is rebuilt from fixpoint's own census before every
   commit. The claim that the first-contact hardening "drops out" is now scoped to
   what actually drops out — hardening a tree fixpoint did not make — and the rest
   is relocated, not deleted.

### Findings from review round 1 (run 20260812-183115), and what changed

The first review with repository access: 44 findings, 41 after merge, 31 kept by
the judge, of which 18 high. All 18 are folded into this revision — ten decisions,
recorded here because several reshape mechanisms rather than patch sentences:

1. **`core.hooksPath` moved inside the repository** (`.git/fixpoint-hooks/`,
   §5.1). The delivered repo no longer references the transient artifact tree —
   the review's most corroborated finding (both agents, three lenses).
2. **The outcome marker commit** (§5.4). Trailer-only resume state could not
   represent `already_satisfied`/`failed`/`blocked`, so `-continue` refused
   exactly the runs it exists to recover — four independent findings. Every
   processed task now leaves one commit; `skipped` is derived, not recorded.
3. **`implement.gate_generated`** (§5.2 step 7). The two-bucket classification
   got lockfiles exactly wrong: created → deleted forever, updated → task failed.
   A third, config-owned category commits them with gate attribution.
4. **`.gitignore` is fixpoint-owned and the metadata invariant grew**
   (§4.3, §5.1). The agent-authored exemption set governed fixpoint's own census;
   and `.git/info/exclude` could hide a source from `git status` while the gate
   read it from the tree.
5. **Control artifacts protected mechanically** (§4.3, §5.2 steps 5 and 8).
   "Immutable" was declared, not enforced; now: reserved in validation, blob-hash
   checked per session, never staged, read from the bootstrap commit on
   `-continue`.
6. **Ignored paths join the transaction** (§5.2 steps 2 and 6, §5.3). Stat
   census; session-created ignored paths deleted before the gate and on discard;
   session-modified ones were made a `hidden-state` task failure — a gate must
   never pass on bytes HEAD does not contain. (Round 2 killed the `hidden-state`
   half and replaced it with the clean-clone check — see the round-2 record.)
7. **The in-session correction pass is removed** (§5.2 step 7, §5.3). Under a
   one-shot agent interface it was a second session inside one attempt — breaking
   one-session-one-commit and contaminating its own census. An attempt is one
   session, one gate; gate output is removed after every gate run, not only after
   success.
8. **The fit rule uses the worst case the config permits** (§4.2 rule 7), and
   `max_run_duration` defaults to 32h so the advertised 8–25 task range fits at
   `max_task_attempts: 2`.
9. **Byte and disk bounds** (`max_task_bytes`, `min_free_disk`, §5.2 step 6, §8).
   Coder output was the one resource with no ceiling.
10. **No `implement-design.yaml` base config** (§7.1). The loader's one-level
    `extends` is deliberate and verified; the stack configs extend `defaults`
    directly and the shared numbers live in code. The one reuse hypothesis whose
    failure invalidated a section, exactly as §0 warned.

Two findings the judge rejected are recorded as rejected for the implementer's
benefit: the between-tasks deadline check stands (per-session and per-gate
timeouts already bound a hung task; a journal heartbeat is a nice-to-have), and
exit 2 continues to mean both "finished with holes" and "deadline expired" (the
scoreboard distinguishes them; a per-pipeline exit taxonomy stays out of scope).

### Findings from review round 2 (run 20260813-003817), and what changed

The first four-reviewer round (claude, codex, deepseek, kimi — the restored
pool): 73 issues, 51 kept by the judge, 30 high. None of revision 2's folds
reappeared; the round went one level deeper, into the mechanisms revision 2
introduced. The mechanism-level findings are folded in this revision:

1. **The `hidden-state` failure is gone** (§5.2 steps 2 and 6, §7.2). Killed
   twice over: stat metadata cannot carry an integrity claim (a same-size write
   restores its own mtime), and ordinary build churn — the coder running the
   tests, behaviour §6 encourages — modifies caches in place from task 2 onward,
   so the rule failed honest tasks. The property it protected is now enforced
   where it is checkable: **`implement.clean_check`** clones HEAD and runs the
   gate in the clone (default once per run, at the end). The census remains as
   the diagnostic pointer.
2. **Every non-committing attempt exits through the §5.3 discard** — the
   earlier text covered two of six failure paths; the others left residue that
   stopped the run at the next task's step 0. The byte-bound breach discards via
   checkout-and-clean so the ceiling never writes the oversized tree into the
   object database.
3. **Infrastructure failures are not outcomes** (§5.4): no marker, no burned
   attempt, `implement.max_infra_tries` retries with an increasing wait, then
   the circuit breaker. A provider outage — or a registry one, for a gate
   command declared `infra` — is a fact about the morning, not about the plan.
4. **`blocked` needs corroboration** (§5.4, §6): citations of the conflicting
   design sections, confirmed by a second independent session — the same
   treatment `already_satisfied` got, for a bigger blast radius.
5. **The artifact root keeps its hardening and is exempt from the census**
   (§2, §4.3, §5.2 step 2): `.fixpoint/` never moved to the new tree, so
   "fixpoint made it" never applied to it; and on `-continue` it is the run's
   own scratch inside the project, which the census must not read as coder
   activity.
6. **Markers got a versioned schema** (§5.4) and the staged/empty commit
   primitive is declared in scope (§0, §3) — `Collector.Commit` cannot express
   either need today.
7. **The fit formula counts the gate's real worst case** (§4.2 rule 7): the
   verify executor times out per command, so a three-command gate is three
   timeouts, not one.
8. **The active agent set is planner + coder, explicitly** (§7.3) — the
   inherited reviewer pool must never be pinged or billed; the create pipeline
   shipped this exact bug (fixed in f424d9f) and the design now refuses it on
   paper instead of in a postmortem.
9. **Imported plans are never attributed to the configured planner** (§7.4),
   and the **Verify-Profile digest covers the effective gate** — commands,
   policy, timeouts, `gate_generated` — not the command strings alone.
10. **Duplicate outline headings are a preflight refusal** (§4.2 rule 6) —
    heading text is a join key, and a collision collapses two sections into one
    coverable identity.

The round's `-continue` trust cluster (7 findings) is deliberately NOT folded as
mechanism; it is recorded in §12 as the design's known weakest joint, with the
fix direction named and deferred until a live run hits it.

### Where the panel did not converge

1. **Worktrees versus a single tree.** Critic 1 argued the candidate-worktree /
   CAS model is what preserves "git revert undoes one task" and makes acceptance
   idempotent under crash; Critic 2 endorsed Proposal A's single tree with stash.
   I chose the single tree, and `-continue` has since shrunk the loss from "the
   whole run" to "the task in flight". What remains genuinely lost is a crash in the
   window between gate-pass and commit: that task is redone by a fresh session. If
   the first live runs crash in that window more than once, the worktree model is
   the fix and Critic 1 was right.
2. **Whether tasks write tests.** Proposal A argued tests only where the design asks
   or where the gate needs them, citing the cost of each task growing and the
   measured non-convergence of reviewing freshly written tests; the opposite view is
   that a project handed to `review-code` with no tests gets a large findings pile
   that `fix-code` must then write tests to close. Unresolved. This design takes A's
   position by default and leaves it in the prompt, where a live run can move it.
3. **Whether the plan should go through `review-design` as a matter of course.**
   `-plan-only` makes it a two-command workflow. Neither critic addressed it. The
   affordance is built; mandating it is not.
4. **Whether `-out` may point at an existing empty repository** — the one the forge
   just created. Refused in v1: `os.Mkdir` fails, and such a repo usually has a
   README anyway. The friction is real, and "fixpoint made this tree, so fixpoint
   trusts it at the moment it makes it" is doing load-bearing work in §2 that an
   adopted tree would weaken. `-continue` is the one adoption path, and it pays for
   itself by re-hardening the tree and checking the bootstrap trailers before it
   believes anything. A real operator's opinion should settle the forge case.

## 12. Open questions

1. **Every reuse hypothesis in §0.** The implementer's first task is to confirm
   that the named helpers exist with the claimed properties. Four would change a
   decision rather than a cost: if `verify` takes argv arrays rather than shell
   strings, adopt Proposal B's structured form directly; if the existing exit-code
   allocation has no free "incomplete" code, §8 needs renegotiating rather than
   asserting; if `Collector.Commit` does not already redact and defang, that work is
   in scope here rather than assumed; and if the discard helper is tracked-only,
   extending it is in scope here (§5.3), not optional.
2. **The numbers are guesses.** 40 tasks, 12 files per task, 2 attempts, 0.34
   vacuous, 32h, 64 MB per attempt, 2 GB free disk, 8–25 preferred, 32 kB of design
   excerpt. Nothing measured them, because nothing has run. Every one is a config
   key or prompt text, and every one refuses rather than truncates — including the
   deadline, which is checked against the plan's worst case before a coder session
   runs — so a wrong guess surfaces as a refusal at startup rather than as a bad
   project or a truncated run.
3. **One gate, one language.** A design implying a polyglot project — a Go server
   and a React front end, each with its own build — gets one `verify` block. Out of
   scope here; the honest answer is that the gate needs per-task or per-directory
   command sets, which is a change to `verify` belonging in its own increment rather
   than smuggled through this one.
4. **`-out` inside a repository, on purpose.** §7.3 refuses it outright. A monorepo
   operator who genuinely wants the new project as a subdirectory of an existing
   repo has a legitimate need and no flag. `-allow-nested-out`, with the outer
   repo's cleanliness checked and its `.gitignore` consulted, is the shape — deferred
   until someone asks for it, because guessing the semantics of committing into
   someone else's history is how this design would acquire a fifth meaning of
   "target".
5. **`review-implementation`.** §9's gap. Until a panel exists that reads a design
   and a repository together, no automated step in this pipeline asks whether the
   built thing matches the document it came from. The coverage rule is a floor
   against wholesale omission at plan time and is not a substitute.
6. **`-continue -retry-failed`.** Carried `failed`/`blocked` outcomes are final
   (§5.5); a flag that re-opens them against the same plan would need to answer
   what a retry means for a task whose dependents already landed. Deferred until a
   real run produces the need. (Softened by revision 3: infrastructure failures
   never become `failed`, so the finality now covers only results about the
   work.)
7. **The `-continue` trust model is the design's weakest joint.** Review round 2
   (run 20260813-003817) put seven high findings on it, and they share one root:
   continuation authenticates history using only the history itself. Trailers
   are self-asserting text any committer can write; the repository invariants
   are re-baselined as trusted rather than re-verified against a record the
   coder could not have authored; a dirty tree cannot resume — which is exactly
   the crash and reboot case §5.5 advertises; a changed Verify-Profile strands a
   half-built project with no path but a fresh directory; task records are not
   bound to the committed plan's identity. v1 ships `-continue` narrow and says
   so: `-trusted-target` is the operator asserting the tree was not tampered
   with between runs, the same assertion every fix run already makes. If live
   runs hit these limits, the fix direction is a **fixpoint-signed ledger**
   (a signed note ref or trailer HMAC keyed outside the repository) plus a
   `-continue -discard-dirty` that runs the §5.3 discard before admission —
   recorded here so the implementer builds toward it rather than around it.
8. **Implementation checks from round 2**, cheaper to list than to re-derive:
   the two-target split needs an owning component (`target.path` is
   single-valued today; decide whether `-out` stays flag-only); the shipped
   stack configs must be validated against the real `verify` schema; task
   `files` lists are advisory and boundary enforcement is not attempted in v1;
   the census does not bind symlink *content* (a symlink whose target changes
   is invisible to a path-and-digest walk over regular files); and the resource
   bounds cover per-attempt bytes but not cumulative classes (stash growth
   across forty tasks, journal size). Each is either an hour's work or a
   documented non-goal — the wrong outcome is discovering them mid-build.
9. **A machine-verifiable approval artifact from `review-design`.** This design
   records the design's path and hash but never claims the document was approved. If
   `review-design` later emits structured approval metadata, implement-design should
   accept it, preserve it in the bootstrap trailers and in `PLAN.json`'s provenance,
   and be able to refuse a design that carries none.
