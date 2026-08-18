package implement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The design pairing is the one assertion a plan file is allowed to make about
// itself, and every way it can be wrong is a refusal (§7.4). A plan for design X
// implemented against design Y would have every goal, acceptance criterion and
// design_ref about a document nobody is building.
func TestLoadOperatorPlan(t *testing.T) {
	const designSHA = "9f2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff001"

	write := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "plan.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	planWith := func(prov string) string {
		return `{"schema_version":1,"project":{"name":"p"},"coverage":[],` + prov +
			`"tasks":[{"id":"T01","title":"t","goal":"g","acceptance":["a"],"files":["a.go"]}]}`
	}

	t.Run("matching digest is accepted", func(t *testing.T) {
		pl, sha, err := LoadOperatorPlan(write(t, planWith(`"provenance":{"design_sha256":"`+designSHA+`"},`)), designSHA)
		if err != nil {
			t.Fatalf("LoadOperatorPlan() = %v", err)
		}
		if len(pl.Tasks) != 1 {
			t.Errorf("parsed %d tasks, want 1", len(pl.Tasks))
		}
		if sha == "" {
			t.Error("the file's own digest was not reported; the audit trail records it")
		}
	})

	t.Run("missing digest is a refusal, not a waiver", func(t *testing.T) {
		_, _, err := LoadOperatorPlan(write(t, planWith("")), designSHA)
		if err == nil {
			t.Fatal("a plan with no design_sha256 was accepted -- the guard is then bypassable by deleting one key")
		}
		// The refusal has to print what to paste, or §8's hand-written-plan
		// recovery is unusable.
		if !strings.Contains(err.Error(), designSHA) {
			t.Errorf("the refusal does not print the digest to add: %v", err)
		}
	})

	t.Run("empty digest is refused too", func(t *testing.T) {
		if _, _, err := LoadOperatorPlan(write(t, planWith(`"provenance":{"design_sha256":""},`)), designSHA); err == nil {
			t.Fatal("an empty design_sha256 was treated as absent-and-fine")
		}
	})

	t.Run("a plan for another design is refused", func(t *testing.T) {
		other := "0000000000000000000000000000000000000000000000000000000000000000"
		_, _, err := LoadOperatorPlan(write(t, planWith(`"provenance":{"design_sha256":"`+other+`"},`)), designSHA)
		if err == nil {
			t.Fatal("a plan written for a different design was accepted")
		}
		if !strings.Contains(err.Error(), other[:12]) || !strings.Contains(err.Error(), designSHA[:12]) {
			t.Errorf("the refusal must name both digests: %v", err)
		}
	})

	t.Run("a file that is not a plan is refused", func(t *testing.T) {
		if _, _, err := LoadOperatorPlan(write(t, "not json at all"), designSHA); err == nil {
			t.Fatal("a non-plan file was accepted")
		}
	})

	t.Run("a missing file names itself", func(t *testing.T) {
		_, _, err := LoadOperatorPlan(filepath.Join(t.TempDir(), "absent.json"), designSHA)
		if err == nil || !strings.Contains(err.Error(), "absent.json") {
			t.Errorf("LoadOperatorPlan(missing) = %v, want a refusal naming the path", err)
		}
	})

	// The digest is of the FILE, so an edit changes it -- that is what makes the
	// audit trail's "operator-supplied (sha …)" mean anything.
	t.Run("the reported digest tracks the bytes", func(t *testing.T) {
		body := planWith(`"provenance":{"design_sha256":"` + designSHA + `"},`)
		_, a, err := LoadOperatorPlan(write(t, body), designSHA)
		if err != nil {
			t.Fatal(err)
		}
		_, b, err := LoadOperatorPlan(write(t, strings.Replace(body, `"name":"p"`, `"name":"q"`, 1)), designSHA)
		if err != nil {
			t.Fatal(err)
		}
		if a == b {
			t.Error("two different plans reported the same digest")
		}
	})

	// A plan is JSON, and the loader must not be a way to read an arbitrary file
	// into memory.
	t.Run("an oversized file is refused before it is parsed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "huge.json")
		if err := os.WriteFile(path, make([]byte, maxOperatorPlan+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadOperatorPlan(path, designSHA); err == nil || !strings.Contains(err.Error(), "larger than a plan") {
			t.Errorf("LoadOperatorPlan(oversized) = %v, want a size refusal", err)
		}
	})

	// Sanity: the fixture really is a valid plan document.
	t.Run("the fixture parses as a Plan", func(t *testing.T) {
		var pl Plan
		if err := json.Unmarshal([]byte(planWith("")), &pl); err != nil {
			t.Fatalf("fixture is not a plan: %v", err)
		}
	})
}
