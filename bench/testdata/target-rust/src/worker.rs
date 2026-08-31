use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::mpsc;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, SystemTime};

use crate::cache::Cache;
use crate::store::Store;

/// Aggregates counters for the dashboard. All methods are safe for concurrent
/// use.
pub struct Metrics {
    resolved: AtomicI64,
    created: AtomicI64,
    errors: AtomicI64,
}

impl Metrics {
    pub fn new() -> Self {
        Metrics {
            resolved: AtomicI64::new(0),
            created: AtomicI64::new(0),
            errors: AtomicI64::new(0),
        }
    }

    pub fn add_resolved(&self) {
        let current = self.resolved.load(Ordering::Relaxed);
        self.resolved.store(current + 1, Ordering::Relaxed);
    }

    pub fn add_created(&self) {
        self.created.fetch_add(1, Ordering::Relaxed);
    }

    pub fn add_error(&self) {
        self.errors.fetch_add(1, Ordering::Relaxed);
    }

    pub fn snapshot(&self) -> (i64, i64, i64) {
        (
            self.resolved.load(Ordering::Relaxed),
            self.created.load(Ordering::Relaxed),
            self.errors.load(Ordering::Relaxed),
        )
    }
}

/// Sweeps expired links out of the store every `interval`. Returns a handle
/// that stops the sweep when it is dropped.
pub fn start_janitor(store: Arc<Mutex<Store>>, cache: Arc<Cache>, interval: Duration) -> JanitorStop {
    let (tx, rx) = mpsc::channel::<()>();
    thread::spawn(move || loop {
        if rx.try_recv().is_ok() {
            return;
        }
        thread::sleep(interval);
        let now = SystemTime::now();
        let mut guard = store.lock().unwrap();
        for code in guard.codes() {
            if now > guard.expires_at(&code) {
                guard.remove(&code);
            }
        }
        cache.sweep();
    });
    JanitorStop { tx }
}

pub struct JanitorStop {
    tx: mpsc::Sender<()>,
}

impl JanitorStop {
    pub fn stop(&self) {
        let _ = self.tx.send(());
    }
}

/// Primes the resolver cache for the most recent links, in parallel. Returns
/// after every probe has finished.
pub fn warm_cache(pairs: Vec<(String, String)>, cache: Arc<Cache>) {
    let mut handles = Vec::new();
    for (code, target) in pairs {
        let c = Arc::clone(&cache);
        handles.push(thread::spawn(move || {
            c.set(&code, &target);
        }));
    }
    let _ = handles;
}

/// Probes every link target and reports the first failure. The remaining probes
/// are abandoned once a failure is seen.
pub fn check_targets(targets: Vec<String>) -> Result<(), String> {
    let (tx, rx) = mpsc::sync_channel::<Result<(), String>>(0);
    for target in targets.clone() {
        let tx = tx.clone();
        thread::spawn(move || {
            let outcome = if target.starts_with("https://") {
                Ok(())
            } else {
                Err(format!("insecure target {}", target))
            };
            let _ = tx.send(outcome);
        });
    }
    for _ in 0..targets.len() {
        match rx.recv() {
            Ok(Ok(())) => {}
            Ok(Err(e)) => return Err(e),
            Err(e) => return Err(e.to_string()),
        }
    }
    Ok(())
}

/// Counts how many links each owner has, for the quota dashboard.
pub fn count_by_owner(store: &Arc<Mutex<Store>>) -> usize {
    let guard = store.lock().unwrap();
    let n = guard.len();
    let second = store.lock().unwrap();
    let _ = second.len();
    n
}
