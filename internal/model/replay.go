package model

import "time"

// ReplayVersion is the schema version stamped on every recorded invocation. A
// reader that does not recognize it must refuse the recording rather than guess:
// a replay whose records are half-understood produces a run that looks real and
// is not, which is worse than no replay at all.
const ReplayVersion = 1

// ReplayStep is one agent invocation, recorded verbatim as a single line of
// replay.jsonl at the run root.
//
// It exists because everything in this program that is worth regression-testing
// sits BETWEEN the agent turns -- contract extraction, the finding matcher, the
// verify gate, the verdict rule, the review body -- and none of it can be
// exercised without an agent's reply. Today changing any of them costs a live
// round: real quota, real wall clock, and a non-deterministic answer that may
// differ from the last one on an unchanged target (the panel's own measurements
// say single-run recall is a sample, not a measurement). A recorded reply turns
// every one of those changes into an ordinary test.
//
// It is a SEPARATE artifact from the .raw step log on purpose. The raw log is
// rendered for a human -- argv, duration and both streams in one blob, under a
// name that logs.pattern decides -- so a reader would have to reverse a
// site-configurable template and then parse prose to recover the reply. This file
// has a fixed name, one record per invocation, and holds the fields the
// orchestrator actually consumed, which is the only definition of "faithful" that
// matters for a replay.
//
// What it does NOT record is the invocation's SIDE EFFECTS. A coder's reply is
// here; the file edits it made are not, because they happened in the target's
// working tree and nothing but that tree ever held them. Replaying a fix round
// would therefore hand the pipeline a coder that claims edits nobody made and a
// verify gate that runs against the unedited tree -- so replay is review-only,
// and replay.Source refuses anything else. See internal/replay.
type ReplayStep struct {
	V   int       `json:"v"`
	Seq int       `json:"seq"`
	At  time.Time `json:"at"`

	// The invocation's identity, which is also the replay key. It is the same
	// tuple the step logs are named by, minus the timestamp: one round's one lens
	// on one agent in one role. Seq breaks ties for a role that legitimately
	// invokes the same tuple twice (a reviewer's reformat retry).
	Role   string `json:"role"`
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
	Round  int    `json:"round"`

	// PromptSHA256 is the hex digest of the prompt this reply answered. Replay
	// compares it against the prompt the replayed run renders and warns when they
	// differ: the reply is then an answer to a question nobody asked, which is
	// exactly the state a prompt-template change produces. It is the digest and not
	// the prompt because the prompt embeds the reviewed material, and this file
	// would otherwise carry a second copy of the diff.
	PromptSHA256 string `json:"prompt_sha256"`

	// The reply as the orchestrator received it: Stdout already unwrapped from the
	// CLI's usage envelope, so a replayed step reaches contract extraction as the
	// live one did.
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr,omitempty"`

	// Err is the invocation's error rendered as text, empty when it succeeded.
	// Recorded as a string because that is all a replay can honestly restore: the
	// error's TYPE steered nothing downstream (every consumer reads its message),
	// and inventing a typed error of the right shape would make a replayed failure
	// claim a provenance it does not have.
	Err string `json:"err,omitempty"`

	DurationMS     int64 `json:"duration_ms"`
	Usage          Usage `json:"usage,omitempty"`
	ProviderStatus int   `json:"provider_status,omitempty"`
}

// ReplayName is the recording's filename, at the run root beside the summary and
// the journal. Fixed, like JournalName and for the same reason: logs.pattern is
// site-configurable, so a reader that had to reverse it could not open a
// recording written under someone else's configuration.
const ReplayName = "replay.jsonl"
