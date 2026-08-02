package logstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/model"
)

// JournalName is the run journal's filename, at the run root beside the summary.
// It spans every round, like the summary, so it does not belong in a round
// directory.
const JournalName = "journal.jsonl"

// Journal appends one state transition to the run journal and flushes it to disk
// before returning.
//
// Durability is the entire point, so this is deliberately not buffered: the
// journal's job is to survive the process that was writing it. Events number in
// the tens per run, so a synchronous open-append-fsync-close per record costs
// nothing measurable next to an agent invocation -- and it means there is no
// handle to leak, no flush to forget, and no teardown path a killed run can skip.
//
// Safe to call concurrently: the parallel reviewer goroutines of a round share one
// Store, and the sequence number must stay gap-free.
func (s *Store) Journal(typ string, round int, data any) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	s.journalMu.Lock()
	defer s.journalMu.Unlock()
	// Claim the number only once marshaling has succeeded. An unserializable Data
	// payload writes no record, so consuming the number first would leave a gap in
	// the ordering key callers are told is gap-free. A failed append is different:
	// the record may have partially landed, so that number stays spent.
	seq := s.journalSeq + 1
	b, err := json.Marshal(model.JournalEvent{
		V:     model.JournalVersion,
		Seq:   seq,
		At:    time.Now(),
		Type:  typ,
		Round: round,
		Data:  data,
	})
	if err != nil {
		return err
	}
	s.journalSeq = seq
	// Redact like every other artifact: payloads carry agent-authored text (reviewer
	// error messages, coder failures), and a prompt-injected agent can smuggle a
	// credential into one. Masking a JSON string value keeps the record well-formed,
	// and json.Marshal escapes any newline, so one event stays one line.
	line := append([]byte(agent.RedactSecrets(string(b))), '\n')

	return appendLine(filepath.Join(s.runDir, JournalName), line)
}

// appendLine appends one record and flushes it, reporting a close failure only when
// the write itself succeeded: on a filesystem that defers errors to close, dropping
// it would report a record as durable when it never landed.
func appendLine(path string, line []byte) (err error) {
	f, ferr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if ferr != nil {
		return ferr
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err := f.Write(line); err != nil {
		return err
	}
	// Sync before Close: a crash must not lose the record that explains the crash.
	return f.Sync()
}

// JournalPath returns the journal's path, for reporting it to the operator. It
// claims the run directory first rather than rendering the template, because the
// claimed name can gain a collision suffix and a reported path that names a
// different directory than the artifacts land in is worse than none. It is empty
// only when the directory cannot be claimed at all -- there is then no journal to
// name, and the write that reports it will fail for the same reason.
//
// Safe to call concurrently, and for the same reason as its neighbours: runDir is
// written inside ensureDir's sync.Once, so going through ensureDir here is what
// orders this read after that write.
func (s *Store) JournalPath() string {
	if err := s.ensureDir(); err != nil {
		return ""
	}
	return filepath.Join(s.runDir, JournalName)
}
