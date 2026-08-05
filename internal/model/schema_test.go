package model

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// properties returns the property names one level down from the named path in a
// parsed schema, so a test can compare them against a struct's json tags.
func properties(t *testing.T, doc []byte, walk ...string) map[string]map[string]any {
	t.Helper()
	var node map[string]any
	if err := json.Unmarshal(doc, &node); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	for _, step := range walk {
		next, ok := node[step].(map[string]any)
		if !ok {
			t.Fatalf("schema has no %q object at this level (have %v)", step, keys(node))
		}
		node = next
	}
	props, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema node has no properties (have %v)", keys(node))
	}
	out := map[string]map[string]any{}
	for name, raw := range props {
		spec, _ := raw.(map[string]any)
		out[name] = spec
	}
	return out
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// jsonFields lists a struct's json tag names, in declaration order.
func jsonFields(t *testing.T, v any) []string {
	t.Helper()
	rt := reflect.TypeOf(v)
	out := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s has no json tag; the schema cannot describe it", rt.Name(), rt.Field(i).Name)
		}
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				tag = tag[:j]
				break
			}
		}
		out = append(out, tag)
	}
	return out
}

// The schema is a hand-written contract and the structs are the implementation,
// so nothing makes them agree except this test. Drift is silent in the worst
// direction: a field added to ReviewFinding but not to the schema is REJECTED by
// the provider (additionalProperties is false), so every schema-enforced reviewer
// starts failing at once, on a change that looks local and harmless.
func TestOutputSchemasDescribeExactlyTheirStructs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		doc    []byte
		walk   []string
		fields []string
	}{
		{"review", ReviewJSONSchema(), []string{"properties", "findings", "items"}, jsonFields(t, ReviewFinding{})},
		{"fix", FixJSONSchema(), []string{"properties", "results", "items"}, jsonFields(t, FixResult{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			props := properties(t, tc.doc, tc.walk...)
			for _, f := range tc.fields {
				if _, ok := props[f]; !ok {
					t.Errorf("struct field %q is missing from the schema; a provider enforcing additionalProperties:false will reject any reply that sets it", f)
				}
			}
			for name := range props {
				found := false
				for _, f := range tc.fields {
					if f == name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("schema property %q has no struct field; the model will be asked for something nothing reads", name)
				}
			}
		})
	}
}

// The top-level envelope is what the extractor unmarshals into, so it has to
// match too -- a schema whose root property is "finding" rather than "findings"
// yields a perfectly valid reply that decodes to zero findings.
func TestOutputSchemaRootsMatchTheEnvelopeStructs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		doc    []byte
		fields []string
	}{
		{"review", ReviewJSONSchema(), jsonFields(t, ReviewOutput{})},
		{"fix", FixJSONSchema(), jsonFields(t, FixOutput{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			props := properties(t, tc.doc)
			for _, f := range tc.fields {
				if _, ok := props[f]; !ok {
					t.Errorf("envelope field %q is missing from the schema root", f)
				}
			}
		})
	}
}

// Severity is a scheduling input: the cap orders by it and the scoreboard reports
// by it, so a vocabulary the schema and the code disagree on means a provider
// either rejects a valid severity or lets through one SeverityRank sorts last.
func TestReviewSchemaSeverityEnumMatchesTheVocabulary(t *testing.T) {
	props := properties(t, ReviewJSONSchema(), "properties", "findings", "items")
	raw, ok := props["severity"]["enum"].([]any)
	if !ok {
		t.Fatalf("severity has no enum: %v", props["severity"])
	}
	got := make([]string, 0, len(raw))
	for _, v := range raw {
		s, _ := v.(string)
		got = append(got, s)
	}
	if !reflect.DeepEqual(got, Severities) {
		t.Errorf("schema enum = %v, want %v (model.Severities)", got, Severities)
	}
}

// Compact, because Claude Code takes the schema inline as an argv element rather
// than as a path: pretty-printing it would put a few hundred bytes of whitespace
// on every reviewer's command line for nothing.
func TestOutputSchemasAreCompactValidJSON(t *testing.T) {
	for name, doc := range map[string][]byte{"review": ReviewJSONSchema(), "fix": FixJSONSchema()} {
		var any1 any
		if err := json.Unmarshal(doc, &any1); err != nil {
			t.Errorf("%s schema is not valid JSON: %v", name, err)
		}
		for _, b := range doc {
			if b == '\n' || b == '\t' {
				t.Errorf("%s schema is not compact; it goes into an argv element", name)
				break
			}
		}
	}
}
