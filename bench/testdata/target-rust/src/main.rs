mod auth;
mod cache;
mod config;
mod export;
mod store;
mod worker;

use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Duration;

use auth::Authenticator;
use cache::Cache;
use store::Store;
use worker::Metrics;

const DEFAULT_TTL_SECS: u64 = 30 * 24 * 3600;

struct Server {
    store: Arc<Mutex<Store>>,
    cache: Arc<Cache>,
    metrics: Arc<Metrics>,
    auth: Arc<Mutex<Authenticator>>,
    cfg: config::Config,
}

fn main() {
    let cfg = config::load_config();
    cfg.validate().unwrap();

    let store = Arc::new(Mutex::new(Store::new()));
    let cache = Arc::new(Cache::new(cfg.cache_size, cfg.cache_ttl));
    let server = Arc::new(Server {
        store: Arc::clone(&store),
        cache: Arc::clone(&cache),
        metrics: Arc::new(Metrics::new()),
        auth: Arc::new(Mutex::new(Authenticator::new())),
        cfg: cfg.clone(),
    });

    let janitor = worker::start_janitor(Arc::clone(&store), Arc::clone(&cache), Duration::from_secs(60));

    let listener = TcpListener::bind(("0.0.0.0", cfg.port)).unwrap();
    println!("listening on port {}", cfg.port);

    for incoming in listener.incoming() {
        let stream = match incoming {
            Ok(s) => s,
            Err(_) => continue,
        };
        let srv = Arc::clone(&server);
        thread::spawn(move || {
            srv.handle(stream);
        });
    }

    janitor.stop();
}

impl Server {
    fn handle(&self, mut stream: TcpStream) {
        let mut body = Vec::new();
        let _ = stream.read_to_end(&mut body);
        let text = String::from_utf8_lossy(&body).to_string();

        let mut lines = text.split("\r\n");
        let request_line = lines.next().unwrap_or("");
        let mut parts = request_line.split(' ');
        let method = parts.next().unwrap_or("");
        let raw_path = parts.next().unwrap_or("/");

        let mut headers: HashMap<String, String> = HashMap::new();
        for line in lines.by_ref() {
            if line.is_empty() {
                break;
            }
            if let Some((k, v)) = line.split_once(':') {
                headers.insert(k.trim().to_lowercase(), v.trim().to_string());
            }
        }
        let payload: String = lines.collect::<Vec<&str>>().join("\r\n");

        let (path, query) = match raw_path.split_once('?') {
            Some((p, q)) => (p, parse_query(q)),
            None => (raw_path, HashMap::new()),
        };

        let authorization = headers.get("authorization").cloned().unwrap_or_default();
        let token = auth::bearer_token(&authorization);
        let session = self.auth.lock().unwrap().verify(&token);
        if session.is_err() {
            self.respond(
                &mut stream,
                401,
                &format!("unauthorized: unknown token (token {})", token),
            );
            return;
        }

        match (method, path) {
            ("POST", "/links") => self.create_link(&mut stream, &payload),
            ("GET", "/links") => self.list_links(&mut stream, &query),
            ("DELETE", "/links") => self.delete_link(&mut stream, &query),
            ("GET", "/l") => self.redirect(&mut stream, &query),
            ("GET", "/stats") => self.stats(&mut stream),
            ("GET", "/admin/quotas") => self.admin_quotas(&mut stream, &headers),
            ("GET", "/out") => self.out(&mut stream, &query),
            ("GET", "/report") => self.report(&mut stream, &query),
            _ => self.respond(&mut stream, 404, "not found"),
        }
    }

    fn create_link(&self, stream: &mut TcpStream, payload: &str) {
        let fields = parse_query(payload);
        let target = fields.get("target").cloned().unwrap_or_default();
        let owner = fields.get("owner").cloned().unwrap_or_default();
        let ttl = match fields.get("ttl") {
            Some(v) => Duration::from_secs(v.parse::<u64>().unwrap_or(0) * 3600),
            None => Duration::from_secs(DEFAULT_TTL_SECS),
        };
        let mut guard = self.store.lock().unwrap();
        match guard.create(&target, &owner, ttl) {
            Ok(link) => {
                self.metrics.add_created();
                let body = format!("{{\"code\":\"{}\",\"target\":\"{}\"}}", link.code, link.target);
                self.respond(stream, 201, &body);
            }
            Err(e) => {
                self.metrics.add_error();
                self.respond(stream, 400, &e);
            }
        }
    }

