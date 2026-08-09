package create

import (
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"
)

// Labels assigns each pool agent its anonymous proposal label, deterministically
// (sorted agent names -> A, B, C ...).
//
// Anonymity here means "not told", not "cryptographic": authorship would hand a
// critic -- and the editor -- a reason that is not evidence, the same argument
// the anonymized refutation documents. The mapping this returns is persisted in
// the run's artifacts, so every prompt stays anonymous while the audit trail
// stays attributable.
//
// Deterministic on purpose: artifacts from two runs over the same pool label the
// same agent the same way, which is what makes them comparable side by side.
func Labels(agents []string) map[string]string {
	sorted := append([]string(nil), agents...)
	sort.Strings(sorted)
	out := make(map[string]string, len(sorted))
	for i, a := range sorted {
		out[a] = Label(i)
	}
	return out
}

// Label renders the i-th proposal label: A..Z, then A1, B1, ... -- pools larger
// than 26 do not exist in practice, but a wrapped label is better than a panic.
func Label(i int) string {
	letter := string(rune('A' + i%26))
	if i < 26 {
		return letter
	}
	return fmt.Sprintf("%s%d", letter, i/26)
}

// Caps derives the per-proposal (and per-critique) byte cap from the pool.
//
//	cap = (B - assignment - overhead) / (2 x pool)
//
// where B is the SMALLEST prompt_budget across the pool and the editor. The pool
// is deliberately heterogeneous -- 900 kB in-house agents beside 400 kB served
// ones -- and a cap derived from anything but the minimum overruns exactly the
// agent least able to take it. The divisor is 2x pool because the SYNTHESIZE
// prompt, the largest of the run, carries every proposal AND every critique.
//
// Budgets of 0 mean "no limit" and are skipped; a pool with no limits at all
// gets no cap (0 = unbounded), because inventing one where no agent declared a
// constraint would be a limit nobody asked for.
//
// The floor is a refusal, not a clamp: an assignment so large that proposals
// would be squeezed under CapFloor fails HERE, at startup, before any session is
// paid for -- a proposal cut to a stub would not be a proposal, and the failure
// would surface as a bad design instead of a bad invocation.
func Caps(budgets []int, assignmentBytes int64, pool int) (int, error) {
	if pool < 1 {
		return 0, errors.New("create: no pool to derive caps for")
	}
	minB := 0
	for _, b := range budgets {
		if b <= 0 {
			continue
		}
		if minB == 0 || b < minB {
			minB = b
		}
	}
	if minB == 0 {
		return 0, nil
	}
	usable := int64(minB) - assignmentBytes - capOverhead
	limit := int(usable / int64(2*pool))
	if limit < CapFloor {
		return 0, fmt.Errorf("create: the assignment (%d bytes) leaves %d bytes per proposal under the pool's smallest prompt budget (%d); the floor is %d -- split the assignment or raise the budget, a proposal cut to a stub is not a proposal",
			assignmentBytes, limit, minB, CapFloor)
	}
	return limit, nil
}

// CapFloor is the least room a proposal may be given before the run refuses.
// 8 kB writes roughly two pages -- below that a "proposal" is an abstract, and
// synthesizing abstracts produces one.
const CapFloor = 8 * 1024

// capOverhead reserves room for everything in a prompt that is not assignment or
// proposals: the contract, the guidance, the critique structure. Measured prompt
// framing in this tool runs 2-4 kB; 16 kB is deliberately generous, because the
// failure mode of a tight estimate is an over-budget refusal AFTER the propose
// phase already spent its sessions.
const capOverhead = 16 * 1024

// Clamp bounds one proposal or critique to the cap, stating what it dropped --
// the same rule as every clamp in this tool: a reader who cannot tell a
// truncated text from a short one reads the missing half as absent.
func Clamp(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	cut := s[:limit]
	// Never split a UTF-8 rune: back up to a boundary so the elision notice is
	// appended to valid text. DecodeLastRune reports an invalid tail as
	// (RuneError, 1) -- stripping one byte at a time converges within a rune's
	// width, and a genuine U+FFFD in the text decodes with its true size and is
	// left alone.
	for cut != "" {
		if r, size := utf8.DecodeLastRuneInString(cut); r == utf8.RuneError && size <= 1 {
			cut = cut[:len(cut)-1]
			continue
		}
		break
	}
	return fmt.Sprintf("%s\n\n[%d further byte(s) not shown; the cap is stated in the prompt]", cut, len(s)-len(cut))
}
