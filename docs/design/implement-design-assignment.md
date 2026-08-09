# Assignment: implement-design

Design the third increment of fixpoint's design pipeline: `implement-design`,
the config that turns a reviewed design document into a working project.

Fixpoint is an agent-agnostic automated code-review loop written in Go. It
already has: review/fix loops over code (one issue per coder session, each fix
committed separately behind a deterministic verify gate); `review-design`
(a panel reviews a design document); and `create-design` (a panel drafts one:
independent proposals, anonymous critique, an editor synthesizes). Agents are
CLI programs (claude, codex, ollama-served models) invoked with a prompt on
stdin; a reviewer pool is shared configuration; trust is asserted only by
command-line flags, never by config files, because a config can ship inside the
thing being reviewed.

What implement-design must do:

- Input: a design document (a markdown file, typically DESIGN.md produced by
  create-design and reviewed by review-design). Output: a new project directory
  with a working initial implementation.
- Implementation is done by the coder alone -- no reviewer panel during
  implementation. Quality comes afterwards, from the existing review-code and
  fix-code loops over the produced project.
- A whole project does not fit one coder session (sessions have a ~30 minute
  timeout and produce one commit each). Something must break the design into
  ordered implementation tasks; each task should be one session and one commit,
  and the project should build after every commit where a build exists.
- The tool's verify gate runs the project's own build/test commands between
  coder sessions, but they must come from the operator's configuration, never
  from the design document -- the design is untrusted input, and letting it
  name commands would let a document commission arbitrary execution.
- Some projects have no build at all (a browser game in plain HTML+JS). The
  pipeline must work there too: no gate, commits land ungated, exactly like
  fixpoint's existing fix loop on such projects.
- The new project starts as a fresh git repository so the per-task commits and
  the later review-code/fix-code loops have something to work on.

Constraints to respect: one issue/task per coder session is a hard-won rule
(verify names the culprit, git revert undoes one fix); the coder is the only
role allowed to edit files; everything the tool publishes or writes states its
provenance; failures degrade loudly rather than silently.

Open questions the design should answer: who decomposes the design into tasks
and in what format; what happens when a task fails its session; how the run
ends (what is reported, what exit code); and where the boundary lies between
implement-design finishing and fix-code taking over.
