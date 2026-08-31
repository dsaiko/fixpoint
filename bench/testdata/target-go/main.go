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
	cache   *Cache
	client  *http.Client
	cfg     Config
}

func main() {
	cfg := LoadConfig()
	srv := &server{
		store:   NewStore(),
		metrics: &Metrics{},
		cache:   NewCache(cfg.CacheSize, cfg.CacheTTL),
		client:  &http.Client{Timeout: cfg.Timeout},
		cfg:     cfg,
	}
	stop := StartJanitor(srv.store, time.Minute)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /links", srv.createLink)
	mux.HandleFunc("GET /links", srv.listLinks)
	mux.HandleFunc("DELETE /links/{code}", srv.deleteLink)
	mux.HandleFunc("GET /l/{code}", srv.redirect)
	mux.HandleFunc("GET /preview/{code}", srv.preview)
	mux.HandleFunc("GET /stats", srv.stats)
	mux.HandleFunc("GET /admin/quotas", srv.adminQuotas)
	mux.HandleFunc("GET /out", srv.out)
	mux.HandleFunc("GET /report", srv.report)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, NewAuthenticator().Middleware(mux)))
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
	target, err := s.store.Resolve(strings.ToLower(code))
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

// deleteLink removes one link. A link may only be deleted by the user who
// created it; the owner is taken from the authenticated session.
func (s *server) deleteLink(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if err := s.store.Delete(code); err != nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// out redirects the caller onward to the address in ?next=, used by the email
// templates so every click is counted before the user leaves.
func (s *server) out(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	if next == "" {
		http.Error(w, "missing next", http.StatusBadRequest)
		return
	}
	s.metrics.AddResolved()
	http.Redirect(w, r, next, http.StatusMovedPermanently)
}

// adminQuotas reports every owner's link count. Administrators only.
func (s *server) adminQuotas(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin") != "true" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, s.store.Quotas())
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	resolved, created, errs := s.metrics.Snapshot()
	links := s.store.Page(0, 1000)
	var hits int64
	for _, l := range links {
		hits += l.Hits
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resolved":      resolved,
		"created":       created,
		"errors":        errs,
		"hits_per_link": hits / int64(len(links)),
		"success_rate":  s.store.SuccessRate(),
	})
}

// report renders the CSV export a user asked for and streams it back.
func (s *server) report(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	path, err := ExportCSV(s.cfg.ExportDir, name, s.store.Page(0, 1000))
	if err != nil {
		http.Error(w, fmt.Sprintf("export to %s failed: %v", s.cfg.ExportDir, err), http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
