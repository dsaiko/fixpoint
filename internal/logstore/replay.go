package logstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/model"
)

// Replay appends one agent invocation to replay.jsonl, so the run can later be
// re-run with the agents served from the recording instead of from quota.
//
// It is a no-op when logs.replay is off, so the caller never has to ask: the
// orchestrator records every invocation through one call site and the switch is
// honored here.
//
// Ordering matters more than it does for the step logs, because the replay key is
// not unique on its own -- a reviewer's reformat retry invokes the same
// (role, agent, prompt, round) twice, and the two replies are different. Seq is
// what orders them, so it is claimed under the same mutex the journal uses rather
// than a second one: both files are appended to from the parallel reviewer
// goroutines of a round, and a replay whose sequence numbers interleave wrongly
// would serve round 1's second reviewer the first one's answer.
//
// The prompt is recorded as a DIGEST, not as text. It embeds the reviewed material
// -- in git-diff and pr mode the whole diff -- so keeping it would give this file
// a second copy of the thing the .prompt artifact already holds, and for the only
// question replay asks of it (is this reply still an answer to this question?) a
// digest is sufficient.
func (s *Store) Replay(role, agentName, promptName string, round int, prompt string, res agent.Result) error {
	if !s.cfg.Replay {
		return nil
	}
	if err := s.ensureDir(); err != nil {
		return err
	}
	s.journalMu.Lock()
	defer s.journalMu.Unlock()

	sum := sha256.Sum256([]byte(prompt))
	rec := model.ReplayStep{
		V:              model.ReplayVersion,
		Seq:            s.replaySeq + 1,
		At:             time.Now(),
		Role:           role,
		Agent:          agentName,
		Prompt:         promptName,
		Round:          round,
		PromptSHA256:   hex.EncodeToString(sum[:]),
		Stdout:         res.Stdout,
		Stderr:         res.Stderr,
		DurationMS:     res.Duration.Milliseconds(),
		Usage:          res.Usage,
		ProviderStatus: res.ProviderStatus,
	}
	if res.Err != nil {
		rec.Err = res.Err.Error()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	// Claim the number only once marshaling has succeeded, exactly as Journal does:
	// a record that was never written must not spend an ordinal, or the replay
	// loader's gap check would report a truncated recording for a run that simply
	// failed to serialize one step.
	s.replaySeq = rec.Seq
	// Redacted like every other artifact. The reply is agent-authored text and a
	// prompt-injected reviewer can quote a discovered secret into it; json.Marshal
	// has already escaped the newlines, so masking a string value keeps the record
	// well-formed and on one line.
	//
	// This is also why a replayed run is not bit-identical to the live one it came
	// from: what is replayed is the REDACTED reply. For everything replay exists to
	// test that is invisible (no contract, matcher or gate reads a credential), but
	// it is the reason the recording must never be described as an exact capture.
	line := append([]byte(agent.RedactSecrets(string(b))), '\n')

	return appendLine(filepath.Join(s.runDir, model.ReplayName), line)
}

// ReplayPath returns the recording's path, for reporting it to the operator. Same
// contract as JournalPath: the run directory is claimed rather than re-rendered,
// because the claimed name can carry a collision suffix and naming a directory the
// artifacts did not land in is worse than naming none.
func (s *Store) ReplayPath() string {
	if !s.cfg.Replay {
		return ""
	}
	if err := s.ensureDir(); err != nil {
		return ""
	}
	return filepath.Join(s.runDir, model.ReplayName)
}
