package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shared prelude is worth nothing unless every lens renders it byte-identically
// and puts it FIRST: Anthropic's cache matches on an exact leading prefix, so one
// lens opening with its own role line diverges at byte one and the round pays for
// the material once per reviewer. This renders every shipped lens against one
// ReviewData and measures the common prefix.
func TestShippedLensesShareARenderedPrefix(t *testing.T) {
	d := ReviewData{Mode: "directory", Path: "/tmp/x", Round: 3,
		ModeGuidance:   ModeGuidance("directory"),
		Target:         strings.Repeat("material line\n", 500),
		History:        "\n## History of previous rounds\nRound 1 (1 fixed, 0 rejected):\n- [i1] a.go:1 t — FIXED: d\n",
		OutputContract: ReviewContract}
	d.Prelude = FormatPrelude(d)

	paths, err := filepath.Glob("../../config/prompts/review-*.md")
	if err != nil || len(paths) < 4 {
		t.Skipf("shipped lenses not reachable from here (%v, %d found)", err, len(paths))
	}
	rendered := make([]string, 0, len(paths))
	for _, p := range paths {
		tpl, err := Load(p)
		if err != nil {
			t.Fatalf("Load(%s) = %v", p, err)
		}
		out, err := Render(tpl, d)
		if err != nil {
			t.Fatalf("Render(%s) = %v", p, err)
		}
		rendered = append(rendered, out)
	}
	common := rendered[0]
	for _, r := range rendered[1:] {
		n := 0
		for n < len(common) && n < len(r) && common[n] == r[n] {
			n++
		}
		common = common[:n]
	}
	// The prelude must be a PREFIX of what they share, not merely present in it:
	// anything before it that differed would push the divergence to byte one and
	// make the sharing zero, whatever the totals said.
	if !strings.HasPrefix(common, d.Prelude) {
		t.Errorf("the prelude is not a leading prefix of what %d lenses share (%d B shared, %d B prelude); a lens is emitting something before it",
			len(rendered), len(common), len(d.Prelude))
	}
	// Guard the property that actually costs money, not just the ordering: the
	// material and history have to be inside the shared part.
	if !strings.Contains(common, "## Material to review") || !strings.Contains(common, "History of previous rounds") {
		t.Error("the shared prefix does not carry both the material and the history; the round would pay for them once per lens")
	}
	t.Logf("%d lenses share %d B of prompt (prelude is %d B; longest render %d B)",
		len(rendered), len(common), len(d.Prelude), len(rendered[0]))
	if os.Getenv("V") != "" {
		t.Logf("shared prefix tail:\n%s", common[max(0, len(common)-300):])
	}
}
