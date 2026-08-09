paralelni coder pres worktree -- TRIED AND DISCARDED 2026-08-08 (PR #4):
  works (3h05m of coder time in a 2h23m run) but the gate went 20/20 -> 10/14
  on first attempt. Sessions share a base, so the 2nd patch is written against
  a tree without the 1st fix; a serial gate names the culprit but cannot prevent
  it, and staleFiles has nothing to say because nothing has moved yet at start.
code --> JIRA|LINK
fix-full (without review): tests suite (?)
full + design (design lens, no tests for html)
review design (architecture, not UI)
create-desing
