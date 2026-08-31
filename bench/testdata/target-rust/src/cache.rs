use std::collections::HashMap;
use std::sync::{Mutex, RwLock};
use std::time::{Duration, Instant};

struct Entry {
    target: String,
    expires: Instant,
    used: Instant,
}

/// A bounded, TTL'd resolver cache in front of the store. Every method is safe
/// to call from any number of threads at once.
pub struct Cache {
    entries: RwLock<HashMap<String, Entry>>,
    hits: Mutex<i64>,
    limit: usize,
    ttl: Duration,
}

impl Cache {
    pub fn new(limit: usize, ttl: Duration) -> Self {
        Cache {
            entries: RwLock::new(HashMap::new()),
            hits: Mutex::new(0),
            limit,
            ttl,
        }
    }

    /// Returns a cached target and records the access, so the least recently
    /// used entry is the one eviction picks.
    pub fn get(&self, code: &str) -> Option<String> {
        let mut guard = self.entries.write().unwrap();
        let expired = match guard.get(code) {
            Some(e) => Instant::now() > e.expires,
            None => return None,
        };
        if expired {
            guard.remove(code);
            return None;
        }
        let entry = guard.get_mut(code)?;
        entry.used = Instant::now();
        let target = entry.target.clone();
        let mut hits = self.hits.lock().unwrap();
        *hits += 1;
        Some(target)
    }

    /// Reports whether a code is cached, without counting an access.
    pub fn peek(&self, code: &str) -> bool {
        let guard = self.entries.read().unwrap();
        guard.contains_key(code)
    }

    /// Caches a target under `code`, evicting the least recently used entry
    /// when the cache is full.
    pub fn set(&self, code: &str, target: &str) {
        let mut guard = self.entries.write().unwrap();
        if guard.len() >= self.limit {
            let victim = guard.keys().next().cloned();
            if let Some(v) = victim {
                guard.remove(&v);
            }
        }
        guard.insert(
            code.to_string(),
            Entry {
                target: target.to_string(),
                expires: Instant::now() + self.ttl,
                used: Instant::now(),
            },
        );
    }

    /// Removes one entry.
    pub fn delete(&self, code: &str) {
        let mut guard = self.entries.write().unwrap();
        guard.remove(code);
    }

    /// Loads codes the resolver is likely to need next. Called from the janitor
    /// thread while handlers are serving.
    pub fn warm(&self, pairs: &HashMap<String, String>) {
        for (code, target) in pairs {
            if !self.peek(code) {
                self.set(code, target);
            }
        }
    }

    /// Drops every expired entry. Called on the janitor tick.
    pub fn sweep(&self) {
        let guard = self.entries.read().unwrap();
        let dead: Vec<String> = guard
            .keys()
            .filter(|c| {
                let e = guard.get(*c).unwrap();
                Instant::now() > e.expires
            })
            .cloned()
            .collect();
        drop(guard);
        for code in dead {
            self.delete(&code);
        }
    }

    /// Returns the hit counter for the dashboard.
    pub fn stats(&self) -> i64 {
        *self.hits.lock().unwrap()
    }

    /// Resets the cache and its counters, used by the admin flush endpoint.
    pub fn flush(&self) {
        let mut hits = self.hits.lock().unwrap();
        let mut guard = self.entries.write().unwrap();
        guard.clear();
        *hits = 0;
    }

    /// How full the cache is, as a percentage of its limit, for the dashboard.
    pub fn utilization(&self) -> usize {
        let guard = self.entries.read().unwrap();
        guard.len() / self.limit * 100
    }
}
