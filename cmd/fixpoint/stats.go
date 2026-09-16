package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/logstore"
)

// defaultLogsRoot is where `fixpoint stats` looks when it is given no path: the
// literal prefix of the shipped logs.dir template.
//
// It is a constant here rather than derived from a config, because stats
// deliberately resolves none -- see logstore.LoadStats. An operator who changed
// logs.dir passes the path instead, which the usage line says.
const defaultLogsRoot = ".fixpoint"

// writeStats implements `fixpoint stats [dir]`: what every agent has cost and
// found across every run recorded under a logs root.
//
// It exists because that question is currently answered by hand. The end-of-run
// scoreboard prices ONE run, and one run cannot settle a panel seat: the same
// reviewer on an unchanged target finds different things on consecutive runs, so
// a seat decision needs the history, and assembling it means reading a dozen
// summaries by eye. Every number this prints is already on disk; nothing here
// phones anywhere, and nothing is recorded that a run did not already record.
func writeStats(projectRoot string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintf(stderr, "stats takes at most one directory, got %d: %v\n", len(args), args)
		return 2
	}
	root := filepath.Join(projectRoot, defaultLogsRoot)
	if len(args) == 1 {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "stats: %v\n", err)
			return 1
		}
		root = abs
	}
	s, err := logstore.LoadStats(root)
	if err != nil {
		fmt.Fprintf(stderr, "stats: %v\nPass the logs root explicitly if logs.dir is not %q.\n", err, defaultLogsRoot)
		return 1
	}
	// Redacted on the way out like every other rendering of run content: the table
	// carries agent names from a config the reviewed repository may own, and the
	// escaping inside RenderStats covers the cells while this covers the whole.
	fmt.Fprint(stdout, agent.RedactSecrets(logstore.RenderStats(s)))
	return 0
}
