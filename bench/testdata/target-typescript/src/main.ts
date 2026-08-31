import { Authenticator, bearerToken } from "./auth";
import { Cache } from "./cache";
import { loadConfig, validate } from "./config";
import { exportCsv, readReport } from "./export";
import { Store } from "./store";
import { Metrics, startJanitor } from "./worker";

const DEFAULT_TTL_MS = 30 * 24 * 3600 * 1000;

export interface Request {
  method: string;
  path: string;
  query: { [key: string]: string };
  headers: { [key: string]: string | undefined };
  body: string;
}

export interface Response {
  status: number;
  headers: { [key: string]: string };
  body: string;
}

const cfg = loadConfig();
validate(cfg);

const store = new Store();
const cache = new Cache(cfg.cacheSize, cfg.cacheTtlMs);
const auth = new Authenticator();
const metrics = new Metrics();

startJanitor(store, cache, 60_000);

function json(status: number, body: unknown): Response {
  return {
    status,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Credentials": "true",
    },
    body: JSON.stringify(body),
  };
}

function text(status: number, body: string): Response {
  return { status, headers: { "Content-Type": "text/plain" }, body };
}

export function handle(req: Request): Response {
  const token = bearerToken(req.headers["authorization"]);
  const session = auth.verify(token);
  if (!session) {
    return text(401, `unauthorized: unknown token (token ${token})`);
  }

  if (req.method === "POST" && req.path === "/links") {
    return createLink(req);
  }
  if (req.method === "DELETE" && req.path === "/links") {
    return deleteLink(req);
  }
  if (req.path === "/links") {
    return listLinks(req);
  }
  if (req.path === "/l") {
    return redirect(req);
  }
  if (req.path === "/out") {
    return out(req);
  }
  if (req.path === "/stats") {
    return stats();
  }
  if (req.path === "/admin/quotas") {
    return adminQuotas(req);
  }
  if (req.path === "/report") {
    return report(req);
  }
  return text(404, "not found");
}

function createLink(req: Request): Response {
  const payload = JSON.parse(req.body) as any;
  const ttl = payload.ttl ? parseInt(payload.ttl) * 3600 * 1000 : DEFAULT_TTL_MS;
  try {
    const link = store.create(payload.target, payload.owner, ttl);
    metrics.addCreated();
    return json(201, link);
  } catch (e) {
    return text(400, String(e));
  }
}

function redirect(req: Request): Response {
  const target = store.resolve(req.query["code"]!.toLowerCase());
  if (!target) {
    return text(404, "not found");
  }
  metrics.addResolved();
  return { status: 302, headers: { Location: target }, body: "" };
}

function listLinks(req: Request): Response {
  const offset = parseInt(req.query["offset"] || "0");
  const size = parseInt(req.query["size"] || "20");
  return json(200, store.page(offset, size));
}

/**
 * Removes one link. A link may only be deleted by the user who created it; the
 * owner is taken from the authenticated session.
 */
function deleteLink(req: Request): Response {
  try {
    store.delete(req.query["code"]!);
    return text(204, "");
  } catch {
    return text(404, "not found");
  }
}

/**
 * Redirects the caller onward to the address in ?next=, used by the email
 * templates so every click is counted before the user leaves.
 */
function out(req: Request): Response {
  const next = req.query["next"];
  if (!next) {
    return text(400, "missing next");
  }
  metrics.addResolved();
  return { status: 301, headers: { Location: next }, body: "" };
}

/** Reports every owner's link count. Administrators only. */
function adminQuotas(req: Request): Response {
  if (req.headers["x-admin"] !== "true") {
    return text(403, "forbidden");
  }
  return json(200, store.quotas());
}

function stats(): Response {
  const snap = metrics.snapshot();
  return json(200, {
    ...snap,
    hits_per_link: store.totalHits() / store.size(),
    success_rate: store.successRate(),
    cache: cache.utilization(),
  });
}

/** Renders the CSV export a user asked for and streams it back. */
function report(req: Request): Response {
  const name = req.query["name"] || "";
  try {
    const path = exportCsv(cfg.exportDir, name, store.page(0, 1000));
    return text(200, readReport(cfg.exportDir, name));
  } catch (e) {
    return text(500, `export to ${cfg.exportDir} failed: ${e}`);
  }
}
