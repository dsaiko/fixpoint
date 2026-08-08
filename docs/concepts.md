# What a run actually does

Target modes, the shape of a review round, how observations become issues, and why every fix is its own commit.

[← back to the README](../README.md)

## Target modes

What gets reviewed is controlled by `target.mode`:

| Mode | What is reviewed |
|---|---|
| `directory` | Every file under `target.path` except what `.gitignore` and the `exclude` globs remove. Scope is denylist-only — there is no allowlist option, since one has to be re-derived per language and silently hides whatever it forgets. |
| `git-diff` | Changes relative to `target.base_ref`. The base is resolved to a concrete commit once at run start, so per-round fix commits extend the reviewed diff instead of shrinking it. A trailing `...` (`origin/main...`) pins the **merge base** with HEAD instead of the ref's tip — what this branch added, which is what `fix-branch` uses. Without it, any commit the base branch has and yours does not appears in the diff *reversed*, and the panel reviews someone else's work as deletions you made. Empty `base_ref` reviews unstaged working changes, so it is review-only: a fix run needs a base ref (it starts from a clean tree, which makes the unstaged diff empty) and is rejected at validation without one. |
| `pr` | A GitHub pull request, named by `target.pr` or per run with `-pr`. The PR branch is checked out locally (`gh pr checkout`) and reviewed against its base, so fixes land in the working tree and later rounds review them too. Pushing fixes back is manual. |

Fix rounds require `target.path` to be a git repository with a clean working
tree at run start (in every mode); `review_only` runs anywhere.

### Choosing a base in git-diff mode

`base_ref` is handed straight to git, so anything git resolves works — in the
config, or per run with `-base-ref`:

| `base_ref` | What gets reviewed |
|---|---|
| `@{upstream}...` | This branch against the branch it tracks. The `fix-branch` default. |
| `origin/HEAD...` | This branch against the remote's default branch — no upstream needed and no branch name hard-coded, so it is the best generic choice. Needs the local ref, which `git remote set-head origin -a` creates. |
| `origin/develop...` | An explicitly named trunk. |
| `@{push}...` | Triangular workflows, where you push somewhere other than you fetch from. |
| `HEAD~3` | The last three commits. |
| `v1.4.0` | Everything since a release tag. |
| `ORIG_HEAD` | Whatever the last rebase, merge, or reset moved past. |
| `HEAD@{yesterday}` | What you have done since yesterday. Read from the reflog, so it is per-clone and approximate. |

**When the `...` matters.** Only when the base can hold commits your HEAD does
not. For a **branch name it is essential** — without it you diff against that
branch's tip, and every commit it has that you lack shows up *reversed*, so the
panel spends a round reviewing someone else's work as deletions you made. With
two machines pushing to one repository that is hours, not months. For anything
already behind you — `HEAD~3`, a merged tag, a SHA on your own history —
`merge-base(X, HEAD)` is just `X`, so the dots are a harmless no-op.

Short version: **a branch name takes the dots; anything already in your history
does not.**

Getting this wrong is silent — both spellings resolve, and only the diff says
which one you meant — so `--check` prints the commit the base resolved to and
the size of the diff it selects, before any agent runs:

```
$ fixpoint fix-branch --check --trusted-target
scope: base_ref "@{upstream}..." -> d122ea0dd503; 9 files changed, 313 insertions(+), 12 deletions(-)
configuration OK: 5 review lens(es), coder claude-coder, strategy rotate
```

A base that resolves to nothing fails there rather than after the first round
has been paid for. `--check` reports scope in `directory` mode too (`77 file(s)
in scope`); in `pr` mode it cannot, because finding out would mean running
`gh pr checkout` and switching your branch.

`target.exclude` and the credential patterns fixpoint enforces on top of it apply
in **every** mode: in `git-diff` and `pr` they become git exclude pathspecs, so an
excluded file never contributes its diff or its name to the collected material.
That matters most there, since a diff carries file *content* into every reviewer's
prompt, where a directory listing only carries paths.

