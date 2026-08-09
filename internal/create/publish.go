package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsaiko/fixpoint/internal/model"
)

// Provenance is what fixpoint itself knows about how the deliverable was made.
//
// Stamped by fixpoint, never written by the editor: a degradation notice the
// degraded model writes about itself is not a notice -- the same rule that keeps
// the review signature outside every region carrying agent text.
type Provenance struct {
	RunID     string
	Editor    string
	Pool      int
	Proposals int // survivors that reached the editor
	Critiques int // critique sets that reached the editor
	// The degradations a reader must be able to see: a synthesis of one proposal
	// is one model's opinion, an uncritiqued one skipped its cross-examination,
	// and an unrevised one carries objections nobody applied.
	SingleModel bool
	Uncritiqued bool
	Unrevised   bool
	// SkippedLinks reports symlinks the snapshot left behind -- part of the
	// assignment the run did NOT see.
	SkippedLinks int
}

// Deliverable assembles the published document: fixpoint's provenance header,
// the editor's text verbatim, and -- when REVISE failed -- the unapplied
// objections in a fixpoint-owned appendix.
//
// The appendix is fixpoint's, never spliced into the editor's prose: the
// document's internal structure is text no machine reliably edits, which the
// specification's own review established when the first draft tried exactly
// that.
func Deliverable(doc string, p Provenance, unapplied []model.Objection) string {
	var b strings.Builder
	b.WriteString(header(p))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(doc, "\n"))
	b.WriteString("\n")
	if len(unapplied) > 0 {
		b.WriteString("\n---\n\n## Appendix: unapplied objections (stamped by the tool)\n\n")
		b.WriteString("The revision pass did not complete, so the panel's blocking objections are\nrecorded here verbatim rather than silently discarded. Weigh them as you read.\n\n")
		for i, obj := range unapplied {
			fmt.Fprintf(&b, "%d. **%s** — %s", i+1, sanitizeLine(obj.Passage), sanitizeLine(obj.Defect))
			if obj.Consequence != "" {
				fmt.Fprintf(&b, " (%s)", sanitizeLine(obj.Consequence))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// header is the provenance block. Blockquote lines rather than an HTML comment:
// this is information a READER needs (is this a synthesis or one model's
// opinion?), not machine bookkeeping to hide.
func header(p Provenance) string {
	var flags []string
	if p.SingleModel {
		flags = append(flags, "SINGLE-MODEL: only one proposal survived; this is one model's design, not a synthesis")
	}
	if p.Uncritiqued {
		flags = append(flags, "UNCRITIQUED: no cross-examination reached the editor")
	}
	if p.Unrevised {
		flags = append(flags, "UNREVISED: the objection pass ran but the revision did not; see the appendix")
	}
	if p.SkippedLinks > 0 {
		flags = append(flags, fmt.Sprintf("PARTIAL ASSIGNMENT: %d symlink(s) in the assignment were not followed", p.SkippedLinks))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "> Drafted by an AI panel and synthesized by a single editor · run %s\n", sanitizeLine(p.RunID))
	fmt.Fprintf(&b, "> %d proposal(s) from a pool of %d, %d critique set(s) · this header is stamped by the tool, not written by the editor\n", p.Proposals, p.Pool, p.Critiques)
	for _, f := range flags {
		fmt.Fprintf(&b, ">\n> **%s**\n", f)
	}
	return b.String()
}

// sanitizeLine keeps a value to one line inside a block fixpoint signs with its
// own voice. Same rule as the reply marker: a value that could add lines could
// forge the header's shape.
func sanitizeLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Publish writes the deliverable, atomically and without ever replacing an
// existing file.
//
// Temp-file-then-link: the content is written beside its destination and
// link(2)ed into place, and link fails when the target exists -- which makes the
// no-overwrite refusal and the publication one atomic operation instead of a
// check racing a write. (Plain rename would silently replace the very file the
// refusal exists to protect; the specification's review caught that in its first
// draft.) A failed write leaves only the temp file, removed here or overwritten
// by the next run -- it can never leave a truncated deliverable squatting on the
// protected name.
func Publish(path, content string) error {
	tmp := path + ".fixpoint-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := os.Link(tmp, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s already exists; a regenerated design must not silently replace a reviewed one -- name a different -out", path)
		}
		return err
	}
	return nil
}

// DefaultOut is where the deliverable goes when -out is not given: DESIGN.md
// beside the assignment.
func DefaultOut(assignment string, isDir bool) string {
	dir := assignment
	if !isDir {
		dir = filepath.Dir(assignment)
	}
	return filepath.Join(dir, "DESIGN.md")
}
