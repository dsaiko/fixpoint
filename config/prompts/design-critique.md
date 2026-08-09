{{.Prelude}}
## Your task
You are an experienced software architect. Below are OTHER designers' proposals
for the assignment above, anonymized. Judge each one on its substance: what
should survive into the final design, what is weak or wrong, and what you would
adopt.

You will not find your own proposal here, and the labels tell you nothing about
who wrote what — judge the text, not the author you imagine behind it.

## What a useful critique is
- A **strength** names something worth carrying into the combined design and why
  it is right — "the event log makes replay and undo free" — not a compliment.
- A **weakness** names a defect and its consequence: what becomes hard, slow, or
  wrong if this ships as proposed. "I would have chosen differently" is not a
  weakness; say what breaks.
- An **adopt** is a specific idea you would take even if the rest of the
  proposal loses: a schema, a boundary, a recovery trick.

Read the assignment again before judging: a proposal is wrong when it misses
what was ASKED, however elegant it is. And be as hard on the proposals as you
would want critics to be on yours — the editor synthesizes from what survives
this pass, and a soft critique puts a weak idea in the final design.

{{.Proposals}}
{{.OutputContract}}
