package implement

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// The outline level is derived from the document, not assumed (§4.2 rule 6):
// the shallowest level that occurs more than once.
func TestExtractOutline(t *testing.T) {
	cases := []struct {
		name     string
		doc      string
		level    int
		headings []string
		wantErr  string
	}{
		{"title plus sections outlines at ##",
			"# Title\n\n## One\ntext\n## Two\n", 2, []string{"One", "Two"}, ""},
		{"sections at # outline at #",
			"# One\n\n# Two\n", 1, []string{"One", "Two"}, ""},
		{"title and deep subsections outline at ###",
			"# Title\n\n### A\n\n### B\n", 3, []string{"A", "B"}, ""},
		{"fenced heading markers are not headings",
			"# T\n```\n# not a heading\n# nor this\n```\n## A\n## B\n", 2, []string{"A", "B"}, ""},
		{"setext underlines count",
			"One\n===\ntext\n\nTwo\n===\n", 1, []string{"One", "Two"}, ""},
		{"setext dashes are level 2",
			"# T\n\nOne\n---\n\nTwo\n---\n", 2, []string{"One", "Two"}, ""},
		{"no headings at all", "just prose\n", 0, nil, "no section outline"},
		{"single heading only", "# Title\nprose\n", 0, nil, "no section outline"},
		{"duplicate headings refused",
			"## Same\n\n## Same\n", 0, nil, "duplicate sections"},
		{"hashbang is not a heading",
			"#!/bin/sh\n## A\n## B\n", 2, []string{"A", "B"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, err := ExtractOutline(tc.doc)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ExtractOutline() err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractOutline() = %v", err)
			}
			if o.Level != tc.level || !reflect.DeepEqual(o.Headings, tc.headings) {
				t.Fatalf("outline = %d %v, want %d %v", o.Level, o.Headings, tc.level, tc.headings)
			}
		})
	}
}

// ErrNoOutline is a distinct type so the CLI can offer -no-coverage-check for
// exactly this failure and no other.
func TestErrNoOutlineIsDetectable(t *testing.T) {
	_, err := ExtractOutline("prose only")
	var e NoOutlineError
	if !errors.As(err, &e) {
		t.Fatalf("want NoOutlineError, got %T: %v", err, err)
	}
	if e.HeadingsFound != 0 {
		t.Errorf("HeadingsFound = %d, want 0", e.HeadingsFound)
	}
}

func TestFormattedHeadings(t *testing.T) {
	o := Outline{Level: 2, Headings: []string{"Rendering"}}
	if got := o.FormattedHeadings(); got[0] != "## Rendering" {
		t.Errorf("FormattedHeadings() = %v", got)
	}
}
