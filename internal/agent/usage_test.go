package agent

import (
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

// claudeUsage mirrors config/agents/claude.yaml. Keeping the shipped paths in the
// test is the point: a typo there is a silent accounting failure, not a crash.
var claudeUsage = config.AgentUsage{
	Format:           config.UsageFormatJSON,
	Text:             "result",
	InputTokens:      "modelUsage.*.inputTokens",
	OutputTokens:     "modelUsage.*.outputTokens",
	CacheReadTokens:  "modelUsage.*.cacheReadInputTokens",
	CacheWriteTokens: "modelUsage.*.cacheCreationInputTokens",
	CostUSD:          "modelUsage.*.costUSD",
}

var codexUsage = config.AgentUsage{
	Format:          config.UsageFormatJSONL,
	Text:            "item.text",
	InputTokens:     "usage.input_tokens",
	OutputTokens:    "usage.output_tokens",
	CacheReadTokens: "usage.cached_input_tokens",
}

// A verbatim envelope from `claude -p --output-format json`, trimmed to the fields
// the paths read. The numbers are the real ones from a one-word probe, and they are
// why this feature exists: ~30 bytes crossed fixpoint's boundary while the session
// actually consumed 21,072 tokens and six cents.
const claudeEnvelope = `{"is_error":false,"num_turns":1,"total_cost_usd":0.061299,
"usage":{"input_tokens":2,"output_tokens":4,"cache_read_input_tokens":15738,"cache_creation_input_tokens":5332},
"modelUsage":{"claude-opus-5":{"inputTokens":2,"outputTokens":4,"cacheReadInputTokens":15738,
"cacheCreationInputTokens":5332,"costUSD":0.061299,"canonicalModel":"claude-opus-5"}},
"result":"I reviewed the code.\n<review>{\"findings\":[]}</review>\n","type":"result"}`

func TestParseUsageUnwrapsClaudeEnvelope(t *testing.T) {
	text, u := ParseUsage(claudeUsage, claudeEnvelope)

	// The reply must come out ready for the contract extractor: if the envelope
	// reached it instead, its escaped quotes would fail to parse and the round
	// would report nothing.
	if !strings.Contains(text, "<review>") || strings.Contains(text, "modelUsage") {
		t.Fatalf("text not unwrapped from the envelope:\n%q", text)
	}
	var out struct {
		Findings []struct{} `json:"findings"`
	}
	if err := ExtractJSON(text, "review", &out); err != nil {
		t.Errorf("unwrapped text is not extractable: %v", err)
	}

	if u.InputTokens != 2 || u.OutputTokens != 4 {
		t.Errorf("in/out tokens = %d/%d, want 2/4", u.InputTokens, u.OutputTokens)
	}
	if u.CacheReadTokens != 15738 || u.CacheWriteTokens != 5332 {
		t.Errorf("cache read/write = %d/%d, want 15738/5332", u.CacheReadTokens, u.CacheWriteTokens)
	}
	// Cache dominates: excluding it would report 6 tokens for a 21,072-token call.
	if got := u.Tokens(); got != 21076 {
		t.Errorf("Tokens() = %d, want 21076 (cache included)", got)
	}
	if !u.CostKnown || u.CostUSD != 0.061299 {
		t.Errorf("cost = %v (known=%v), want 0.061299", u.CostUSD, u.CostKnown)
	}
}

// codex streams events; the totals land on the last line and the reply on an
// earlier one. Verbatim shape from `codex exec --json`.
func TestParseUsageReadsCodexJSONLStream(t *testing.T) {
	stream := `{"type":"thread.started","thread_id":"019f"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"<review>{\"findings\":[]}</review>"}}
{"type":"turn.completed","usage":{"input_tokens":13854,"cached_input_tokens":11008,"cache_write_input_tokens":0,"output_tokens":5}}`

	text, u := ParseUsage(codexUsage, stream)
	if !strings.Contains(text, "<review>") {
		t.Fatalf("reply not extracted from the stream:\n%q", text)
	}
	if u.InputTokens != 13854 || u.OutputTokens != 5 || u.CacheReadTokens != 11008 {
		t.Errorf("usage = %+v, want 13854 in / 5 out / 11008 cached", u)
	}
	// codex reports no price, so the scoreboard must show nothing rather than $0.
	if u.CostKnown {
		t.Error("CostKnown is true for a CLI that reports no cost; the run total would silently understate")
	}
}

// A stream reports running totals for ONE session, so the newest line supersedes
// the last. Adding them would count every earlier partial again.
func TestParseUsageStreamSupersedesRatherThanAccumulates(t *testing.T) {
	stream := `{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":10}}
{"type":"turn.completed","usage":{"input_tokens":250,"output_tokens":30}}`
	_, u := ParseUsage(codexUsage, stream)
	if u.InputTokens != 250 || u.OutputTokens != 30 {
		t.Errorf("usage = %+v, want the final totals 250/30, not the sum", u)
	}
}

// A wildcard expands to one entry per model, and a session that used two spent
// both — so those DO add, unlike successive stream lines.
func TestParseUsageSumsAcrossModelsInOneObject(t *testing.T) {
	env := `{"result":"hi","modelUsage":{
		"claude-opus-5":{"inputTokens":100,"outputTokens":10,"costUSD":0.5},
		"claude-haiku-4-5":{"inputTokens":40,"outputTokens":4,"costUSD":0.1}}}`
	_, u := ParseUsage(claudeUsage, env)
	if u.InputTokens != 140 || u.OutputTokens != 14 {
		t.Errorf("usage = %+v, want 140/14 summed across both models", u)
	}
	if got := u.CostUSD; got < 0.599 || got > 0.601 {
		t.Errorf("cost = %v, want 0.6 summed across both models", got)
	}
}

// The reply matters more than the accounting: a CLI that died mid-write, errored,
// or changed its output shape must not cost the round its findings.
func TestParseUsageFailsOpenOnUnparseableOutput(t *testing.T) {
	for name, out := range map[string]string{
		"not json":  "boom: the CLI crashed\n<review>{\"findings\":[]}</review>",
		"truncated": `{"result":"<review>{}</review>","modelUsage":{"claude`,
		"empty":     "",
	} {
		t.Run(name, func(t *testing.T) {
			text, u := ParseUsage(claudeUsage, out)
			if text != out {
				t.Errorf("output was altered on a parse failure:\ngot  %q\nwant %q", text, out)
			}
			if u.Tokens() != 0 || u.CostKnown {
				t.Errorf("usage = %+v, want zero when nothing could be read", u)
			}
		})
	}
}

// An agent with no usage block is the default and must pass through untouched.
func TestParseUsageDisabledPassesOutputThrough(t *testing.T) {
	const raw = "plain text <review>{}</review>"
	text, u := ParseUsage(config.AgentUsage{}, raw)
	if text != raw {
		t.Errorf("text = %q, want the raw output unchanged", text)
	}
	if u.Tokens() != 0 || u.CostKnown {
		t.Errorf("usage = %+v, want zero", u)
	}
}

// A path that does not match must not silently substitute a wrong number, and must
// not destroy the reply either.
func TestParseUsageIgnoresPathsThatDoNotMatch(t *testing.T) {
	u := config.AgentUsage{
		Format: config.UsageFormatJSON, Text: "result",
		InputTokens: "nope.missing", CostUSD: "also.missing",
	}
	text, got := ParseUsage(u, `{"result":"hello"}`)
	if text != "hello" {
		t.Errorf("text = %q, want hello", text)
	}
	if got.InputTokens != 0 || got.CostKnown {
		t.Errorf("usage = %+v, want zero for unmatched paths", got)
	}
}

// A CLI reporting a genuine zero cost (a local model) is different from one that
// reports no cost at all, and the scoreboard renders them differently.
func TestParseUsageDistinguishesZeroCostFromNoCost(t *testing.T) {
	_, zero := ParseUsage(claudeUsage, `{"result":"x","modelUsage":{"local":{"costUSD":0}}}`)
	if !zero.CostKnown || zero.CostUSD != 0 {
		t.Errorf("a reported $0 must be known-zero, got %+v", zero)
	}
	_, none := ParseUsage(codexUsage, `{"item":{"text":"x"},"usage":{"input_tokens":5}}`)
	if none.CostKnown {
		t.Error("a CLI that reports no cost must leave CostKnown false")
	}
}
