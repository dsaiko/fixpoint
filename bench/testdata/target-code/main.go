package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultTTL = 30 * 24 * time.Hour

type server struct {
	store   *Store
	metrics *Metrics
	client  *http.Client
}

func main() {
	srv := &server{
		store:   NewStore(),
		metrics: &Metrics{},
		client:  &http.Client{Timeout: 5 * time.Second},
	}
	stop := StartJanitor(srv.store, time.Minute)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /links", srv.createLink)
	mux.HandleFunc("GET /links", srv.listLinks)
	mux.HandleFunc("GET /l/{code}", srv.redirect)
	mux.HandleFunc("GET /preview/{code}", srv.preview)
	mux.HandleFunc("GET /stats", srv.stats)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) createLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
		Owner  string `json:"owner"`
		TTL    string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request body", http.StatusBadRequest)
		return
	}
	ttl := defaultTTL
	if req.TTL != "" {
		hours, _ := strconv.Atoi(req.TTL)
		ttl = time.Duration(hours) * time.Hour
	}
	link, err := s.store.Create(req.Target, req.Owner, ttl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.metrics.AddCreated()
	writeJSON(w, http.StatusCreated, link)
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	target, err := s.store.Resolve(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.metrics.AddResolved()
	http.Redirect(w, r, target, http.StatusFound)
}

// preview fetches the target page and returns its first bytes, so a user can
// inspect where a short link goes without following it.
func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	target, err := s.store.Resolve(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	resp, err := s.client.Get(target)
	if err != nil {
		http.Error(w, "target unreachable", http.StatusBadGateway)
		return
	}
	head, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		http.Error(w, "target read failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s\n---\n%s", target, strings.ToValidUTF8(string(head), "?"))
}

func (s *server) listLinks(w http.ResponseWriter, r *http.Request) {
	offset, size := 0, 20
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "bad offset", http.StatusBadRequest)
			return
		}
		offset = n
	}
	if v := r.URL.Query().Get("size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			http.Error(w, "bad size", http.StatusBadRequest)
			return
		}
		size = n
	}
	writeJSON(w, http.StatusOK, s.store.Page(offset, size))
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	resolved, created, errs := s.metrics.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"resolved":     resolved,
		"created":      created,
		"errors":       errs,
		"success_rate": s.store.SuccessRate(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
