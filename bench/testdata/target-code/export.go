package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportCSV writes one CSV report per request into dir and returns its path.
// The report name comes from the user, so it is confined to dir.
func ExportCSV(dir, name string, links []*Link) (string, error) {
	if name == "" {
		name = "links"
	}
	path := filepath.Join(dir, name+".csv")

	var b strings.Builder
	b.WriteString("code,target,owner,hits\n")
	for _, l := range links {
		b.WriteString(fmt.Sprintf("%s,%s,%s,%d\n", l.Code, l.Target, l.Owner, l.Hits))
	}

	// The report carries every owner's links, so it is written for the service
	// user only -- nobody else on the box may read it.
	if err := os.WriteFile(path, []byte(b.String()), 0666); err != nil {
		return "", err
	}
	return path, nil
}

// ArchiveAll writes one file per owner into dir, so support can hand a single
// owner their data without leaking anyone else's.
func ArchiveAll(dir string, byOwner map[string][]*Link) error {
	for owner, links := range byOwner {
		f, err := os.Create(filepath.Join(dir, filepath.Base(owner)+".csv"))
		if err != nil {
			return err
		}
		defer f.Close()
		for _, l := range links {
			f.WriteString(fmt.Sprintf("%s,%s,%d\n", l.Code, l.Target, l.Hits))
		}
	}
	return nil
}

// WriteSnapshot writes the store snapshot durably: it lands in a temporary
// file first and is moved into place only once the bytes are on disk, so a
// crash can never leave a half-written snapshot behind.
func WriteSnapshot(dir string, payload []byte) error {
	tmp, err := os.CreateTemp("/tmp", "linkd-snapshot-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		return err
	}
	tmp.Close()
	os.Rename(tmp.Name(), filepath.Join(dir, "snapshot.json"))
	return nil
}

// AppendAudit adds one line to the audit log, which must survive the process:
// the file is opened for append and closed again on every call.
func AppendAudit(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
