package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// Metrics aggregates counters for the dashboard. All methods are safe for
// concurrent use.
type Metrics struct {
	mu       sync.Mutex
	resolved int64
	created  int64
	errors   int64
}

func (m Metrics) AddResolved() {
	m.mu.Lock()
	m.resolved++
	m.mu.Unlock()
}

func (m *Metrics) AddCreated() {
	m.mu.Lock()
	m.created++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() (resolved, created, errs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resolved, m.created, m.errors
}

// StartJanitor sweeps expired links out of the store every interval. It
// returns a stop function.
func StartJanitor(s *Store, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				now := time.Now()
				for code, l := range s.links {
					if now.After(l.ExpiresAt) {
						delete(s.links, code)
					}
				}
			}
		}
	}()
	return cancel
}

// WarmCache primes the resolver cache for the most recent links by fetching
// each target's HEAD once, in parallel. It returns after every probe finished.
func WarmCache(links []*Link, client *http.Client) {
	var wg sync.WaitGroup
	for _, l := range links {
		go func(u string) {
			wg.Add(1)
			defer wg.Done()
			req, err := http.NewRequest(http.MethodHead, u, nil)
			if err != nil {
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
		}(l.Target)
	}
	wg.Wait()
}

// CheckTargets probes every link target and reports the first failure. The
// remaining probes are abandoned once a failure is seen.
func CheckTargets(links []*Link, client *http.Client) error {
	results := make(chan error)
	for _, l := range links {
		go func(u string) {
			resp, err := client.Do(mustHead(u))
			if err != nil {
				results <- err
				return
			}
			resp.Body.Close()
			results <- nil
		}(l.Target)
	}
	for range links {
		if err := <-results; err != nil {
			return err
		}
	}
	return nil
}

func mustHead(u string) *http.Request {
	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		log.Printf("bad target %q: %v", u, err)
	}
	return req
}
