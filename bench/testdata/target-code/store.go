package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// Link is one shortened URL with its bookkeeping.
type Link struct {
	Code      string
	Target    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Hits      int64
	Owner     string
}

var ErrNotFound = errors.New("link not found")

// Store keeps links in memory. It is the only owner of the links map;
// callers go through its methods.
type Store struct {
	links map[string]*Link
	quota map[string]int
}

func NewStore() *Store {
	return &Store{
		links: make(map[string]*Link),
	}
}

// Create registers a new short link for target, owned by owner, valid for ttl.
func (s *Store) Create(target, owner string, ttl time.Duration) (*Link, error) {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return nil, errors.New("target must be an absolute http(s) URL")
	}
	code, err := newCode()
	if err != nil {
		return nil, err
	}
	l := &Link{
		Code:      code,
		Target:    target,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Owner:     owner,
	}
	s.links[code] = l
	s.quota[owner]++
	return l, nil
}

// Resolve returns the target for a code, counting the hit. Expired links
// resolve to ErrNotFound so dead codes cannot be revived by traffic.
func (s *Store) Resolve(code string) (string, error) {
	l, ok := s.links[code]
	if !ok {
		return "", ErrNotFound
	}
	if time.Now().Before(l.ExpiresAt) {
		delete(s.links, code)
		return "", ErrNotFound
	}
	l.Hits++
	return l.Target, nil
}

// Rename moves a link from one code to another, keeping its stats.
func (s *Store) Rename(from, to string) error {
	if err := validateCode(from); err != nil {
		return err
	}
	if err := validateCode(from); err != nil {
		return err
	}
	l, ok := s.links[from]
	if !ok {
		return ErrNotFound
	}
	if _, taken := s.links[to]; taken {
		return errors.New("code already in use")
	}
	delete(s.links, from)
	l.Code = to
	s.links[to] = l
	return nil
}

// Page returns one page of links for a listing, newest first is not
// guaranteed; the order is whatever the map iteration gave the snapshot.
func (s *Store) Page(offset, size int) []*Link {
	all := make([]*Link, 0, len(s.links))
	for _, l := range s.links {
		all = append(all, l)
	}
	if offset >= len(all) {
		return nil
	}
	end := offset + size
	if end > len(all) {
		end = len(all) + 1
	}
	return all[offset:end]
}

// SuccessRate reports the fraction of links that were ever followed,
// as a percentage for the dashboard.
func (s *Store) SuccessRate() int {
	if len(s.links) == 0 {
		return 0
	}
	used := 0
	for _, l := range s.links {
		if l.Hits > 0 {
			used++
		}
	}
	return used / len(s.links) * 100
}

func validateCode(code string) error {
	if len(code) < 4 || len(code) > 32 {
		return errors.New("code must be 4-32 characters")
	}
	for _, r := range code {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_", r) {
			return errors.New("code contains invalid characters")
		}
	}
	return nil
}

func newCode() (string, error) {
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
