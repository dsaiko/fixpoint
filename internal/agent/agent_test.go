package agent

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		wantErr bool
		wantN   int // findings count
	}{
		{"plain", `noise <review>{"findings":[]}</review>`, false, 0},
		{"one finding", `<review>{"findings":[{"title":"x"}]}</review>`, false, 1},
		{"fenced json inside tags", "<review>```json\n{\"findings\":[{\"title\":\"x\"}]}\n```</review>", false, 1},
		{"last block wins", `<review>{"findings":[{"title":"a"},{"title":"b"}]}</review> later <review>{"findings":[{"title":"x"}]}</review>`, false, 1},
		{"missing tags", `{"findings":[]}`, true, 0},
		{"invalid json", `<review>{"findings":}</review>`, true, 0},
		// A payload without an explicit findings array must not read as a
		// clean review.
		{"empty object payload", `<review>{}</review>`, true, 0},
		{"null payload", `<review>null</review>`, true, 0},
		{"null findings", `<review>{"findings":null}</review>`, true, 0},
		{"leading chatter, block last", "I looked at the code.\n<review>\n{\"findings\": []}\n</review>\n", false, 0},
		// The contract requires the block to be the LAST thing printed, so only
		// whitespace may follow it. Untagged output after a (here, clean) block
		// means the real final response was not tagged: reject rather than accept
		// the earlier block, which would otherwise be a false convergence.
		{"untagged trailing output rejected",
			"<review>{\"findings\":[]}</review>\nSorry, I ran out of time before finishing.",
			true, 0},
		// Regression: an agent reviewing THIS codebase quotes literal tags
		// inside JSON strings; a naive non-greedy match pairs the wrong tags
		// and yields truncated JSON.
		{"tags quoted inside payload strings",
			`<review>{"findings":[{"title":"ExtractJSON mishandles the last <review>...</review> block","description":"quoting </review> inside a string"}]}</review>`,
			false, 1},
		// The echoed prompt carries the contract's example block; the real
		// block printed later must win.
		{"echoed contract example before real block",
			"user prompt echo: end with <review>\n{\"findings\": [{\"title\": \"example\"} , {\"title\": \"example2\"}]}\n</review> and nothing else.\nmodel output:\n<review>{\"findings\":[]}</review>",
			false, 0},
		// A malformed FINAL block must not fall back to an earlier valid block
		// echoed/quoted from the prompt or the reviewed source: an embedded clean
		// review would otherwise turn a failed reviewer into a false convergence.
		{"malformed final block does not resurrect an earlier clean block",
			"quoted source line: <review>{\"findings\":[]}</review>\nmodel output:\n<review>{\"findings\": BROKEN}</review>",
			true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out model.ReviewOutput
			err := ExtractJSON(tc.output, "review", &out)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if err == nil && len(out.Findings) != tc.wantN {
				t.Fatalf("findings = %d, want %d", len(out.Findings), tc.wantN)
			}
		})
	}
}

// Candidate-pair parsing is quadratic in the number of tags; a hostile agent
// emitting thousands of pairs must hit the attempt cap instead of stalling
// the orchestrator.
func TestExtractJSONBoundedAttempts(t *testing.T) {
	hostile := strings.Repeat(`<review>{"findings":}</review>`, 500)
	var out model.ReviewOutput
	err := ExtractJSON(hostile, "review", &out)
	if err == nil || !strings.Contains(err.Error(), "candidate blocks") {
		t.Fatalf("err = %v, want attempt-cap error", err)
	}
	// A valid last block among earlier garbage must still be found.
	if err := ExtractJSON(hostile+`<review>{"findings":[]}</review>`, "review", &out); err != nil {
		t.Fatalf("valid last block after garbage: %v", err)
	}
}

func TestExtractJSONFix(t *testing.T) {
	var out model.FixOutput
	err := ExtractJSON(`<fix>{"results":[{"id":"r1.1","verdict":"fixed","detail":"d"}]}</fix>`, "fix", &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].Verdict != "fixed" {
		t.Fatalf("unexpected results: %+v", out.Results)
	}
}

