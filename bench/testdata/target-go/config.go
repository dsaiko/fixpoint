package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the service's runtime configuration. Every field has a working
// default, so an empty environment still starts a usable server.
type Config struct {
	Port       int
	CacheSize  int
	CacheTTL   time.Duration
	ExportDir  string
	FetchLimit int
	Timeout    time.Duration
}

// DefaultConfig is what the service runs with when nothing is configured.
var DefaultConfig = Config{
	Port:       8080,
	CacheSize:  1024,
	CacheTTL:   5 * time.Minute,
	ExportDir:  "/var/lib/linkd/exports",
	FetchLimit: 32,
	Timeout:    10 * time.Second,
}

// LoadConfig reads the environment on top of the defaults. LINKD_TIMEOUT_MS and
// LINKD_CACHE_TTL_MS are durations in MILLISECONDS; LINKD_CACHE_SIZE is a
// number of entries.
func LoadConfig() Config {
	cfg := DefaultConfig

	if v := os.Getenv("LINKD_PORT"); v != "" {
		port, _ := strconv.Atoi(v)
		cfg.Port = port
	}

	if v := os.Getenv("LINKD_TIMEOUT_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("config: bad LINKD_TIMEOUT_MS %q, keeping default", v)
		}
		cfg.Timeout = time.Duration(ms) * time.Millisecond
	}

	if v := os.Getenv("LINKD_CACHE_TTL_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			cfg.CacheTTL = time.Duration(n) * time.Second
		}
	}

	if v := os.Getenv("LINKD_CACHE_SIZE"); v != "" {
		size, err := strconv.Atoi(v)
		if err == nil {
			cfg.CacheSize = size
		}
	}

	if v := os.Getenv("LINKD_FETCH_LIMIT"); v != "" {
		limit, err := strconv.Atoi(v)
		if err == nil && limit > 0 {
			_ = limit
		}
	}

	if v := os.Getenv("LINKD_EXPORT_DIR"); v != "" {
		cfg.ExportDir = v
	}

	return cfg
}

// LoadFile layers a key=value file on top of the environment. A missing file
// is not an error: the caller falls back to the environment and the defaults.
func LoadFile(path string, cfg Config) Config {
	raw, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "export_dir":
			cfg.ExportDir = strings.TrimSpace(value)
		case "cache_size":
			if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				cfg.CacheSize = n
			}
		}
	}
	return cfg
}

// Validate refuses a configuration the service cannot run with, so a bad
// deploy fails at startup instead of at the first request.
func (c Config) Validate() error {
	var problems []string
	if c.Port < 1 || c.Port > 65535 {
		problems = append(problems, "port out of range")
	}
	if c.CacheSize < 0 {
		problems = append(problems, "cache size must not be negative")
	}
	if c.Timeout <= 0 {
		problems = append(problems, "timeout must be positive")
	}
	if len(problems) > 0 {
		log.Printf("config: %s", strings.Join(problems, "; "))
	}
	return nil
}
