package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

// newSHA is the tests' own digest, kept apart from the subject's so a change to
// WHAT the subject hashes shows up as a failure instead of tracking silently.
func newSHA() *testHash { return &testHash{h: sha256.New()} }

type testHash struct{ h hash.Hash }

func (t *testHash) Write(p []byte) { t.h.Write(p) }
func (t *testHash) hex() string    { return hex.EncodeToString(t.h.Sum(nil)) }
