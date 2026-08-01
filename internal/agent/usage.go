package agent

import (
	"encoding/json"
	"strings"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// ParseUsage reads an agent's machine-readable output: it returns the agent's
// actual reply, extracted from the envelope, plus what the CLI says the call
// cost. See config.AgentUsage for why this is configured per agent rather than
// known centrally.
//
// It FAILS OPEN. A CLI that errored, was killed mid-write, or changed its output
// shape yields unparseable output, and the reply matters far more than the
// accounting: on any failure the raw stdout is returned unchanged and usage is
// left zero. The alternative -- discarding a round's findings because the token
// counter could not be read -- inverts the priorities.
func ParseUsage(u config.AgentUsage, stdout string) (text string, usage model.Usage) {
	if !u.Enabled() {
		return stdout, model.Usage{}
	}
	objs := decodeEnvelope(u.Format, stdout)
	if len(objs) == 0 {
		return stdout, model.Usage{}
	}
	// Last value wins: a jsonl stream reports its running state line by line, and
	// the final mention of a path is the settled one. For a single object it is
	// simply the only value.
	text = stdout
	if s, ok := lastString(objs, u.Text); ok {
		text = s
	}
	usage.InputTokens = lastInt(objs, u.InputTokens)
	usage.OutputTokens = lastInt(objs, u.OutputTokens)
	usage.CacheReadTokens = lastInt(objs, u.CacheReadTokens)
	usage.CacheWriteTokens = lastInt(objs, u.CacheWriteTokens)
	if c, ok := lastFloat(objs, u.CostUSD); ok {
		usage.CostUSD = c
		usage.CostKnown = true
	}
	return text, usage
}

// decodeEnvelope parses stdout into the objects the paths are resolved against.
// A jsonl stream is read line by line and undecodable lines are skipped, because
// a CLI may interleave non-JSON noise (a warning, a progress bar) with its
// records and one such line must not discard the rest.
func decodeEnvelope(format, stdout string) []map[string]any {
	switch format {
	case config.UsageFormatJSONL:
		var out []map[string]any
		for _, line := range strings.Split(stdout, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m map[string]any
			if json.Unmarshal([]byte(line), &m) == nil {
				out = append(out, m)
			}
		}
		return out
	case config.UsageFormatJSON:
		var m map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(stdout)), &m) != nil {
			return nil
		}
		return []map[string]any{m}
	default:
		return nil
	}
}

// lookup walks a dotted path through nested objects, returning every value the
// path matches. A `*` segment matches every key at that level, which is how a
// per-model usage map keyed by the model's own name is read -- the key is the
// model id, so it cannot be written into a static path.
func lookup(obj map[string]any, path string) []any {
	if path == "" {
		return nil
	}
	cur := []any{obj}
	for _, key := range strings.Split(path, ".") {
		var next []any
		for _, c := range cur {
			m, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if key == "*" {
				for _, v := range m {
					next = append(next, v)
				}
				continue
			}
			if v, ok := m[key]; ok {
				next = append(next, v)
			}
		}
		if len(next) == 0 {
			return nil
		}
		cur = next
	}
	return cur
}

// lastString returns the final string value at path. Later objects win, and
// within one object the last match does -- a wildcard is not meaningful for the
// reply text, so there is nothing to combine.
func lastString(objs []map[string]any, path string) (string, bool) {
	out, found := "", false
	for _, o := range objs {
		for _, v := range lookup(o, path) {
			if s, ok := v.(string); ok {
				out, found = s, true
			}
		}
	}
	return out, found
}

// lastFloat returns the numeric value at path: SUMMED across a wildcard's matches
// within one object, and taking the last object that matched at all.
//
// The two rules cover the two shapes a CLI reports in. A wildcard expands to one
// entry per model used, and a session that switched models spent both, so those
// add. Successive objects are a stream's running state for the SAME session, where
// the newest line supersedes the last -- adding those would count every earlier
// partial again.
func lastFloat(objs []map[string]any, path string) (float64, bool) {
	out, found := 0.0, false
	for _, o := range objs {
		sum, matched := 0.0, false
		for _, v := range lookup(o, path) {
			if f, ok := v.(float64); ok {
				sum, matched = sum+f, true
			}
		}
		if matched {
			out, found = sum, true
		}
	}
	return out, found
}

func lastInt(objs []map[string]any, path string) int {
	f, _ := lastFloat(objs, path)
	return int(f)
}
