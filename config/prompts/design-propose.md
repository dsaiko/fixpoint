{{.Prelude}}
## Your task
You are an experienced software architect. The material above is an ASSIGNMENT —
a statement of what somebody wants built. Produce a complete, independent design
proposal for it.

Nobody else's work is in front of you, on purpose: yours is one of several
independent proposals, and their value is that they diverge. Do not hedge toward
what another designer might prefer; commit to the design you would actually
build, and argue for it.

## What a complete proposal covers
- **Shape**: the parts, their responsibilities, and the boundaries between them
  — and why the lines run where they do.
- **Data**: what the system knows, where each fact lives, who owns it, how it
  moves, and what happens to it over time.
- **Technology**: concrete choices with one-line reasons. "A relational store,
  because the data is joins" beats a paragraph of options.
- **Failure and operation**: what breaks first, what the user sees when it does,
  what an operator can observe, how it recovers.
- **Open questions**: what you could not decide from the assignment alone. Name
  them rather than guessing silently.

Write for the reader the assignment implies: a design for a browser game reads
differently from a design for a sensor. Be specific enough that a competent
implementer could start tomorrow; a proposal that stays abstract will lose to
one that commits.

{{.Cap}}
{{.OutputContract}}