// Regression: each candidate opening under the final close tag must be decoded
// into a FRESH value, so a nearer candidate's partial decode cannot leak into
// out. FixOutput has no custom UnmarshalJSON, so the decoder populates Results
// before failing on the trailing garbage; because every candidate here is
// malformed, ExtractJSON must fail closed AND leave out untouched (never carry
// the partial Results it decoded into a discarded fresh value).
func TestExtractJSONFreshValuePerCandidate(t *testing.T) {
	// Two openings under one final close, both malformed: the nearest partially
	// decodes Results before breaking on " BAD", the furthest swallows the inner
	// <fix> literal and breaks too. Neither may reach out.
	output := `<fix>{"results":[{"id":"r1.1","verdict":"fixed"}] BAD <fix>{"results":[{"id":"r1.1","verdict":"fixed"}] BAD</fix>`
	var out model.FixOutput
	if err := ExtractJSON(output, "fix", &out); err == nil {
		t.Fatal("ExtractJSON = nil, want an error for an all-malformed final block")
	}
	if len(out.Results) != 0 {
		t.Fatalf("out inherited stale partial state from a discarded candidate: %+v", out.Results)
	}
}

// ExtractJSON must reject a nil or non-pointer out rather than panicking in
// reflect.
func TestExtractJSONRejectsBadOut(t *testing.T) {
	var out model.FixOutput
	if err := ExtractJSON(`<fix>{}</fix>`, "fix", out); err == nil { // pass by value, not pointer
		t.Error("ExtractJSON(non-pointer) = nil, want error")
	}
	var nilPtr *model.FixOutput
	if err := ExtractJSON(`<fix>{}</fix>`, "fix", nilPtr); err == nil {
		t.Error("ExtractJSON(nil pointer) = nil, want error")
	}
}