In `directory` mode against a git repository, scope comes from
`git ls-files --cached --others --exclude-standard` — tracked files plus
untracked ones the project doesn't ignore. So build output, caches, local
binaries and coverage profiles drop out per project without a single config
entry, because the project already declared them in `.gitignore`. A non-git
target falls back to a filesystem walk, where only `target.exclude` narrows
scope. What belongs in `target.exclude` is therefore just what's *committed*
but not worth reviewing — vendored dependencies, fixtures, and (defensively) a
committed `.env`.

An entry with no wildcard at all names a *directory* as well as a file:
`config/secrets` — or `config/secrets/`, the two are the same entry — excludes
everything beneath it, the way a git pathspec prefix does. One carrying a wildcard
anywhere (`*.env`, `**/node_modules`) describes a file shape and matches names
only; to take the contents of every matching directory too, say so:
`**/node_modules/**`.

Your entries match case-sensitively, as written. The credential patterns fixpoint
enforces on top of them do not: `PRODUCTION.ENV` and `ID_RSA` are dropped the
same way their lowercase spellings are.

## What a review run does

A `review-` config never invokes the coder, so it cannot modify the target — and
it no longer even names one. The pipeline is:

```
  panel  →  merge  →  refutation  →  judge  →  verdict  →  body  →  (post)
```

**Panel.** Every reviewer runs every lens (`strategy: all`). A review has one
round and nothing after it, so a lens seen by a single model is a blind spot
nothing catches.

**Refutation.** Every reviewer is shown the *merged* finding set and must take an
evidenced position on each one: maintain, refute, or unsure. What all of them
refute is dropped; what any of them still stands behind is kept and marked
contested.

"All of them" is counted over the reviewers that **answered**, one position per
reviewer per finding, and a finding is dropped only when every one of them refuted
it. Counting the positions that happened to arrive instead would make a single
refuter enough whenever the others simply omitted that id — the likely failure,
since the contract asks for a position on all of them and nothing enforces
coverage. A partial refutation marks the finding contested rather than deleting it:
silence is not agreement.

This replaces the obvious idea — keep only what two reviewers both found — which
measurement ruled out. Across 19 runs, **under 4% of findings were reported by
more than one reviewer, and none of the 25 later rejected as false positives were
among them**: an intersection would have filtered none of the noise while
discarding 237 of 248 confirmed defects. Reviewers here do not vote on a shared
list, they sample from a large space of defensible observations, so each finding
is judged on its own evidence instead.

Unanimous refutation, not majority, because the errors are not symmetric: a wrong
refutation deletes a real defect and nothing downstream looks for it again, while
a wrongly-kept finding costs a human a paragraph. Unanimity is measured over every
reviewer the round asked, not over the ones that answered: a refuter that fails its
contract casts no vote, and cannot thereby leave the one that did answer alone with
a deletion.

Only findings at or above `review.refute_at` (default `high`, the same floor as
`review.block_at`) are put to the round. It costs a full extra pass per reviewer
per round, and as a filter it has never fired: over the two measured runs that put
everything to it — 97 findings — 29 came back contested and **zero** were dropped
unanimously. What it does earn is the judge gate below. That gate covers blocking
findings only, so those are what the round is now asked about; below the floor the
judge already decides alone and a second opinion changes nothing it may do. Set
`refute_at: low` to refute everything again.

The round is not merely redundant, which is why it is scoped rather than removed:
contested findings went on to be dropped by the judge at 55% against 26% for the
rest (Fisher two-sided p = 0.010), and the judge never sees the contested flag —
the canonical list it reads carries only id, severity, location, title and
description. Two independent judgments agreeing is worth something; it is just not
worth a pass on findings the gate does not protect.

**Judge.** One read-only agent decides which survivors are worth a human's
attention — the value judgment the coder makes in a fix run, in hands that cannot
edit. It became necessary rather than optional when the verdict gained a hard
severity gate: if one `high` blocks a merge, something must filter severity before
the gate, or the noisiest reviewer decides the outcome. It fails closed — a judge
that dies leaves every finding standing and blocks the approval — and it cannot
delete a blocker alone: dropping a finding at or above the blocking severity is
honoured only where the refutation round already doubted it, so removing what would
block a merge takes two independent agents rather than the one that could be talked
around by the code it is reading.

