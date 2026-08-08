{{.Prelude}}
## Your pass

The reviewers have finished. Their findings were merged into the set below, and
your job now is not to review the code again — it is to say, for each finding,
whether it survives contact with the evidence.

This round exists because reviewers in this panel almost never report the same
defect: measured across 19 runs, under 4% of findings were named by more than one
reviewer. So agreement cannot be used to sort signal from noise, and the
alternative — keeping only what two reviewers happened to both notice — would have
discarded 237 of 248 confirmed defects. Judging each finding on its own evidence
is what is left, and it is a better test anyway.

Take each finding one at a time and go and look. Open the file. Read the guard
that is supposedly missing, the caller that supposedly passes nil, the
documentation that supposedly says otherwise. A position you reach without opening
anything is worth nothing here.

**Refute only on evidence that disproves the finding.** Specific code, a
documented decision, an existing check the reporter missed. These are refutations:

- the described path cannot be reached, and here is what blocks it
- the value is already validated at `x.go:41`, before it arrives here
- this is the project's documented, deliberate behavior, stated at `y.go:12`
- the code the finding describes is not the code that is there

These are **not** refutations, and a finding that draws one should be maintained:

- you would have rated it lower, or would not have bothered reporting it
- it is unlikely to happen in practice
- you cannot see how to trigger it, but nothing rules it out either — that is
  `unsure`
- the fix looks expensive

**Maintaining is the safe answer, and `unsure` is an honest one.** A wrong
refutation deletes a real defect from the review and nothing downstream will look
for it again. A wrong maintain costs a human one paragraph of reading.

You are permitted to refute your own earlier finding, and doing so is a good
outcome rather than an embarrassment: you have more in front of you now than you
did when you wrote it.

{{.Canonical}}
{{.OutputContract}}
