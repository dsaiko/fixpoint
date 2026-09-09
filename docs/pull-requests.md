# Pull requests

How fixpoint reviews a pull request, what publishing a review does under your
identity, and every refusal that stands between a run and somebody's branch.

Start with the [README's pull-request examples](../README.md#3-review-a-pull-request);
this is the reference behind them.

## Reviewing a pull request

```sh
# 1. See what it would review, and what it would cost, before spending anything.
./fixpoint review-pr -pr 170 --check

# 2. Review it. The result is written to .fixpoint/<run>/review-body.md and
#    nothing leaves this machine.
./fixpoint review-pr -pr 170

#    On the pull request's own branch, drop the number entirely:
./fixpoint review-pr

# 3. Read that file. Then publish THOSE bytes as a comment -- no approval, no
#    block, no second review, and no agent invoked:
./fixpoint -post-run .fixpoint/<run-timestamp>

# 4. Or let the published review carry its verdict, approving or requesting
#    changes:
./fixpoint -post-run .fixpoint/<run-timestamp> -post-verdict
```

### Which pull request

`-pr 170` says which one. Without it, and with no number in the config either,
fixpoint takes **the pull request of the branch that is checked out** — the answer
`gh pr view` gives, which is the same branch-to-PR resolution `gh pr checkout`
acts on, so the number and the tree the run reviews cannot disagree. The resolved
number is logged with its base and URL: nobody typed it, so that line is the run's
only record of what it decided to review, and `--check` prints it as the scope.

An explicit `-pr` wins, and so does a config that names a real pull request. The
resolution fills the placeholder zero, it does not override a choice.

It refuses rather than guess:

- **No pull request for the branch.** Open one, or pass the number.
- **More than one.** A head branch can have several open pull requests, differing
  only in base, and `gh` picks one of them silently. Both numbers are in the
  refusal. A same-named branch on *another* fork is not ambiguity — the head owner
  tells them apart, which is what makes this work on a fork's branch.
- **A listing that cannot prove uniqueness**: one that comes back empty, that does
  not contain the pull request `gh` just named, or that hit the 50-row limit before
  the head-owner filter could run. "Could not tell" is not "unique".
- **A merged or closed pull request.** `gh` answers with one when that is all the
  branch has. Reviewing merged work by inference is never what the command meant,
  and a fix run would commit onto it. The number is in the refusal, so `-pr 170`
  still gets you there.
- **A detached HEAD**, which names no branch at all.
- **A branch that moved while the answer was being computed** — a concurrent
  fixpoint run, or you switching branches by hand.
- **A pull request from a fork**, whatever the number came from. See below.

The resolution happens **after the target-integrity preflight**, not while the
flags are being read, and it asks for those gates itself rather than trusting its
caller to have run them. It executes `git` and an authenticated `gh` inside the
target, and this is the one pr-mode path where the pull request's content is in
the tree *before* fixpoint starts: you are standing on its branch. So the guards
that refuse a `git`/`gh` resolved from inside the checkout, or a redirected work
tree, run first — `review-pr` asserts only `-trusted-bundle`, which leaves every
one of them armed.

On a real run it also sits **behind the repository lock**, and the guards are
re-probed once that lock is held, because a competing run could have switched
branches between the two probes. `--check` and `--check-live` take no lock —
they switch no branches and commit nothing — so there the preflight alone
precedes it.

### Resolving from the branch is for your OWN pull requests

Everything above is safe for a branch in this repository, where pushing already
required write access. It is **not** how you review a contributor's fork.

Standing on a fork's branch means its content is your working directory, and
whatever you run there is somebody else's code running as you — starting *before*
fixpoint exists. `make review-pr` has make parse the pull request's own Makefile
and run its `build`, `test` and `vet` recipes; a `config/` bundle resolved from
the target is the pull request's too, which is exactly the assumption
`-trusted-bundle` was written under (*those are OUR files, read before the
checkout replaced the tree*) and it does not hold here. fixpoint cannot
retroactively guard what make already ran.

So **any** pr-mode run refuses a cross-repository pull request whose head is the
checked-out branch, and says to use `-pr <number>` from a checkout that is **not
on that branch** — `git switch` to your trunk, or a separate clone. There
fixpoint does the checkout itself, behind its guards, with the bundle read before
the tree changed. That is what `make review-pr PR=170` from your trunk has always
done, and it stays the way to review a fork.

The refusal is not conditional on how the number arrived, and that matters twice
over. A pull request's own `config/` can set `target.pr`, and the bundle search
looks in the project's `config/` first — so gating the refusal on "no number was
found" would let the pull request switch it off. And typing `-pr 170` in the
directory where the refusal just fired would otherwise reproduce every condition
the refusal named.

`-trusted-target` overrides it, and means what it says everywhere else: this
checkout is mine. Whether the checkout is a fork's is decided from two signals
and fails **closed**: `gh pr view` for this branch, and a `gh pr list --head`
that exits cleanly on an empty result. If neither can be obtained, the run does
not start.

`make review-pr` and `make fix-pr` take `PR=<n>` the same way, and omitting it now
works instead of failing the usage guard.

Step 3 is `-post-run` and not a second `review-pr -pr 170 -post` because the two
are not the same act. `-post` publishes the review the run in front of it just
produced, so on its own it publishes a review nobody has read; re-running to post
what you read would re-review everything, cost the same again, and publish a
*different* review — the panel is not deterministic, and two runs over one pull
request here produced 34 findings and then 49. `-post-run` takes the run directory
and posts the body off disk, with the anchors that run computed. That is what makes
reading the file first mean anything. It refuses a run that has no verdict (a fix
run), one that did not review a pull request, one from before the PR number was
recorded, one that did not finish — the verdict is written before the round
checks whether it was interrupted, so a review stopped part-way leaves an approval
on disk that the run itself refused to post and exited non-zero over — and one
that has *already* been published, so `-post` followed by `-post-run` cannot leave
two identical reviews on the pull request. Publishing also leaves a `review-posted`
receipt in the run directory, so a second `-post-run` over the same run refuses
too — including after a submission that failed *after* it was sent, which the forge
may well have accepted; the receipt says so, and only you can decide whether the
review is there.

It also refuses a run directory that is **tracked by git**. A run directory is only
files, so a pull request can commit a lookalike `.fixpoint/<run>` next to your own:
a plausible `review-body.md` to read, and a summary whose fields choose a different
pull request, an approval, and inline comments of its own — none of which are in the
file you inspected. fixpoint commits no run artifacts, so a tracked file there means
the directory came in with the code under review rather than from a run on this
machine, and nothing is published. Every anchor's file and line is also logged
before the submission goes out, because the line comments are separate bytes from
`review-body.md` and reading that file does not show them.

Either way, the review is bound to the **commit it was about**. A run records the
head it reviewed, and posting reads the pull request's current head first: if the
author has pushed since — while the panel ran, or in the days between a run and its
`-post-run` — nothing is published. A forge applies a review to whatever the pull
request points at now, so without that check an approval could clear code no
reviewer ever read: push something clean, collect the approval, push the payload.
On GitHub the submission also names that commit (`commit_id`), so the review is
recorded against it and its comments are marked outdated if the branch moves
afterwards.

The **approval itself is not**, and no client can make it so. A forge counts an
approval toward its merge requirements until something clears it, so once fixpoint
has exited an author can push and merge on an approval given for the commit before
— exactly what the check above refuses *during* the run. Only the repository can
close that: on GitHub enable *Dismiss stale pull request approvals when new commits
are pushed*, on GitLab *Remove all approvals when commits are added to the source
branch*. Turn it on before you let anything approve with `-post-verdict`, machine
or human — every approval fixpoint publishes says so in the log, naming the commit
it was for.

It is bound to the **repository** it was about too. A commit is not a destination:
the run directory records the path it ran in, and `-post-run` submits through
whatever checkout is at that path when you run it — which, days later, may have been
reused for another repository on the same forge or had its remotes rewritten. Pull
request 170 of *that* repository would then receive the review, and the head check
would not notice, because the reviewed commit is public and anyone can open a pull
request proposing it. So a run also records the repository `gh` resolved for the
checkout, and `-post-run` refuses unless the checkout still resolves to it — or
cannot say what it resolves to at all.

`-post`/`-post-run` and `-post-verdict` are separate flags because they are
separate acts. The first makes a machine review visible; the second approves
somebody's change or formally blocks it. None of them can be set from a config
file — the first bundle on the search path belongs to the target, so a YAML key
would let reviewed code arrange to have a review posted under your identity.

Reviewing the same pull request twice is fine: findings it already carries are
recognized and not repeated, the new ones are published, and the body says how
many it left out. Recognition is bound to the commit each finding was published
about, so one whose code has been pushed to since is published again in full rather
than counted as old news — see
[config/README.md](../config/README.md#reviewing-the-same-pull-request-twice).

Findings that name a file and a line are also posted as **inline comments**, so a
reader meets each one beside the code instead of in a list at the bottom. Each one
carries the same signature as the body, because it is read in the Files tab with no
sight of the review it belongs to — unsigned, it would be an unattributed assertion
sitting on somebody's code. A forge only accepts an anchor on a line the pull
request actually touches; if it refuses any of them it refuses the whole
submission, so fixpoint retries with the summary alone and says so. Nothing is lost
either way — every finding is in the body.

> **GitLab is not usable today.** There is a `glab`-based provider — it posts a
> note, adds inline discussions, approves and unapproves — but nothing can reach
> it: `pr` mode's `Prepare` unconditionally runs `gh pr checkout`, so a run against
> a GitLab remote fails before any provider is chosen. The code is written and
> tested against a fake CLI; what is missing is a checkout path that is not `gh`.
> Treat every mention of GitLab below as describing code that has never run
> against a real merge request.

**GitHub refuses an approval or a change request on your OWN pull request**, so
`-post-verdict` only does anything when the token belongs to somebody other than
the PR's author — a bot account, or a reviewer running it on a colleague's branch.
`-post` works either way, which is part of why the comment is the default: on your
own PR it is the only thing that can land. A refusal is reported, not swallowed,
and the review is still on disk.

## Fixing a pull request

```sh
./fixpoint review-pr -pr 170                        # read it first
./fixpoint fix-pr -pr 170 --allow-untrusted-fix     # then let it edit
./fixpoint fix-pr -pr 170 --allow-untrusted-fix -post   # …and answer the threads
```

`fix-pr` also **triages the pull request's open comments**: a read-only agent
decides each one before any fixing starts, accepted ones become ordinary issues
with their own session, gate and commit, and declined ones are answered with the
reason. Every conversation gets a decision: a declined one is answered straight
away, an accepted one once its fix commits — and if the coder then rejects that
issue or its gate fails, nothing is claimed and the thread is left for a later
run to decide again. That lets PR comments
direct work, which is a real widening of what untrusted text can ask for — see
[config/README.md](../config/README.md#letting-the-comments-commission-work) for what
bounds it.

Conversations are read **whole**, replies included, and one whose last word is
already this tool's answer is left alone — otherwise every later run would answer
the same comment again, since replying does not resolve a thread. When a person
writes back, the thread is live again and the agent sees the entire exchange,
including what it said last time.

`fix-pr` is the most dangerous config in the bundle and its flag says so: it edits
a working tree holding **externally authored** code, with an agent whose
permission checks are disabled. A payload in the diff, in a commit message, or in
a review comment reaches the coder. It is also shown the pull request's
unresolved conversations and may answer the ones its work addressed — see
[config/README.md](../config/README.md#answering-a-pull-requests-conversations). A
reply is posted only once that session's fix has been **committed**: it goes out
under your identity as a claim that the work landed, and a session whose gate failed
or whose issue was rejected has its edits withdrawn or stashed, so a reply about it
would be a claim about work that does not exist.
