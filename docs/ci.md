# Running in CI

Using the exit code as a merge gate, and what a CI run must not be allowed to do.

[← back to the README](../README.md)

## fixpoint is already a gate

There is no GitHub App to install and no service to run. fixpoint is a CLI whose
**exit code is the verdict**, so a workflow job that runs it is a check like any
other — and a check is what branch protection makes required. That is the whole
mechanism:

| Exit | Meaning | As a gate |
|---|---|---|
| `0` | Converged, or the review **approved** | pass |
| `4` | The review **requested changes** | fail |
| `5` | The review was **inconclusive** — nothing blocking was found, but the panel did not reach quorum or the judge did not finish | your call; see below |
| `3` | The coder rejected every issue, so nothing changed | fail — "nobody agreed there was a problem" is not "the code is clean" |
| `2` | Hit `max_iterations` without converging, or a usage error | fail |
| `1` | Any other failure or interruption | fail |

A verdict only ever makes the status worse, so an errored run keeps its own code:
the job cannot go green because a reviewer crashed.

## A review gate on pull requests

```yaml
name: fixpoint
on: pull_request

permissions:
  contents: read        # no write scope: this job must not be able to push
  pull-requests: write  # only if you pass -post; drop it otherwise

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # the diff needs the base commit

      # The repository is private, so the asset needs an authenticated download.
      # Unpack it whole and put the DIRECTORY on PATH: fixpoint compiles no
      # fallback configuration into the binary, so a binary installed away from
      # its `config/` bundle refuses to run and prints where it looked.
      - name: Install fixpoint
        env:
          GH_TOKEN: ${{ secrets.FIXPOINT_RELEASE_TOKEN }}
        run: |
          gh release download --repo dsaiko/fixpoint --pattern '*Linux_x86_64.tar.gz'
          mkdir -p "$HOME/.local/fixpoint"
          tar xzf fixpoint_*_Linux_x86_64.tar.gz -C "$HOME/.local/fixpoint"
          echo "$HOME/.local/fixpoint" >> "$GITHUB_PATH"

      # Install whatever agent CLIs your panel names, and give each one its
      # credential as a repository secret. An agent's env.pass names the
      # variables it receives; nothing else in the environment reaches it.
      - name: Review
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: fixpoint review-pr -pr ${{ github.event.pull_request.number }} -trusted-bundle
```

Make that job required in branch protection and a pull request whose review
requests changes cannot be merged.

## What a CI run must not be allowed to do

CI is the worst place to be careless about this, because the code under review is
the pull request's and nobody is watching the run.

- **Review only. No fix rounds.** `review-pr` is review-only by design. Do not
  pass `-allow-untrusted-fix` in CI: the coder edits files with permission checks
  disabled, and the content steering it is the pull request's.
- **`-trusted-bundle`, never `-trusted-target`.** The narrow flag says only that
  the bundle files are yours to run. `-trusted-target` additionally speaks for the
  pull request author's content — which in CI you have not read. See
  [Security model](security.md).
- **Give the job no write scope.** `contents: read`. A prompt-injected reviewer
  cannot push what the token cannot push.
- **A fork's pull request gets no secrets, and that is correct.** `pull_request`
  does not expose secrets to a fork, so the job will fail to authenticate rather
  than hand a stranger's branch your API keys. Do not reach for
  `pull_request_target` to "fix" it: that runs with your secrets against the
  fork's code, which is precisely the exposure the split exists to prevent. Review
  forks on a machine you control instead.
- **Consider `sandbox.command`.** A CI runner is disposable, which removes much of
  the motive — but the agent still holds your API key and can still read the
  runner's filesystem. See
  [Confining agents](security.md#confining-agents-with-sandboxcommand).

## Deciding what to do with inconclusive

Exit `5` means the panel found nothing blocking *and* could not speak with
authority — a reviewer died, or quorum was not reached. It is deliberately not
`0`, because silence from a panel that did not finish is not evidence.

Which way it should fail is a policy question, so fixpoint does not answer it:

```yaml
      - name: Review
        run: |
          set +e
          fixpoint review-pr -pr ${{ github.event.pull_request.number }} -trusted-bundle
          status=$?
          set -e
          # Treat inconclusive as a pass, but say so loudly in the log.
          if [ "$status" -eq 5 ]; then
            echo "::warning::fixpoint could not reach a verdict; merging on an unreviewed diff"
            exit 0
          fi
          exit "$status"
```

Two things about that snippet are load-bearing.

`set +e` around the call, because Actions runs each `run:` under `bash -e`: without
it the step aborts the moment fixpoint exits non-zero, and the `if` below — the
whole point of the step — never executes.

And the command runs **bare**, with its own `$?` read straight afterwards. Pipe it
into anything, `tee` or a formatter, and `$?` is the pipeline's status rather than
fixpoint's, so a review that requested changes reports success and the gate opens.

## Keeping the run reproducible

Add `-replay` to a scheduled job and you can re-run a recorded review against a
new fixpoint build with no quota at all, which is how you tell a change in the
tool from a change in the models:

```sh
fixpoint review-code -review-only -replay .fixpoint/20260916-081500
```

See [the replay recording](logs.md#the-replay-recording) for what that is and is
not faithful to.

## Artifacts

Run artifacts can contain secrets even after redaction. If you upload them, mark
the artifact private and short-lived, or upload only `review-body.md`. `.fixpoint/`
is owner-only on disk; an uploaded artifact is not.
