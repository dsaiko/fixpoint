{{.Prelude}}
{{if .Objections}}## Your task: revise
Below are your own draft and the panel's blocking objections to it. Address
every objection, one of two ways:

- **Amend the document.** The objection found a real defect; fix the design.
- **Stand firm.** Record the objection in the "Decisions and dissent" section
  with your reason — a reason you would defend out loud, not "considered and
  rejected".

There is no second pass. What you return here is what ships, so return the
COMPLETE revised document, not a description of the changes.

{{.Draft}}

## The objections
{{.Objections}}
{{else}}## Your task: synthesize
You are the editor. Below are several independent design proposals for the
assignment above, anonymized, and the panel's critiques of each. The panel does
not converge — that is its value — so YOU hold the pen: write the one design
document that will be reviewed and implemented.

How to weigh what is in front of you:

- Judge ideas on the critiques they survived, not on volume or confidence of
  prose. An idea every critic would adopt belongs in the design; an idea one
  proposal states and one critique kills needs your own reading of the code
  of the argument, not a coin flip.
- Synthesis is not averaging. Two half-designs stitched together fail in the
  seam; take a coherent core from wherever it is strongest and graft only what
  fits it.
- You may reject all of them on a point and decide differently — you read
  everything they read — but a decision nobody proposed needs the same scrutiny
  you gave theirs, stated in the dissent section.

The document must contain a section titled exactly **"Decisions and dissent"**:
which proposal each major decision came from, what was rejected and why, and
where the panel disagreed without resolution. Dissent is recorded, not erased —
the reader of this design deserves to know what the panel could not settle.

Write the design itself the way the strongest proposal would: shape, data,
technology, failure and operation, open questions. Specific enough that a
competent implementer could start tomorrow.

## The proposals
{{.Proposals}}

## The critiques
{{.Critiques}}
{{end}}
{{.OutputContract}}
