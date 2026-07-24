package target

import "testing"

func TestCompileGlobs(t *testing.T) {
	cases := []struct {
		glob  string
		path  string
		match bool
	}{
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/agent/agent.go", true},
		{"**/*.go", "main.txt", false},
		{"*.go", "main.go", true},
		{"*.go", "internal/main.go", false}, // * must not cross separators
		{"**/vendor/**", "vendor/x/y.go", true},
		{"**/vendor/**", "a/vendor/y.go", true},
		{"**/vendor/**", "avendor/y.go", false},
		{"internal/**", "internal/config/config.go", true},
		{"internal/**", "cmd/main.go", false},
		{"?.go", "a.go", true},
		{"?.go", "ab.go", false},
		{"**/café.go", "src/café.go", true}, // multi-byte literals
		{".github/**/*.yaml", ".github/workflows/ci.yaml", true},
	}
	for _, tc := range cases {
		res, err := compileGlobs([]string{tc.glob})
		if err != nil {
			t.Fatalf("compile %q: %v", tc.glob, err)
		}
		if got := matchAny(res, tc.path); got != tc.match {
			t.Errorf("glob %q vs %q: got %v, want %v", tc.glob, tc.path, got, tc.match)
		}
	}
}
