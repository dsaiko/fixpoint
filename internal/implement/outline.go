package implement

import (
	"fmt"
	"strings"
)

// Outline is the design document's section skeleton as the coverage rule reads
// it: the chosen heading level and the headings at that level, in order.
// Extracted by fixpoint from the snapshot itself -- never from the planner --
// so the coverage check (§4.2 rule 6) matches against strings no model chose.
type Outline struct {
	// Level is the outline level: 1 for #, 2 for ##, ...
	Level int
	// Headings are the heading texts at Level, in document order, without the
	// marker ("Rendering", not "## Rendering").
	Headings []string
}

// NoOutlineError reports a document the coverage rule cannot read: no heading
// level occurs more than once. The caller turns this into a preflight refusal
// or, under -no-coverage-check, into a stated "coverage: unchecked".
type NoOutlineError struct{ HeadingsFound int }

func (e NoOutlineError) Error() string {
	return fmt.Sprintf("the design has no section outline fixpoint can use for coverage checking (headings found: %d); add sections, or pass -no-coverage-check to run without this check", e.HeadingsFound)
}

// ExtractOutline reads the document's headings and picks the outline level:
// the SHALLOWEST level that occurs more than once. ATX headings (#..######)
// and setext underlines (=== -> level 1, --- -> level 2) both count; fenced
// code blocks are skipped so a comment character in a shell snippet is not a
// heading. Duplicate headings at the outline level are an error -- heading
// text is the join key shared by the extractor, `coverage` and `design_refs`,
// and a collision collapses two sections into one coverable identity
// (DESIGN.md §4.2 rule 6).
func ExtractOutline(doc string) (Outline, error) {
	type heading struct {
		level int
		text  string
	}
	var all []heading
	lines := strings.Split(doc, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if lvl, text, ok := atxHeading(line); ok {
			all = append(all, heading{lvl, text})
			continue
		}
		// A setext heading is the PREVIOUS line, underlined by this one. The
		// underline must follow a non-empty, non-heading line.
		if i > 0 && isSetextUnderline(trimmed) {
			prev := strings.TrimSpace(lines[i-1])
			if prev != "" && !strings.HasPrefix(prev, "#") && !isSetextUnderline(prev) {
				lvl := 1
				if trimmed[0] == '-' {
					lvl = 2
				}
				all = append(all, heading{lvl, prev})
			}
		}
	}

	counts := map[int]int{}
	for _, h := range all {
		counts[h.level]++
	}
	level := 0
	for l := 1; l <= 6; l++ {
		if counts[l] > 1 {
			level = l
			break
		}
	}
	if level == 0 {
		return Outline{}, NoOutlineError{HeadingsFound: len(all)}
	}

	out := Outline{Level: level}
	seen := map[string]bool{}
	for _, h := range all {
		if h.level != level {
			continue
		}
		if seen[h.text] {
			return Outline{}, fmt.Errorf("the design has two %q sections named %q; heading text is the identity the coverage rule and design_refs join on, so duplicate sections at the outline level must be renamed", strings.Repeat("#", level), h.text)
		}
		seen[h.text] = true
		out.Headings = append(out.Headings, h.text)
	}
	return out, nil
}

// atxHeading parses "## Title" into (2, "Title", true). Up to three leading
// spaces are markdown-legal; a marker with no following space is not a heading
// ("#!/bin/sh" outside a fence, "#hashtag").
func atxHeading(line string) (level int, text string, ok bool) {
	s := strings.TrimLeft(line, " ")
	if len(line)-len(s) > 3 {
		return 0, "", false
	}
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n == len(s) || (s[n] != ' ' && s[n] != '\t') {
		return 0, "", false
	}
	t := strings.TrimSpace(s[n:])
	t = strings.TrimRight(t, "#")
	t = strings.TrimSpace(t)
	if t == "" {
		return 0, "", false
	}
	return n, t, true
}

// isSetextUnderline reports a line of only = or only - (at least one), the two
// setext markers. A "---" horizontal rule is indistinguishable from a level-2
// underline without lookbehind; the caller checks the preceding line.
func isSetextUnderline(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	if c != '=' && c != '-' {
		return false
	}
	for i := range len(s) {
		if s[i] != c {
			return false
		}
	}
	return true
}

// FormatHeading renders a heading the way plans reference it ("## Rendering"),
// which is also how the planner prompt quotes the extracted list -- one string
// identity end to end.
func (o Outline) FormatHeading(text string) string {
	return strings.Repeat("#", o.Level) + " " + text
}

// FormattedHeadings is every outline heading in reference form, for the
// planner prompt and the coverage refusals.
func (o Outline) FormattedHeadings() []string {
	out := make([]string, len(o.Headings))
	for i, h := range o.Headings {
		out[i] = o.FormatHeading(h)
	}
	return out
}