func TestRawRedactsSecrets(t *testing.T) {
	// SYNTHETIC FIXTURES -- none of these is (or ever was) a real credential.
	// The values are keyboard sequences (ABCDEF..., 1234...) shaped like real
	// provider tokens, because matching those exact shapes is what redactRules
	// is for. Each provider-shaped token is assembled from two source literals:
	// the compiler folds them into the contiguous token the rule must match,
	// while secret scanners (which scan source text) no longer see a
	// credential-shaped literal and so don't page the repo owner about it.
	const (
		fakeAnthropicKey  = "sk-ant" + "-api03-ABCDEFGHIJKLMNOP"
		fakeOpenAIProjKey = "sk-" + "proj-NOTAREALKEYabcdefghijklmno"
		fakeOpenAIKey     = "sk-" + "abcdefghijklmnopqrstuvwxyz01"
		fakeGithubToken   = "ghp" + "_ABCDEFGHIJKLMNOPQRSTUVWXYZ0"
		fakeGithubPAT     = "github_pat" + "_ABCDEFGHIJ1234567890abcd"
		fakeSlackToken    = "xoxb" + "-1234567890-abcdef"
		fakeAWSKey        = "AKIA" + "1234567890ABCDEF"
		fakeGoogleKey     = "AIza" + "SyABCDEFGHIJKLMNOPQRSTUVWXYZ0123456"
		fakeStripeKey     = "sk_" + "live_ABCDEFGHIJKLMNOPQRSTUVWX"
		fakeJWT           = "eyJ" + "hbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefGHIJKLmnop"
	)
	// A synthetic PEM private key block: BEGIN/END markers with a body the rule
	// must mask in full. Assembled so scanners see no contiguous key literal.
	fakePEM := "-----BEGIN RSA " + "PRIVATE KEY-----\nMIIBOgIBAAJBAKj34GkxFhD\nAQABAkEA\n-----END RSA " + "PRIVATE KEY-----"
	cases := []struct {
		name   string
		stderr string
		secret string // must NOT appear in the redacted output ("" = nothing masked)
		keep   string // must still appear
	}{
		// One case per redactRules entry, so a rule that silently fails to
		// compile or match is caught. Each asserts both that the secret is gone
		// AND that the [REDACTED] mask is present (masking, not elision).
		{"authorization header", "401: Authorization: Token abc123DEF456ghi789xyz\n", "abc123DEF456ghi789xyz", "Authorization:"},
		{"standalone bearer", "got bearer abcdefghijklmnopqrstuvwx\n", "abcdefghijklmnopqrstuvwx", "bearer"},
		{"anthropic key", "using key " + fakeAnthropicKey + "\n", fakeAnthropicKey, "using key"},
		{"openai project key", "key " + fakeOpenAIProjKey + "\n", fakeOpenAIProjKey, "key"},
		{"openai key", "key " + fakeOpenAIKey + "\n", fakeOpenAIKey, "key"},
		{"github token", "remote " + fakeGithubToken + "\n", fakeGithubToken, "remote"},
		{"github pat", "pat " + fakeGithubPAT + "\n", fakeGithubPAT, "pat"},
		{"slack token", "using " + fakeSlackToken + "\n", fakeSlackToken, "using"},
		{"aws access key", "aws " + fakeAWSKey + "\n", fakeAWSKey, "aws"},
		{"google api key", "google " + fakeGoogleKey + "\n", fakeGoogleKey, "google"},
		{"stripe key", "stripe " + fakeStripeKey + "\n", fakeStripeKey, "stripe"},
		{"jwt", "auth " + fakeJWT + "\n", fakeJWT, "auth"},
		{"pem private key", "loaded\n" + fakePEM + "\ndone\n", fakePEM, "loaded"},
		{"uri password", "conn postgres://app:s3cr3tpw@dbhost:5432/db\n", "s3cr3tpw", "postgres://"},
		{"uri password empty user", "redis://:mypassword123@cache:6379\n", "mypassword123", "redis://"},
		{"generic api_key", "API_KEY=supersecretvalue123\n", "supersecretvalue123", "API_KEY="},
		{"generic secret", "secret: mytopsecretval\n", "mytopsecretval", "secret"},
		{"generic token assignment", "token=abcd1234efgh\n", "abcd1234efgh", "token="},
		{"generic password", `password="hunter2secret"` + "\n", "hunter2secret", "password"},
		{"generic passwd", "passwd = supersecretpw\n", "supersecretpw", "passwd"},
		// Quoted-key (JSON) forms: a reviewed diff touching a config file carries
		// the credential this way, and an ordinary password matches none of the
		// shape-based rules above, so these are the only rule that can catch it.
		{"json quoted key", `{"password": "hunter2secret"}`, "hunter2secret", `"password"`},
		{"json quoted key no space", `{"api_key":"supersecretvalue123"}`, "supersecretvalue123", `"api_key"`},
		{"json quoted key escaped", `"detail": "{\"password\": \"hunter2secret\"}"`, "hunter2secret", `\"password\"`},
		{"json single quoted key", `{'secret': 'mytopsecretval'}`, "mytopsecretval", `'secret'`},
		{"plain diagnostic untouched", "connection refused after 3 retries\n", "", "connection refused after 3 retries"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := Result{Stdout: "ok", Stderr: tc.stderr}.Raw([]string{"agent"})
			if tc.secret != "" {
				if strings.Contains(out, tc.secret) {
					t.Errorf("secret %q not redacted from raw output:\n%s", tc.secret, out)
				}
				if !strings.Contains(out, redactionMask) {
					t.Errorf("expected the %s mask (rule elided instead of masking?):\n%s", redactionMask, out)
				}
			}
			if !strings.Contains(out, tc.keep) {
				t.Errorf("expected %q to survive redaction:\n%s", tc.keep, out)
			}
		})
	}
}

// The generic key/token/password rule deliberately excludes the backslash from
// its value class so that when RedactSecrets runs over already-serialized JSON
// (the .json step log and the summary), masking a secret cannot consume the "\"
// of a trailing \" and unbalance the surrounding string quote. This invariant is
// documented on the rule but only exercised end-to-end in the logstore package;
// assert it directly here so a future regex tweak that drops the exclusion is
// caught in the package that owns the rule.
func TestRedactSecretsPreservesJSONEscaping(t *testing.T) {
	// A secret whose value ENDS on a quote: once serialized, that quote becomes
	// the escape \" the redaction must not eat.
	payload := map[string]string{"detail": `token=abcd1234efgh5678"`}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	out := RedactSecrets(string(b))
	if strings.Contains(out, "abcd1234efgh5678") {
		t.Errorf("secret not masked in serialized JSON:\n%s", out)
	}
	if !strings.Contains(out, redactionMask) {
		t.Errorf("expected the %s mask:\n%s", redactionMask, out)
	}
	var back map[string]string
	if err := json.Unmarshal([]byte(out), &back); err != nil {
		t.Fatalf("redacted JSON no longer parses (escaping corrupted?): %v\n%s", err, out)
	}
	if !strings.HasSuffix(back["detail"], `"`) {
		t.Errorf("the trailing quote was eaten by redaction: %q", back["detail"])
	}
}

