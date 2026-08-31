# Archived: the 2026-08-20 sweep, on the retired 100-point scale

These 28 rows (14 models) were measured against **60 go seeds + 40 design
seeds**. The 2026-09-01 retire/replace changed that pool to 56 + 34, so the
scale they were scored on no longer exists.

They are kept, not deleted, and **not** because deletion is hard to undo — git
has them either way. They are kept because the working tree is what people read,
and three things here are still load-bearing.

## Read this before quoting a number

| column | status |
|---|---|
| `recall`, `matched_seeds`, `seeds`, `found_per_mtok` | **VOID** — different seed set. Never compare these to a current row. |
| `errors` (contract failures) | **still true** — a session that returned unparseable output did so regardless of what the manifest asked. |
| `duration_s`, `tokens_in`, `tokens_out`, `cache_read`, `sessions` | **still true** — these measure the model and the route, not the seed pool. |

## Why these rows still matter

1. **14 models exist only here**, including `minimax-m3:cloud` — the ollama seat
   the panel is running today. Deleting this file would leave the seated
   reviewer with no measurement anywhere.
2. **`config/defaults.yaml` cites these numbers** as the reasoning for the
   current panel composition ("deepseek is the better reviewer alone, 66/100 to
   minimax's 57"). The scores are on the retired scale, but they are the record
   of why the seats are what they are.
3. **Two disqualifications rest on the scale-independent half**:
   `mistral-large-3:675b-cloud` with 2 contract failures and
   `nemotron-3-super:cloud` with 1. Contract compliance is the benchmark's
   disqualifier and it does not care how many seeds there were.

## Why there is an archive at all

Because the summaries were gone. Only 15 of 43 runs still had a
`.fixpoint/<run>/summary-*.json`, so only those could be re-scored onto the new
scale for free; the rest had nothing left to re-score. `bench/summaries/` now
keeps a re-scoring extract of every scored run precisely so that a future
manifest change costs nothing and produces no second archive.
