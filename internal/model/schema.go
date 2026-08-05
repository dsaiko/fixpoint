package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// JSON Schema documents for the two output contracts, for agents whose CLI can
// enforce a schema natively (`--json-schema` on Claude Code, `--output-schema` on
// Codex). An agent that gets one returns the bare JSON value instead of the
// <review>/<fix> envelope the prompt otherwise asks for, which removes an entire
// class of failure: the run's contract errors were never wrong FINDINGS, they were
// well-reasoned reviews wrapped in prose, fenced in markdown, or truncated
// mid-envelope, and each one cost a salvage round trip or the whole step.
//
// These are hand-written rather than generated from the structs by reflection,
// because a schema is a CONTRACT and the struct is an implementation: `line` is
// an int in Go and must accept only integers here, but `suggestion` being a Go
// string says nothing about whether the model may omit it. The parity test in
// schema_test.go is what keeps the two from drifting -- it walks the structs'
// json tags and fails if a field exists on one side and not the other.
//
// additionalProperties is false everywhere. A model that invents a field is
// telling us it misread the contract, and finding that out at the provider
// (which re-asks) is better than silently dropping the field here.
//
// Only `issue` carries a description. Structured output makes a model fill in
// every property it is shown -- measured: asked with this schema and no prompt
// contract, Haiku wrote a whole paragraph into `issue`, which means "this is a
// re-report of issue i7" and nothing else. A wrong id degrades safely (the ledger
// falls back to the fingerprint), but the field is the one place where a
// plausible-looking value is meaningless, so the schema says so itself rather
// than relying on the prompt alone.

// reviewSchema mirrors ReviewOutput/ReviewFinding.
//
// Only file, line and title are required. severity and category are NOT, even
// though every finding needs them, because validateReviewFindings already
// normalizes and rejects them with a message that names the offending value --
// a schema violation would instead surface as an opaque provider retry. Nothing
// is gained by enforcing the same rule twice in the less informative place.
const reviewSchema = `{
  "type": "object",
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "issue": {"type": "string", "description": "Only an existing issue id from the History section, to say this is the SAME problem. Omit it otherwise; never put a description here."},
          "category": {"type": "string"},
          "severity": {"type": "string", "enum": [%s]},
          "file": {"type": "string"},
          "line": {"type": "integer"},
          "title": {"type": "string"},
          "description": {"type": "string"},
          "suggestion": {"type": "string"}
        },
        "required": ["file", "line", "title"],
        "additionalProperties": false
      }
    }
  },
  "required": ["findings"],
  "additionalProperties": false
}`

// fixSchema mirrors FixOutput/FixResult.
//
// id and verdict are required: the orchestrator matches every issue it handed
// over against exactly one result, so a result missing either is unusable rather
// than merely incomplete. detail is required too -- a rejection with no reason is
// the one verdict shape a human cannot audit, and the fix prompt now asks the
// coder to reject on value, which makes the reason the whole point.
const fixSchema = `{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "verdict": {"type": "string", "enum": ["fixed", "rejected"]},
          "detail": {"type": "string"}
        },
        "required": ["id", "verdict", "detail"],
        "additionalProperties": false
      }
    },
    "notes": {"type": "string"},
    "replies": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "thread": {"type": "string"},
          "message": {"type": "string"}
        },
        "required": ["thread", "message"],
        "additionalProperties": false
      }
    }
  },
  "required": ["results"],
  "additionalProperties": false
}`

// ReviewJSONSchema returns the review output schema as a compact JSON document.
//
// Compact because Claude Code takes the schema INLINE as an argv element rather
// than as a path, so every byte is an argument-list byte. The document is
// fixpoint's own contract shape and contains nothing from the target, so unlike
// prompt_via: arg there is no secret to leak into a world-readable argv.
func ReviewJSONSchema() []byte {
	quoted := make([]string, 0, len(Severities))
	for _, s := range Severities {
		quoted = append(quoted, `"`+s+`"`)
	}
	return compact(fmt.Sprintf(reviewSchema, strings.Join(quoted, ", ")))
}

// FixJSONSchema returns the coder output schema as a compact JSON document.
func FixJSONSchema() []byte { return compact(fixSchema) }

// compact strips the formatting the constants above carry for readability. It
// panics on invalid JSON: the input is a compile-time constant in this file, so a
// failure here is a typo a developer must fix, never a runtime condition -- and
// returning an error would push a nil-schema path into every caller for a case
// that cannot happen in a built binary. The schema tests parse both documents.
func compact(doc string) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(doc)); err != nil {
		panic("model: output schema is not valid JSON: " + err.Error())
	}
	return buf.Bytes()
}
