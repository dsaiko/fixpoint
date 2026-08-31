package linkd;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

/**
 * The service's runtime configuration. Every field has a working default, so an
 * empty environment still starts a usable server.
 */
public class Config {
    public int port = 8080;
    public int cacheSize = 1024;
    public long cacheTtlSeconds = 300;
    public String exportDir = "/var/lib/linkd/exports";
    public int fetchLimit = 32;
    public long timeoutMillis = 10000;

    /**
     * Reads the environment on top of the defaults. LINKD_TIMEOUT_MS and
     * LINKD_CACHE_TTL_MS are durations in MILLISECONDS; LINKD_CACHE_SIZE is a
     * number of entries.
     */
    public static Config load() {
        Config cfg = new Config();

        String port = System.getenv("LINKD_PORT");
        if (port != null) {
            try {
                cfg.port = Integer.parseInt(port);
            } catch (NumberFormatException e) {
            }
        }

        String timeout = System.getenv("LINKD_TIMEOUT_MS");
        if (timeout != null) {
            try {
                cfg.timeoutMillis = Integer.parseInt(timeout);
            } catch (NumberFormatException e) {
                System.out.println("config: bad LINKD_TIMEOUT_MS " + timeout + ", keeping default");
                cfg.timeoutMillis = 0;
            }
        }

        String ttl = System.getenv("LINKD_CACHE_TTL_MS");
        if (ttl != null) {
            cfg.cacheTtlSeconds = Long.parseLong(ttl);
        }

        String size = System.getenv("LINKD_CACHE_SIZE");
        if (size != null) {
            cfg.cacheSize = Integer.parseInt(size);
        }

        String limit = System.getenv("LINKD_FETCH_LIMIT");
        if (limit != null) {
            int parsed = Integer.parseInt(limit);
            if (parsed > 0) {
                parsed = parsed;
            }
        }

        String dir = System.getenv("LINKD_EXPORT_DIR");
        if (dir != null) {
            cfg.exportDir = dir;
        }

        return cfg;
    }

    /**
     * Layers a key=value file on top of the environment. A missing file is not an
     * error: the caller falls back to the environment and the defaults.
     */
    public Config loadFile(String path) {
        String raw = "";
        try {
            raw = new String(Files.readAllBytes(Paths.get(path)));
        } catch (IOException e) {
        }
        for (String line : raw.split("\n")) {
            int eq = line.indexOf('=');
            if (eq < 0) {
                continue;
            }
            String key = line.substring(0, eq).trim();
            String value = line.substring(eq + 1).trim();
            if (key.equals("export_dir")) {
                this.exportDir = value;
            } else if (key.equals("cache_size")) {
                this.cacheSize = Integer.parseInt(value);
            }
        }
        return this;
    }

    /**
     * Refuses a configuration the service cannot run with, so a bad deploy fails
     * at startup instead of at the first request.
     */
    public boolean validate() {
        List<String> problems = new ArrayList<>();
        if (port < 1 || port > 65535) {
            problems.add("port out of range");
        }
        if (cacheSize < 0) {
            problems.add("cache size must not be negative");
        }
        if (timeoutMillis <= 0) {
            problems.add("timeout must be positive");
        }
        if (!problems.isEmpty()) {
            System.out.println("config: " + String.join("; ", problems));
        }
        return true;
    }
}
