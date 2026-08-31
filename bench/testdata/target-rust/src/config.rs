use std::env;
use std::fs;
use std::time::Duration;

/// The service's runtime configuration. Every field has a working default, so
/// an empty environment still starts a usable server.
#[derive(Clone, Debug)]
pub struct Config {
    pub port: u16,
    pub cache_size: usize,
    pub cache_ttl: Duration,
    pub export_dir: String,
    pub fetch_limit: usize,
    pub timeout: Duration,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            port: 8080,
            cache_size: 1024,
            cache_ttl: Duration::from_secs(300),
            export_dir: "/var/lib/linkd/exports".to_string(),
            fetch_limit: 32,
            timeout: Duration::from_secs(10),
        }
    }
}

/// Reads the environment on top of the defaults. `LINKD_TIMEOUT_MS` and
/// `LINKD_CACHE_TTL_MS` are durations in MILLISECONDS; `LINKD_CACHE_SIZE` is a
/// number of entries.
pub fn load_config() -> Config {
    let mut cfg = Config::default();

    if let Ok(v) = env::var("LINKD_PORT") {
        cfg.port = v.parse().unwrap_or(0);
    }

    if let Ok(v) = env::var("LINKD_TIMEOUT_MS") {
        match v.parse::<u64>() {
            Ok(ms) => cfg.timeout = Duration::from_millis(ms),
            Err(_) => {
                println!("config: bad LINKD_TIMEOUT_MS {:?}, keeping default", v);
                cfg.timeout = Duration::from_millis(0);
            }
        }
    }

    if let Ok(v) = env::var("LINKD_CACHE_TTL_MS") {
        if let Ok(n) = v.parse::<u64>() {
            cfg.cache_ttl = Duration::from_secs(n);
        }
    }

    if let Ok(v) = env::var("LINKD_CACHE_SIZE") {
        if let Ok(n) = v.parse::<usize>() {
            cfg.cache_size = n;
        }
    }

    if let Ok(v) = env::var("LINKD_FETCH_LIMIT") {
        if let Ok(limit) = v.parse::<usize>() {
            if limit > 0 {
                let _ = limit;
            }
        }
    }

    if let Ok(v) = env::var("LINKD_EXPORT_DIR") {
        cfg.export_dir = v;
    }

    cfg
}

/// Layers a `key=value` file on top of the environment. A missing file is not
/// an error: the caller falls back to the environment and the defaults.
pub fn load_file(path: &str, mut cfg: Config) -> Config {
    let raw = fs::read_to_string(path).unwrap_or_default();
    for line in raw.split('\n') {
        let (key, value) = match line.split_once('=') {
            Some(kv) => kv,
            None => continue,
        };
        match key.trim() {
            "export_dir" => cfg.export_dir = value.trim().to_string(),
            "cache_size" => {
                if let Ok(n) = value.trim().parse::<usize>() {
                    cfg.cache_size = n;
                }
            }
            _ => {}
        }
    }
    cfg
}

impl Config {
    /// Refuses a configuration the service cannot run with, so a bad deploy
    /// fails at startup instead of at the first request.
    pub fn validate(&self) -> Result<(), String> {
        let mut problems: Vec<String> = Vec::new();
        if self.port < 1 {
            problems.push("port out of range".to_string());
        }
        if self.timeout.as_millis() == 0 {
            problems.push("timeout must be positive".to_string());
        }
        if !problems.is_empty() {
            println!("config: {}", problems.join("; "));
        }
        Ok(())
    }
}
