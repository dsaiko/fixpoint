// Package replay serves a finished run's recorded agent replies back to the
// pipeline, so everything between the agent turns can be re-run without an agent.
//
// The value it buys is deterministic regression testing of the parts of this
// program that are not the model: contract extraction, the finding matcher, issue
// identity across rounds, the verify gate, the verdict rule and the review body.
// Every one of those is exercised today only by spending a live round -- real
// quota, thirty minutes of wall clock, and an answer that is a SAMPLE rather than
// a measurement, because the same reviewer on an unchanged target finds different
// things on consecutive runs. Against a recording they become ordinary tests.
//
// What a replay is faithful to, and what it is not, is the whole of its contract:
//
//   - Faithful: everything downstream of an agent's REPLY. The bytes the
//     orchestrator consumed are the bytes it consumed again, minus redaction (see
//     logstore.Store.Replay), which no part of the pipeline reads.
//   - Not faithful: an invocation's SIDE EFFECTS. The coder's edits lived in the
//     target's working tree and were never in any artifact, so a replayed fix round
//     would pair a coder claiming edits with a tree that does not have them and a
//     gate that therefore checks the wrong thing. Source refuses to serve a fix
//     role for exactly that reason -- see ErrNotReviewOnly.
//   - Not faithful: wall clock and quota. A replayed step reports the duration and
//     usage the live one did, because those are what the summary is for; nothing
//     was spent this time.
package replay

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/model"
)

// ErrNotReviewOnly is returned when a replayed run asks for an invocation whose
// side effects the recording cannot restore. It is a distinct error because the
// caller turns it into a refusal with an explanation, not a failed step: the run
// is misconfigured, not broken.
var ErrNotReviewOnly = errors.New("replay cannot serve this role")

// sideEffectRoles are the roles whose value to the run is not their reply.
//
// A fix or task invocation EDITS the target. The recording holds what the coder
// said, never what it wrote, so replaying one hands the pipeline a report of work
// that did not happen: the gate then runs against an unedited tree, the round
// commits nothing, and the summary describes a fix that exists only in prose. That
// is not a degraded replay, it is a fabricated one, so it is refused rather than
// warned about.
//
// Reviewer, refute, judge and triage roles are pure: their whole contribution is
// the text they return, which is exactly what was recorded.
var sideEffectRoles = map[string]string{
	"fix":  "a fix round edits the target, and the recording holds the coder's reply but not its edits",
	"task": "an implement task writes files, and the recording holds the coder's reply but not what it wrote",
	"plan": "an implement plan drives a coder that writes files; replaying the plan alone would claim a project that was never built",
}

// key identifies one invocation. It is the tuple the step artifacts are named by
// with the timestamp dropped, which is what makes a record addressable from a
// replayed run that is producing its own timestamps.
type key struct {
	round  int
	role   string
	agent  string
	prompt string
}

func (k key) String() string {
	return fmt.Sprintf("round %d %s/%s via %s", k.round, k.role, k.agent, k.prompt)
}

// Source serves recorded replies. It is safe for concurrent use: a round fans its
// reviewers out in parallel, and each one asks for its own key.
type Source struct {
	dir string
	// queued holds the records for each key in recorded order. Serving pops from
	// the front, so a role that legitimately invokes one key twice in a round -- a
	// reviewer's reformat retry -- gets the two replies in the order it got them
	// live, rather than the same one twice.
	mu     sync.Mutex
	queued map[key][]model.ReplayStep
	served map[key]int
	// mismatches accumulates keys whose recorded prompt digest did not match the
	// prompt the replayed run rendered. Reported once at the end rather than per
	// step: a prompt-template change makes EVERY key mismatch, and a warning per
	// reviewer per round would bury the run log.
	mismatches []string
	// refused accumulates the invocations the recording could not answer. They are
	// NOT visible in the unserved tally -- a key with no records at all never
	// appears in queued, so a replay that asked for something the recording lacks
	// and consumed everything it did have would otherwise report "matched the
	// recording exactly" over a run that diverged (review run 20260916-085129,
	// finding i4).
	refused []string
}

// Load reads a run's replay.jsonl. dir is the run directory -- the one holding
// summary, journal and recording -- which is what the operator has a path to.
func Load(dir string) (*Source, error) {
	path := filepath.Join(dir, model.ReplayName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no recording at %s: the run that produced %s was made with logs.replay off, or by a fixpoint older than the recording. Re-run it once with `replay: true` to record one", path, dir)
		}
		return nil, err
	}
	// The error is deliberately dropped: this handle is read-only, so a close
	// failure cannot have lost anything the read already returned.
	defer func() { _ = f.Close() }()

	src := &Source{dir: dir, queued: map[key][]model.ReplayStep{}, served: map[key]int{}}
	sc := bufio.NewScanner(f)
	// A reply is capped at 10 MB per stream by agent.maxOutput, and one record
	// carries both plus the envelope, so the default 64 KB token is far too small
	// and a long reviewer answer would fail the scan as "token too long".
	sc.Buffer(make([]byte, 0, 64*1024), 24<<20)
	var seqs []int
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var rec model.ReplayStep
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		// Refuse a version this build does not know rather than serving the fields it
		// happens to recognize. A recording that is half-understood produces a run that
		// looks real and is not, which is the one outcome a replay must never have.
		if rec.V != model.ReplayVersion {
			return nil, fmt.Errorf("%s line %d: recording schema v%d, this build understands v%d", path, line, rec.V, model.ReplayVersion)
		}
		k := key{round: rec.Round, role: rec.Role, agent: rec.Agent, prompt: rec.Prompt}
		src.queued[k] = append(src.queued[k], rec)
		seqs = append(seqs, rec.Seq)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(seqs) == 0 {
		return nil, fmt.Errorf("%s is empty: the run recorded no agent invocation, so there is nothing to replay", path)
	}
	// Order each key's records by the sequence they were recorded in. The file is
	// append-only so it is already in that order, but the records of ONE key are
	// interleaved with every other key's, and a truncated or hand-edited file is
	// exactly the case where trusting file order silently serves the wrong reply.
	for k := range src.queued {
		recs := src.queued[k]
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Seq < recs[j].Seq })
		src.queued[k] = recs
	}
	// A gap means the recording is incomplete -- a killed run, a truncated copy --
	// and a replay over it would report a reviewer as having answered nothing when
	// really nothing was written down. Say so now, with the whole file in hand,
	// rather than at the step that comes up missing an hour in.
	sort.Ints(seqs)
	for i, seq := range seqs {
		if seq != i+1 {
			return nil, fmt.Errorf("%s: the recording is incomplete -- expected sequence %d, found %d (the run it came from was killed, or the file was truncated)", path, i+1, seq)
		}
	}
	return src, nil
}