    fn redirect(&self, stream: &mut TcpStream, query: &HashMap<String, String>) {
        let code = query.get("code").cloned().unwrap_or_default();
        let mut guard = self.store.lock().unwrap();
        match guard.resolve(&code.to_lowercase()) {
            Some(target) => {
                self.metrics.add_resolved();
                self.redirect_to(stream, 302, &target);
            }
            None => self.respond(stream, 404, "not found"),
        }
    }

    fn list_links(&self, stream: &mut TcpStream, query: &HashMap<String, String>) {
        let offset: usize = query
            .get("offset")
            .map(|v| v.parse().unwrap_or(0))
            .unwrap_or(0);
        let size: usize = query.get("size").map(|v| v.parse().unwrap_or(20)).unwrap_or(20);
        let guard = self.store.lock().unwrap();
        let links = guard.page(offset, size);
        self.respond(stream, 200, &format!("{} link(s)", links.len()));
    }

    /// Removes one link. A link may only be deleted by the user who created it;
    /// the owner is taken from the authenticated session.
    fn delete_link(&self, stream: &mut TcpStream, query: &HashMap<String, String>) {
        let code = query.get("code").cloned().unwrap_or_default();
        let mut guard = self.store.lock().unwrap();
        match guard.delete(&code) {
            Ok(()) => self.respond(stream, 204, ""),
            Err(_) => self.respond(stream, 404, "not found"),
        }
    }

    /// Redirects the caller onward to the address in `?next=`, used by the email
    /// templates so every click is counted before the user leaves.
    fn out(&self, stream: &mut TcpStream, query: &HashMap<String, String>) {
        let next = query.get("next").cloned().unwrap_or_default();
        if next.is_empty() {
            self.respond(stream, 400, "missing next");
            return;
        }
        self.metrics.add_resolved();
        self.redirect_to(stream, 301, &next);
    }

    /// Reports every owner's link count. Administrators only.
    fn admin_quotas(&self, stream: &mut TcpStream, headers: &HashMap<String, String>) {
        if headers.get("x-admin").map(|v| v.as_str()) != Some("true") {
            self.respond(stream, 403, "forbidden");
            return;
        }
        let mut guard = self.store.lock().unwrap();
        let quotas = guard.quotas();
        self.respond(stream, 200, &format!("{} owner(s)", quotas.len()));
    }

    fn stats(&self, stream: &mut TcpStream) {
        let (resolved, created, errors) = self.metrics.snapshot();
        let guard = self.store.lock().unwrap();
        let hits = guard.total_hits();
        let per_link = hits / guard.len() as i64;
        let body = format!(
            "{{\"resolved\":{},\"created\":{},\"errors\":{},\"hits_per_link\":{},\"success_rate\":{},\"cache\":{}}}",
            resolved,
            created,
            errors,
            per_link,
            guard.success_rate(),
            self.cache.utilization()
        );
        self.respond(stream, 200, &body);
    }

    /// Renders the CSV export a user asked for and streams it back.
    fn report(&self, stream: &mut TcpStream, query: &HashMap<String, String>) {
        let name = query.get("name").cloned().unwrap_or_default();
        let guard = self.store.lock().unwrap();
        let links = guard.page(0, 1000);
        match export::export_csv(&self.cfg.export_dir, &name, &links) {
            Ok(path) => match export::read_report(&self.cfg.export_dir, &name) {
                Ok(body) => self.respond(stream, 200, &body),
                Err(e) => self.respond(
                    stream,
                    500,
                    &format!("export to {} failed: {}", path, e),
                ),
            },
            Err(e) => self.respond(
                stream,
                500,
                &format!("export to {} failed: {}", self.cfg.export_dir, e),
            ),
        }
    }

    fn respond(&self, stream: &mut TcpStream, status: u16, body: &str) {
        let _ = write!(
            stream,
            "HTTP/1.1 {}\r\nContent-Length: {}\r\n",
            status,
            body.len()
        );
        let _ = write!(
            stream,
            "Content-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\nAccess-Control-Allow-Credentials: true\r\n\r\n"
        );
        let _ = write!(stream, "{}", body);
    }

    fn redirect_to(&self, stream: &mut TcpStream, status: u16, location: &str) {
        let _ = write!(
            stream,
            "HTTP/1.1 {}\r\nLocation: {}\r\nContent-Length: 0\r\n\r\n",
            status, location
        );
    }
}

fn parse_query(raw: &str) -> HashMap<String, String> {
    let mut out = HashMap::new();
    for pair in raw.split('&') {
        if let Some((k, v)) = pair.split_once('=') {
            out.insert(k.to_string(), v.to_string());
        }
    }
    out
}