// The built-in rules are shapes, so a site's own opaque token is exactly what
// they cannot recognize; logs.redact is the operator's answer and must reach
// every persisted format, not just Raw. Both spellings are asserted: a bare
// pattern masks its whole match, and a pattern with a capture group keeps the
// visible prefix -- the convention the built-in rules use, and the one that keeps
// the surrounding log line readable.
func TestSetExtraRedactions(t *testing.T) {
	t.Cleanup(func() { SetExtraRedactions(nil) })
	SetExtraRedactions([]*regexp.Regexp{
		regexp.MustCompile(`(?i)(x-internal-auth\s*:\s*)\S+`),
		regexp.MustCompile(`ACME-[A-Z0-9]{8}`),
	})
	// A shape no built-in rule matches: an opaque token under a header name that
	// is not "authorization", and a bare site-specific key with no keyword at all.
	const line = "x-internal-auth: opaqueTOKENvalue and ACME-ABCD1234 seen\n"
	if got := RedactSecrets(line); strings.Contains(got, "opaqueTOKENvalue") ||
		strings.Contains(got, "ACME-ABCD1234") ||
		!strings.Contains(got, "x-internal-auth: ") ||
		!strings.Contains(got, redactionMask) {
		t.Errorf("RedactSecrets with extra patterns = %q; want both secrets masked and the header name kept", got)
	}
	// Raw shares the one pass, so the .raw log -- the artifact that keeps the
	// subprocess's verbatim stderr, where a leaked token actually lands -- is
	// covered by the same install.
	raw := Result{Stdout: "ok", Stderr: line}.Raw([]string{"agent"})
	if strings.Contains(raw, "opaqueTOKENvalue") {
		t.Errorf("Raw() = %q; want the extra patterns applied to the raw log too", raw)
	}
	// Uninstalling restores the built-in-only behavior, so one test's patterns
	// cannot leak into another's expectations.
	SetExtraRedactions(nil)
	if got := RedactSecrets(line); !strings.Contains(got, "opaqueTOKENvalue") {
		t.Errorf("RedactSecrets after clearing = %q; want the extra patterns gone", got)
	}
}

