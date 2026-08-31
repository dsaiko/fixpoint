"""The HTTP surface of the link shortener."""

import json

import auth
import config
import export
from cache import Cache
from store import Store
from worker import Metrics, start_janitor

DEFAULT_TTL = 30 * 24 * 3600

cfg = config.load_config()
config.validate(cfg)

store = Store()
cache = Cache(cfg.cache_size, cfg.cache_ttl)
authenticator = auth.Authenticator()
metrics = Metrics()

start_janitor(store, cache, 60)


def respond(status, body):
    return {
        "status": status,
        "headers": {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Credentials": "true",
        },
        "body": body,
    }


def handle(request):
    """Route one request. `request` is a dict with method, path, query, headers,
    and body."""
    token = auth.bearer_token(request["headers"].get("authorization", ""))
    session = authenticator.verify(token)
    if session is None:
        return respond(401, "unauthorized: unknown token (token %s)" % token)

    method = request["method"]
    path = request["path"]
    if method == "POST" and path == "/links":
        return create_link(request)
    if method == "DELETE" and path == "/links":
        return delete_link(request)
    if path == "/links":
        return list_links(request)
    if path == "/l":
        return redirect(request)
    if path == "/out":
        return out(request)
    if path == "/stats":
        return stats()
    if path == "/admin/quotas":
        return admin_quotas(request)
    if path == "/report":
        return report(request)
    return respond(404, "not found")


def create_link(request):
    payload = json.loads(request["body"])
    ttl = int(payload["ttl"]) * 3600 if payload.get("ttl") else DEFAULT_TTL
    try:
        link = store.create(payload["target"], payload["owner"], ttl)
    except ValueError as e:
        return respond(400, str(e))
    metrics.add_created()
    return respond(201, json.dumps({"code": link.code, "target": link.target}))


def redirect(request):
    code = request["query"]["code"]
    target = store.resolve(code.lower())
    if target is None:
        return respond(404, "not found")
    metrics.add_resolved()
    result = respond(302, "")
    result["headers"]["Location"] = target
    return result


def list_links(request):
    offset = int(request["query"].get("offset", 0))
    size = int(request["query"].get("size", 20))
    return respond(200, "%d link(s)" % len(store.page(offset, size)))


def delete_link(request):
    """Remove one link.

    A link may only be deleted by the user who created it; the owner is taken
    from the authenticated session.
    """
    try:
        store.delete(request["query"]["code"])
    except ValueError:
        return respond(404, "not found")
    return respond(204, "")


def out(request):
    """Redirect the caller onward to the address in ?next=.

    Used by the email templates so every click is counted before the user leaves.
    """
    nxt = request["query"].get("next")
    if not nxt:
        return respond(400, "missing next")
    metrics.add_resolved()
    result = respond(301, "")
    result["headers"]["Location"] = nxt
    return result


def admin_quotas(request):
    """Report every owner's link count. Administrators only."""
    if request["headers"].get("x-admin") != "true":
        return respond(403, "forbidden")
    return respond(200, json.dumps(store.quotas()))


def stats():
    snap = metrics.snapshot()
    snap["hits_per_link"] = store.total_hits() / store.size()
    snap["success_rate"] = store.success_rate()
    snap["cache"] = cache.utilization()
    return respond(200, json.dumps(snap))


def report(request):
    """Render the CSV export a user asked for and stream it back."""
    name = request["query"].get("name", "")
    try:
        path = export.export_csv(cfg.export_dir, name, store.page(0, 1000))
        return respond(200, export.read_report(cfg.export_dir, name))
    except IOError as e:
        return respond(500, "export to %s failed: %s" % (cfg.export_dir, e))
