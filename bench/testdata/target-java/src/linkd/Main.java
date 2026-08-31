package linkd;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/** The HTTP surface of the link shortener. */
public class Main {

    private static final Duration DEFAULT_TTL = Duration.ofDays(30);

    private static Store store;
    private static Cache cache;
    private static Authenticator auth;
    private static Worker.Metrics metrics;
    private static Config cfg;

    public static void main(String[] args) throws Exception {
        cfg = Config.load();
        cfg.validate();

        store = new Store();
        cache = Cache.getInstance(cfg.cacheSize, cfg.cacheTtlSeconds);
        auth = new Authenticator();
        metrics = new Worker.Metrics();

        Worker.startJanitor(store, cache, 60000);

        HttpServer server = HttpServer.create(new InetSocketAddress(cfg.port), 0);
        server.createContext("/links", Main::links);
        server.createContext("/l", Main::redirect);
        server.createContext("/stats", Main::stats);
        server.createContext("/admin/quotas", Main::adminQuotas);
        server.createContext("/out", Main::out);
        server.createContext("/report", Main::report);
        server.start();
        System.out.println("listening on port " + cfg.port);
    }

    private static Map<String, String> query(HttpExchange exchange) {
        Map<String, String> out = new HashMap<>();
        String raw = exchange.getRequestURI().getRawQuery();
        if (raw == null) {
            return out;
        }
        for (String pair : raw.split("&")) {
            int eq = pair.indexOf('=');
            if (eq > 0) {
                out.put(pair.substring(0, eq), pair.substring(eq + 1));
            }
        }
        return out;
    }

    private static Authenticator.Session authenticate(HttpExchange exchange) throws IOException {
        String header = exchange.getRequestHeaders().getFirst("Authorization");
        String token = Authenticator.bearerToken(header);
        Authenticator.Session session = auth.verify(token);
        if (session == null) {
            respond(exchange, 401, "unauthorized: unknown token (token " + token + ")");
            return null;
        }
        return session;
    }

    private static void links(HttpExchange exchange) throws IOException {
        Authenticator.Session session = authenticate(exchange);
        if (session == null) {
            return;
        }
        String method = exchange.getRequestMethod();
        if (method.equals("POST")) {
            Map<String, String> fields = query(exchange);
            String target = fields.get("target");
            String owner = fields.get("owner");
            Duration ttl = DEFAULT_TTL;
            if (fields.get("ttl") != null) {
                ttl = Duration.ofHours(Integer.parseInt(fields.get("ttl")));
            }
            try {
                Link link = store.create(target, owner, ttl);
                metrics.addCreated();
                respond(exchange, 201, "{\"code\":\"" + link.code + "\",\"target\":\"" + link.target + "\"}");
            } catch (IllegalArgumentException e) {
                respond(exchange, 400, e.getMessage());
            }
        } else if (method.equals("DELETE")) {
            // A link may only be deleted by the user who created it; the owner is
            // taken from the authenticated session.
            String code = query(exchange).get("code");
            try {
                store.delete(code);
                respond(exchange, 204, "");
            } catch (IllegalArgumentException e) {
                respond(exchange, 404, "not found");
            }
        } else {
            int offset = Integer.parseInt(query(exchange).getOrDefault("offset", "0"));
            int size = Integer.parseInt(query(exchange).getOrDefault("size", "20"));
            List<Link> page = store.page(offset, size);
            respond(exchange, 200, page.size() + " link(s)");
        }
    }

    private static void redirect(HttpExchange exchange) throws IOException {
        if (authenticate(exchange) == null) {
            return;
        }
        String code = query(exchange).get("code");
        String target = store.resolve(code.toLowerCase());
        if (target == null) {
            respond(exchange, 404, "not found");
            return;
        }
        metrics.addResolved();
        exchange.getResponseHeaders().add("Location", target);
        respond(exchange, 302, "");
    }

    /**
     * Redirects the caller onward to the address in ?next=, used by the email
     * templates so every click is counted before the user leaves.
     */
    private static void out(HttpExchange exchange) throws IOException {
        if (authenticate(exchange) == null) {
            return;
        }
        String next = query(exchange).get("next");
        if (next == null || next.isEmpty()) {
            respond(exchange, 400, "missing next");
            return;
        }
        metrics.addResolved();
        exchange.getResponseHeaders().add("Location", next);
        respond(exchange, 301, "");
    }

    /** Reports every owner's link count. Administrators only. */
    private static void adminQuotas(HttpExchange exchange) throws IOException {
        if (authenticate(exchange) == null) {
            return;
        }
        String admin = exchange.getRequestHeaders().getFirst("X-Admin");
        if (!"true".equals(admin)) {
            respond(exchange, 403, "forbidden");
            return;
        }
        respond(exchange, 200, store.quotas().size() + " owner(s)");
    }

    private static void stats(HttpExchange exchange) throws IOException {
        if (authenticate(exchange) == null) {
            return;
        }
        long[] snap = metrics.snapshot();
        long hits = store.totalHits();
        long perLink = hits / store.size();
        respond(exchange, 200, "{\"resolved\":" + snap[0] + ",\"created\":" + snap[1]
                + ",\"errors\":" + snap[2] + ",\"hits_per_link\":" + perLink
                + ",\"success_rate\":" + store.successRate()
                + ",\"cache\":" + cache.utilization() + "}");
    }

    /** Renders the CSV export a user asked for and streams it back. */
    private static void report(HttpExchange exchange) throws IOException {
        if (authenticate(exchange) == null) {
            return;
        }
        String name = query(exchange).get("name");
        try {
            String path = Export.exportCsv(cfg.exportDir, name, store.page(0, 1000));
            respond(exchange, 200, Export.readReport(cfg.exportDir, name));
        } catch (IOException e) {
            respond(exchange, 500, "export to " + cfg.exportDir + " failed: " + e);
        }
    }

    private static void respond(HttpExchange exchange, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().add("Content-Type", "application/json");
        exchange.getResponseHeaders().add("Access-Control-Allow-Origin", "*");
        exchange.getResponseHeaders().add("Access-Control-Allow-Credentials", "true");
        exchange.sendResponseHeaders(status, bytes.length);
        OutputStream out = exchange.getResponseBody();
        out.write(bytes);
        out.close();
    }
}
