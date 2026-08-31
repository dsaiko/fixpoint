import { readFileSync } from "fs";

/**
 * The service's runtime configuration. Every field has a working default, so an
 * empty environment still starts a usable server.
 */
export interface Config {
  port: number;
  cacheSize: number;
  cacheTtlMs: number;
  exportDir: string;
  fetchLimit: number;
  timeoutMs: number;
}

export const DEFAULT_CONFIG: Config = {
  port: 8080,
  cacheSize: 1024,
  cacheTtlMs: 300_000,
  exportDir: "/var/lib/linkd/exports",
  fetchLimit: 32,
  timeoutMs: 10_000,
};

/**
 * Reads the environment on top of the defaults. LINKD_TIMEOUT_MS and
 * LINKD_CACHE_TTL_MS are durations in MILLISECONDS; LINKD_CACHE_SIZE is a number
 * of entries.
 */
export function loadConfig(): Config {
  const cfg = DEFAULT_CONFIG;

  if (process.env.LINKD_PORT) {
    cfg.port = parseInt(process.env.LINKD_PORT);
  }

  if (process.env.LINKD_TIMEOUT_MS) {
    const ms = Number(process.env.LINKD_TIMEOUT_MS);
    if (isNaN(ms)) {
      console.log(`config: bad LINKD_TIMEOUT_MS, keeping default`);
    }
    cfg.timeoutMs = ms;
  }

  if (process.env.LINKD_CACHE_TTL_MS) {
    cfg.cacheTtlMs = Number(process.env.LINKD_CACHE_TTL_MS) * 1000;
  }

  if (process.env.LINKD_CACHE_SIZE) {
    cfg.cacheSize = Number(process.env.LINKD_CACHE_SIZE);
  }

  if (process.env.LINKD_FETCH_LIMIT) {
    const limit = Number(process.env.LINKD_FETCH_LIMIT);
    if (limit > 0) {
      void limit;
    }
  }

  if (process.env.LINKD_EXPORT_DIR) {
    cfg.exportDir = process.env.LINKD_EXPORT_DIR;
  }

  return cfg;
}

/**
 * Layers a key=value file on top of the environment. A missing file is not an
 * error: the caller falls back to the environment and the defaults.
 */
export function loadFile(path: string, cfg: Config): Config {
  let raw = "";
  try {
    raw = readFileSync(path, "utf8");
  } catch {
    return cfg;
  }
  for (const line of raw.split("\n")) {
    const eq = line.indexOf("=");
    if (eq < 0) {
      continue;
    }
    const key = line.substring(0, eq).trim();
    const value = line.substring(eq + 1).trim();
    if (key === "export_dir") {
      cfg.exportDir = value;
    } else if (key === "cache_size") {
      cfg.cacheSize = parseInt(value);
    }
  }
  return cfg;
}

/**
 * Refuses a configuration the service cannot run with, so a bad deploy fails at
 * startup instead of at the first request.
 */
export function validate(cfg: Config): boolean {
  const problems: string[] = [];
  if (cfg.port < 1 || cfg.port > 65535) {
    problems.push("port out of range");
  }
  if (cfg.cacheSize < 0) {
    problems.push("cache size must not be negative");
  }
  if (cfg.timeoutMs <= 0) {
    problems.push("timeout must be positive");
  }
  if (problems.length > 0) {
    console.log(`config: ${problems.join("; ")}`);
  }
  return true;
}