// Dir is the run directory this recording was loaded from, for the summary field
// that marks a replayed run as replayed.
func (s *Source) Dir() string { return s.dir }

// Steps is how many invocations the recording holds, for the line that tells the
// operator what they are about to replay.
func (s *Source) Steps() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, recs := range s.queued {
		n += len(recs)
	}
	return n
}

// Serve returns the recorded reply for one invocation.
//
// prompt is the prompt the REPLAYED run rendered; it is hashed and compared with
// the digest recorded beside the reply. A mismatch does not fail the step, because
// changing a prompt template is one of the things a replay exists to test -- it is
// collected and reported at the end, so that "the matcher behaves differently" is
// never silently explained by "the reviewer was answering a different question".
func (s *Source) Serve(role, agentName, promptName string, round int, prompt string) (agent.Result, error) {
	if why, ok := sideEffectRoles[role]; ok {
		return agent.Result{}, fmt.Errorf("%w: role %q -- %s. Replay a review-only run (-review-only), or re-run this one live", ErrNotReviewOnly, role, why)
	}
	k := key{round: round, role: role, agent: agentName, prompt: promptName}

	s.mu.Lock()
	defer s.mu.Unlock()
	recs := s.queued[k]
	n := s.served[k]
	if n >= len(recs) {
		// Fail the step rather than invent a reply. A replayed run that reaches an
		// invocation the recording does not have has diverged from the run it came
		// from -- a different lens list, a different agent, an extra round -- and the
		// useful answer is to say which invocation and stop, not to serve an empty
		// review that the pipeline would read as a clean one.
		s.refused = append(s.refused, k.String())
		if len(recs) == 0 {
			return agent.Result{}, fmt.Errorf("the recording has no reply for %s: the replayed run is invoking something the recorded one did not (a changed lens list, agent or round count). Replay the config the recording was made with", k)
		}
		return agent.Result{}, fmt.Errorf("the recording has only %d repl%s for %s, and the replayed run asked for another: it is taking more turns than the recorded one did", len(recs), plural(len(recs)), k)
	}
	rec := recs[n]
	s.served[k] = n + 1

	sum := sha256.Sum256([]byte(prompt))
	if got := hex.EncodeToString(sum[:]); got != rec.PromptSHA256 {
		s.mismatches = append(s.mismatches, k.String())
	}
	res := agent.Result{
		Stdout:         rec.Stdout,
		Stderr:         rec.Stderr,
		Duration:       time.Duration(rec.DurationMS) * time.Millisecond,
		Usage:          rec.Usage,
		ProviderStatus: rec.ProviderStatus,
	}
	if rec.Err != "" {
		// Restored as a plain error: only its message steered anything downstream, and
		// a synthesized typed error would claim a provenance the recording does not
		// have. See model.ReplayStep.Err.
		res.Err = errors.New(rec.Err)
	}
	return res, nil
}

// Divergence describes how the replayed run differed from the recorded one, for
// the report at the end. Both halves matter and neither is an error: a replay that
// used fewer steps converged earlier, which is exactly the kind of change a
// matcher or gate edit is supposed to produce and exactly what a reader must be
// told about rather than left to infer from a summary that looks normal.
type Divergence struct {
	// Unserved are recorded invocations the replayed run never asked for.
	Unserved []string
	// PromptChanged are invocations whose prompt no longer hashes to the recorded
	// one, so the reply is an answer to a different question.
	PromptChanged []string
	// Refused are invocations the recording had no reply for. Each one also failed
	// its step, so the run reports them too -- but they belong here because this is
	// the structure that decides whether the replay "matched", and a replay that was
	// refused anything did not.
	Refused []string
}

// Any reports whether anything is worth printing.
func (d Divergence) Any() bool {
	return len(d.Unserved) > 0 || len(d.PromptChanged) > 0 || len(d.Refused) > 0
}

// Divergence reports what did not line up. Call it once the run is over.
func (s *Source) Divergence() Divergence {
	s.mu.Lock()
	defer s.mu.Unlock()
	var d Divergence
	for k, recs := range s.queued {
		if extra := len(recs) - s.served[k]; extra > 0 {
			d.Unserved = append(d.Unserved, fmt.Sprintf("%s (%d unserved)", k, extra))
		}
	}
	sort.Strings(d.Unserved)
	d.PromptChanged = append(d.PromptChanged, s.mismatches...)
	sort.Strings(d.PromptChanged)
	d.Refused = append(d.Refused, s.refused...)
	sort.Strings(d.Refused)
	return d
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