*Independent* is enforced by name, not by count. A panel commonly includes the
agent that judges — the shipped pr configuration does — and doubt recorded by that
same agent is one model reached twice by the same injected text, not two agents
agreeing. So the refutation round records **which** reviewers doubted each finding,
and the judge's drop of a blocker is honoured only when at least one of them is
somebody else.

**Verdict.** Computed in code, never asked of a model. See
[config/README.md](../config/README.md#what-a-review-run-concludes) for the rules,
the quorum, and why the floor is `high`.

**Body.** `review-body.md` in the run directory: verdict first, blocking findings
separated from the rest, signed. Agent-authored text is neutralized for a forge
*at render time*, so the file you read is byte-for-byte what gets published —
`-post-run` sends those bytes off disk, and the anchors that run computed with
them.

**A machine reply is marked.** Every answer this tool posts into a conversation
ends with an invisible HTML comment carrying the run id, so a later run can tell
its own answer from a person's — replies go out under the operator's account, and
author identity cannot make that distinction. A conversation whose last word is
ours is skipped as answered; when somebody writes back it is live again. See
[config/README.md](../config/README.md#letting-the-comments-commission-work).

**The signature does not name fixpoint.** It reads *"Reviewed by AI panel · agent,
agent · run <id>"*, and a reply posted into a conversation reads *"Answered by AI
panel"* — a reply is not a review, and the agent it names is the one coder that
wrote it. Reviews land in other people's repositories, where the tool's own name
means nothing to the reader and reads as an unexplained internal string; that a
machine wrote it is the fact that decides how much weight the comment deserves.
Both templates are settable (`review.signature`, `review.reply_signature`), both
are rendered by fixpoint from fixpoint's own facts, and both are placed outside
every region carrying agent text — a signature composed from a finding's prose
could be forged by whatever wrote that prose.

### Several fixes at once

`loop.parallel_fixes` runs N coder sessions concurrently, each in its own git
worktree of the round's base, and applies what they produce **serially** through
the same gate as always — one patch, one verify, one commit. The sessions are the
part worth parallelizing (70% of a measured run's wall clock, nearly all of it
spent waiting on a provider); the gate is the part that must not be, because two
fixes that pass alone can fail together and a serial gate is what names the one
that broke the build. See
[config/README.md](../config/README.md#running-several-coder-sessions-at-once).

## Observations and issues

A reviewer's report is an **observation**. What the coder works from is an
**issue**: one distinct problem, with every observation that reported it attached.

The distinction is not bookkeeping. When a finding was simultaneously a reviewer's
report, the unit of work, and the thing tracked across rounds, two agents reporting
the same problem produced two findings — and so two units of work. Agreement between
reviewers therefore *cost* more, rather than telling the run more. In one real run
two lenses reported a single racy-ordinal defect at the same file and line, one
calling it `concurrency` and the other `tests`, and it was scheduled twice — twice
against `loop.max_findings_per_round`, which was 8 at the time, and now twice as a
coder session.

Now the panel's agreement is surfaced as corroboration — to the coder ("reported
independently by 2 agents") and in the run summary — and is one issue, one session,
one commit.

Grouping works in two stages, because matching within a round and matching across
rounds are different problems:

- **Fingerprint** — normalized path plus the exact line, or a normalized title when
  no line is given. A shared location is where two reports *may* be about one
  defect; agreeing titles are what say they are, so reports in one file merge at any
  distance when their titles agree and stay apart when they do not — even on the
  same line, since one statement routinely holds two defects and one issue carries
  one verdict. Merging them would tell the coder to fix one thing when there are
  two, which is worse than leaving a duplicate that merely costs a slot. The
  category is deliberately **not** part of identity — the real duplicate above
  arrived under two different ones.
- **Across rounds** — a reviewer may set `"issue": "<id>"` on a report to declare it
  is the same problem as an entry in the history it was shown. No lexical rule gets
  from *"Severity vocabulary has two independent declarations"* to *"duplicated
  severity vocabulary"* while the line number moves underneath it, so the reviewer
  that can see both is asked. A nonexistent id falls back to the fingerprint.

An issue carries its own status across rounds, and a re-report means different
things depending on how it was closed. A **rejected** issue stays rejected and is
not handed back — re-submitting a decided question would spend a slot every round.
A **fixed** issue **reopens**: reviewers still seeing it is evidence the fix did not
work, and treating it as closed would let a failed fix end the run as converged.

### Rejection is also a value judgment

The shipped `fix` prompt authorizes the coder to reject a finding that is *correct
but not worth making*, not only one that is wrong. Because rejection is durable,
this is the loop's one damper: a fix costs a commit, a verify run, and whatever the
next round writes about the code it added, so a change that buys less than that is
a net loss. The prompt names the recurring shapes — a finding that restates a
decision the code already documents, a request for a test of a test, a speculative
defect with no input that triggers it, a refactor riskier than the edge case it
removes.

Two limits keep it from becoming a way out of work, and both are in the prompt: a
`high` or `critical` finding may be rejected only for being *wrong*, never for
being expensive, and every rejection reason is recorded in the run summary for a
human to read. Rejecting on value is a judgment the operator can audit afterwards,
which is the only reason it is safe to delegate.

## One fix, one commit

The coder is handed **exactly one issue per session**, under every commit policy.
`loop.commit_policy` then decides how those commits are grouped:

| Policy | Result |
|---|---|
| `per_fix` *(default)* | One commit per issue, each verified on its own. |
| `per_round` | Each round's commits squashed into one. |
| `per_run` | The whole run squashed into one, after the closing round. |

One issue per session is not a policy choice, because nothing downstream can undo
batching. A session handed eight issues edits files for all eight at once, and its
report says only *id, verdict, detail* — nothing that attributes a change to an
issue. So a commit built from that session cannot honestly claim to contain one fix,
however the commits are later grouped. Working one at a time buys three things a
batch cannot:

- **The verify gate names the fix that broke the build**, not the round. A round of
  eight fixes that fails the gate is discarded entirely today; the same eight as
  eight gated fixes lose only the one that failed.
- **`git revert` undoes a single bad fix**, and `git bisect` lands on one change.
- **A rejection is checkable.** A session that rejected its one issue should have
  left the tree untouched, and a session that claimed a fix should have changed
  something. With one issue per session both are verifiable claims rather than
  aggregate ones — a fabricated fix now fails the run instead of hiding behind seven
  real ones.

The cost is that the verify gate runs once per fix rather than once per round, so a
round of eight fixes runs the project's checks eight times. That is the price of
knowing which fix broke them, and it buys back the eight-fix round that used to be
discarded whole. Each pass runs every configured command rather than stopping at the
first failure: the coder gets one bounded correction attempt, so it has to see every
failure at once — fixing the formatter and only then discovering on the retry that
the test suite is red too would waste the round. Ordering `verify.commands`
cheapest-first therefore saves no time, but it still reads best: the log summary and
the failure block handed to the coder follow the configured order.

Squashing re-commits the index as it stands, so a squashed commit's content is
byte-for-byte what the per-fix commits already verified one at a time. The
replacement commit is built before the branch moves, so a squash that fails leaves
the per-fix commits exactly where they were. It only ever removes
information, which is why `per_fix` is the default. A failed or interrupted run is
never squashed: its per-fix commits are how you see how far it got.

`commit_message` is per-fix shaped by default (`{issue}`, `{title}`). A squashing
policy renders a round-shaped header instead (`{round}`, `{fixed}`, `{rejected}`)
with a body listing every fixed and rejected issue; set `commit_message` to a
round-shaped template to control that wording yourself.

### Why there is no per-round cap by default

`loop.max_findings_per_round` used to default to 8. That number was one coder
*session's* capacity — 17 issues blew the 30m timeout, ~8 fit — and with one issue
per session it no longer bounds anything real, so it now defaults to `0`
(unlimited).

It was never free. In one five-round run a cap of 8 deferred 18 issues, and in the
closing round — where no later round follows — it dropped 5 coverage gaps outright.
Set it only as a **cost lever**, to bound what a single round spends, knowing the
overflow waits for a later round (worst severity first, and every deferral promotes
an issue one tier so the tail cannot be starved). Note that `commit_policy` is not
a way to get the old behavior back: `per_round` groups commits, it does not batch
work.
