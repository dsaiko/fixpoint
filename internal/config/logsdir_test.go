package config

import (
	"path/filepath"
	"testing"
)

// The three paths derived from a logs.dir template must agree with the layout the
// template describes: the literal prefix is the git/collection exclusion, the
// run part holds the summary and is the atomically-claimed unit, and the {round}
// part onward is the per-round directory.
func TestLogsDirDerivedPaths(t *testing.T) {
	const ts = "20260724-120000"
	cases := []struct {
		name      string
		dir       string
		base      string // StaticBase
		run       string // RunPath
		round2    string // RoundPath for round 2, beneath run
		roundIsRu bool   // RoundPath returns the run dir unchanged
	}{
		{
			name:   "default layout",
			dir:    "logs/{timestamp}/round-{round}",
			base:   "logs",
			run:    filepath.Join("logs", ts),
			round2: filepath.Join("logs", ts, "round-2"),
		},
		{
			name:      "no round segment puts every round in the run dir",
			dir:       "logs/{timestamp}",
			base:      "logs",
			run:       filepath.Join("logs", ts),
			roundIsRu: true,
		},
		{
			name:   "placeholders may share a segment with literals",
			dir:    "artifacts/run-{timestamp}/r{round}",
			base:   "artifacts",
			run:    filepath.Join("artifacts", "run-"+ts),
			round2: filepath.Join("artifacts", "run-"+ts, "r2"),
		},
		{
			name:   "deep literal prefix is kept whole",
			dir:    "var/log/fixpoint/{timestamp}/round-{round}",
			base:   filepath.Join("var", "log", "fixpoint"),
			run:    filepath.Join("var", "log", "fixpoint", ts),
			round2: filepath.Join("var", "log", "fixpoint", ts, "round-2"),
		},
		{
			// strings.Join keeps the leading empty segment that filepath.Join would
			// drop, so an absolute template does not silently become relative.
			name:   "absolute template stays absolute",
			dir:    "/var/log/fixpoint/{timestamp}/round-{round}",
			base:   filepath.FromSlash("/var/log/fixpoint"),
			run:    filepath.FromSlash("/var/log/fixpoint/" + ts),
			round2: filepath.FromSlash("/var/log/fixpoint/" + ts + "/round-2"),
		},
		{
			// The intermediate "rounds" literal belongs to the per-round side: the
			// summary stays at the run root rather than in a rounds-only directory.
			name:   "round may nest deeper than one segment",
			dir:    "logs/{timestamp}/rounds/{round}",
			base:   "logs",
			run:    filepath.Join("logs", ts),
			round2: filepath.Join("logs", ts, "rounds", "2"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Logs{Dir: tc.dir}
			if got := l.StaticBase(); got != tc.base {
				t.Errorf("StaticBase() = %q, want %q", got, tc.base)
			}
			run := l.RunPath(ts)
			if run != tc.run {
				t.Errorf("RunPath() = %q, want %q", run, tc.run)
			}
			// RoundPath renders beneath the *claimed* run dir, which may carry a
			// collision suffix, so pass a stand-in rather than re-rendering.
			wantRound := tc.round2
			if tc.roundIsRu {
				wantRound = run
			}
			if got := l.RoundPath(run, 2, ts); got != wantRound {
				t.Errorf("RoundPath() = %q, want %q", got, wantRound)
			}
		})
	}
}

// The {timestamp} in logs.dir is the run's start time, so it must be identical
// for every round -- otherwise a long run scatters its rounds across sibling
// directories and the summary lands away from the artifacts it describes.
func TestLogsDirTimestampIsStablePerRun(t *testing.T) {
	l := Logs{Dir: "logs/{timestamp}/round-{round}"}
	run := l.RunPath("20260724-120000")
	for _, round := range []int{1, 2, 7} {
		got := l.RoundPath(run, round, "20260724-120000")
		if filepath.Dir(got) != run {
			t.Errorf("round %d dir %q is not under the run dir %q", round, got, run)
		}
	}
}