// An agent whose argv is empty (a blank pool entry that slipped past config
// validation) must fail the single invocation with a descriptive error rather
// than panicking on argv[0] or starting a process.
func TestRunEmptyArgv(t *testing.T) {
	res := Run(t.Context(), config.Agent{Command: nil, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}, "prompt", t.TempDir())
	if res.Err == nil || !strings.Contains(res.Err.Error(), "empty argv") {
		t.Fatalf("Run() err = %v, want a descriptive empty-argv error", res.Err)
	}
	if res.Stdout != "" || res.Stderr != "" {
		t.Errorf("empty-argv Run started a process: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
}

// A Run whose PromptVia is empty or unrecognized must fail the invocation with
// a descriptive error rather than silently starting the agent with no prompt
// delivered -- mirroring the empty-argv guard. Both branches (append-to-argv and
// set-stdin) key off PromptVia, so any other value delivers nothing.
func TestRunUnsupportedPromptVia(t *testing.T) {
	for _, via := range []string{"", "env"} {
		a := config.Agent{
			Command:   []string{script(t, `cat; printf '%s' "$1"`)},
			PromptVia: via,
			Timeout:   config.Duration(time.Minute),
		}
		res := Run(t.Context(), a, "the prompt", t.TempDir())
		if res.Err == nil || !strings.Contains(res.Err.Error(), "prompt_via") {
			t.Fatalf("PromptVia=%q: Run() err = %v, want a descriptive prompt_via error", via, res.Err)
		}
		if res.Stdout != "" || res.Stderr != "" {
			t.Errorf("PromptVia=%q: Run started a process: stdout=%q stderr=%q", via, res.Stdout, res.Stderr)
		}
	}
}

// BoundedBuffer must always report the full input length, even when it drops
// bytes past its cap: a short-write return makes os/exec's io.Copy fail with
// io.ErrShortWrite, turning oversized subprocess output into an error instead of
// returning it truncated. The marker must appear only when bytes were actually
// dropped -- writing exactly maxBytes bytes fills the buffer without truncating.
// HumanSize's contract is that the number it prints tracks the byte cap it is
// derived from (coarse MB/KB/bytes), so the truncation markers can never lie
// about where output was cut. Pin the unit choices and the exact production caps.
func TestHumanSize(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{300_000, "300 KB"},
		{10 << 20, "10 MB"},
		{4 << 20, "4 MB"},
		{maxOutput, "10 MB"},
		{999, "999 bytes"},
		{1_000, "1 KB"},
		{1_000_000, "1 MB"},
	}
	for _, tc := range cases {
		if got := HumanSize(tc.n); got != tc.want {
			t.Errorf("HumanSize(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// TruncationMarker must embed the HumanSize of its cap, so an operator reading a
// truncated log sees the real limit rather than a stale hardcoded figure. A
// regression pinning the wrong unit (or dropping the size entirely) breaks this.
func TestTruncationMarker(t *testing.T) {
	for _, n := range []int{maxOutput, 4 << 20, 300_000} {
		m := TruncationMarker(n)
		if !strings.Contains(m, HumanSize(n)) {
			t.Errorf("TruncationMarker(%d) = %q, missing size %q", n, m, HumanSize(n))
		}
		if !strings.Contains(m, "truncated") {
			t.Errorf("TruncationMarker(%d) = %q, missing 'truncated'", n, m)
		}
	}
}

func TestBoundedBuffer(t *testing.T) {
	const maxBytes = 1 << 20
	const marker = "\n[... truncated ...]"
	cases := []struct {
		name      string
		size      int
		truncated bool
	}{
		{"below cap", maxBytes - 1, false},
		{"at cap", maxBytes, false},
		{"above cap", maxBytes + 4096, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBoundedBuffer(maxBytes, marker)
			n, err := b.Write(make([]byte, tc.size))
			if n != tc.size || err != nil {
				t.Fatalf("Write returned (%d, %v), want (%d, nil)", n, err, tc.size)
			}
			s := b.String()
			if got := strings.Contains(s, marker); got != tc.truncated {
				t.Errorf("truncation marker present = %v, want %v", got, tc.truncated)
			}
			if len(s) > maxBytes+len(marker) {
				t.Fatalf("buffer exceeded cap: %d", len(s))
			}
		})
	}
}

// A single oversized Write always takes the len(p) > room branch, so the
// already-full else branch (room <= 0) -- the path exec's copy goroutine hits on
// every write after the cap is reached -- is never exercised by TestBoundedBuffer.
// Fill the buffer with one Write(limit), then a second Write must still report
// n==len(p) with nil error (so io.Copy never sees a short write), append the
// marker exactly once, and never grow past limit+len(marker).
func TestBoundedBufferWriteAfterFull(t *testing.T) {
	const limit = 1 << 20
	const marker = "\n[... truncated ...]"
	b := NewBoundedBuffer(limit, marker)
	if n, err := b.Write(make([]byte, limit)); n != limit || err != nil {
		t.Fatalf("first Write returned (%d, %v), want (%d, nil)", n, err, limit)
	}
	second := make([]byte, 4096)
	if n, err := b.Write(second); n != len(second) || err != nil {
		t.Fatalf("Write after full returned (%d, %v), want (%d, nil)", n, err, len(second))
	}
	s := b.String()
	if got := strings.Count(s, marker); got != 1 {
		t.Errorf("marker appears %d times, want exactly 1:\n%q", got, s[len(s)-len(marker)*2:])
	}
	if len(s) > limit+len(marker) {
		t.Fatalf("buffer exceeded cap: %d > %d", len(s), limit+len(marker))
	}
}

// The BoundedBuffer mutex guards a copy goroutine still writing while a reader
// calls String. Supervise waits for its copy goroutines before returning, so the
// runners in this program no longer produce that overlap -- but the buffer is
// exported and the guarantee is not local to it, and a raced strings.Builder
// garbles output or panics rather than failing visibly. Sequential coverage
// (TestBoundedBuffer) never overlaps the two, so a removal of the locking would
// still pass it; this test overlaps Write and String and must be run under -race
// to be meaningful (the subprocess side is exercised by
// TestRunDrainGraceBoundsLeakedPipe in guard_test.go).
func TestBoundedBufferConcurrentWriteString(t *testing.T) {
	b := NewBoundedBuffer(maxOutput, "\n[... truncated ...]")
	chunk := []byte(strings.Repeat("x", 4096))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 2000 {
			if _, err := b.Write(chunk); err != nil {
				t.Errorf("Write: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range 2000 {
			_ = b.String()
		}
	}()
	wg.Wait()
}
