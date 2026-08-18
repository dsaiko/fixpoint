package implement

import (
	"encoding/json"
	"fmt"
	"os"
)

// maxOperatorPlan bounds a plan file read from disk. The largest plan the
// pipeline can accept is max_tasks entries of prose; a megabyte is far past
// that and still refuses a file that is not a plan at all.
const maxOperatorPlan = 1 << 20

// LoadOperatorPlan reads a plan supplied with -plan (§7.4) and checks the ONE
// assertion the file itself is allowed to make: that it was written for this
// design.
//
// The digest rule is the whole reason this is a function rather than a
// json.Unmarshal at the call site:
//
//   - The file must carry provenance.design_sha256 and it must equal the design
//     passed on this invocation. A plan for design X must never be implemented
//     against design Y -- every task's goal, every acceptance criterion and
//     every design_ref would then be about a document nobody is building.
//   - A MISSING digest is a refusal, not a waiver. Accept-if-absent would make
//     the guard bypassable by deleting one key, and that key is exactly what an
//     operator editing the JSON by hand is most likely to disturb.
//   - The refusal prints the line to paste, because §8's hand-written-plan
//     recovery has to stay usable: asserting the pairing costs one line, and
//     omitting it never silently succeeds.
//
// Every EXECUTION field -- run id, coder, verify profile -- is fixpoint's to
// write and is re-injected by the caller for this run. The planner field is
// never forged: a plan that arrived this way is stamped operator-supplied, since
// recording the configured planner agent as the author of a plan it never saw
// would make the audit trail state a falsehood.
func LoadOperatorPlan(path, designSHA string) (Plan, string, error) {
	var pl Plan
	st, err := os.Stat(path)
	if err != nil {
		return pl, "", fmt.Errorf("-plan %s: %w", path, err)
	}
	if st.Size() > maxOperatorPlan {
		return pl, "", fmt.Errorf("-plan %s: %d bytes is larger than a plan can be (%d)", path, st.Size(), maxOperatorPlan)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return pl, "", fmt.Errorf("-plan %s: %w", path, err)
	}
	planSHA := DigestBytes(raw)
	if err := json.Unmarshal(raw, &pl); err != nil {
		return pl, planSHA, fmt.Errorf("-plan %s: not a plan document: %w", path, err)
	}

	claimed := ""
	if pl.Provenance != nil {
		claimed = pl.Provenance.DesignSHA256
	}
	switch {
	case claimed == "":
		return pl, planSHA, fmt.Errorf("-plan %s carries no provenance.design_sha256, so nothing says which design it was written for -- add it to assert the pairing:\n    \"provenance\": {\"design_sha256\": %q}", path, designSHA)
	case claimed != designSHA:
		return pl, planSHA, fmt.Errorf("-plan %s was written for design %.12s, and the design given on this invocation hashes to %.12s -- a plan's tasks, acceptance criteria and design_refs are all about one document, so fixpoint will not implement it against another", path, claimed, designSHA)
	}
	return pl, planSHA, nil
}
