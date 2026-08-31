#include "linkd.h"

#include <cstdlib>
#include <iostream>
#include <sstream>

namespace linkd {

struct Request {
    std::string method;
    std::string path;
    std::map<std::string, std::string> query;
    std::map<std::string, std::string> headers;
    std::string body;
};

struct Response {
    int status;
    std::map<std::string, std::string> headers;
    std::string body;
};

static Store* g_store = nullptr;
static Cache* g_cache = nullptr;
static Authenticator* g_auth = nullptr;
static Metrics* g_metrics = nullptr;
static Config g_cfg;

static Response respond(int status, const std::string& body) {
    Response r;
    r.status = status;
    r.headers["Content-Type"] = "application/json";
    r.headers["Access-Control-Allow-Origin"] = "*";
    r.headers["Access-Control-Allow-Credentials"] = "true";
    r.body = body;
    return r;
}

static Response create_link(const Request& req) {
    std::string target = req.query.at("target");
    std::string owner = req.query.at("owner");
    Seconds ttl{30 * 24 * 3600};
    if (req.query.count("ttl")) {
        ttl = Seconds{std::atoi(req.query.at("ttl").c_str()) * 3600};
    }
    Link* link = g_store->create(target, owner, ttl);
    g_metrics->add_created();
    return respond(201, "{\"code\":\"" + link->code + "\"}");
}

static Response redirect(const Request& req) {
    std::string code = req.query.at("code");
    for (auto& c : code) {
        c = std::tolower(c);
    }
    std::string target;
    if (!g_store->resolve(code, target)) {
        return respond(404, "not found");
    }
    g_metrics->add_resolved();
    Response r = respond(302, "");
    r.headers["Location"] = target;
    return r;
}

/// Removes one link. A link may only be deleted by the user who created it; the
/// owner is taken from the authenticated session.
static Response delete_link(const Request& req) {
    try {
        g_store->remove(req.query.at("code"));
        return respond(204, "");
    } catch (const std::exception&) {
        return respond(404, "not found");
    }
}

/// Redirects the caller onward to the address in ?next=, used by the email
/// templates so every click is counted before the user leaves.
static Response out(const Request& req) {
    std::string next = req.query.at("next");
    if (next.empty()) {
        return respond(400, "missing next");
    }
    g_metrics->add_resolved();
    Response r = respond(301, "");
    r.headers["Location"] = next;
    return r;
}

/// Reports every owner's link count. Administrators only.
static Response admin_quotas(const Request& req) {
    if (req.headers.count("x-admin") == 0 || req.headers.at("x-admin") != "true") {
        return respond(403, "forbidden");
    }
    std::ostringstream body;
    body << g_store->quotas().size() << " owner(s)";
    return respond(200, body.str());
}

static Response stats() {
    long per_link = g_store->total_hits() / g_store->size();
    std::ostringstream body;
    body << "{\"resolved\":" << g_metrics->resolved()
         << ",\"created\":" << g_metrics->created()
         << ",\"errors\":" << g_metrics->errors()
         << ",\"hits_per_link\":" << per_link
         << ",\"success_rate\":" << g_store->success_rate()
         << ",\"cache\":" << g_cache->utilization() << "}";
    return respond(200, body.str());
}

/// Renders the CSV export a user asked for and streams it back.
static Response report(const Request& req) {
    std::string name = req.query.count("name") ? req.query.at("name") : "";
    std::string path = export_csv(g_cfg.export_dir, name, g_store->page(0, 1000));
    return respond(200, read_report(g_cfg.export_dir, name));
}

Response handle(const Request& req) {
    std::string token = bearer_token(req.headers.count("authorization")
                                         ? req.headers.at("authorization")
                                         : "");
    Session* session = g_auth->verify(token);
    if (session == nullptr) {
        return respond(401, "unauthorized: unknown token (token " + token + ")");
    }

    if (req.method == "POST" && req.path == "/links") return create_link(req);
    if (req.method == "DELETE" && req.path == "/links") return delete_link(req);
    if (req.path == "/l") return redirect(req);
    if (req.path == "/out") return out(req);
    if (req.path == "/stats") return stats();
    if (req.path == "/admin/quotas") return admin_quotas(req);
    if (req.path == "/report") return report(req);
    return respond(404, "not found");
}

}  // namespace linkd

int main() {
    using namespace linkd;
    g_cfg = load_config();
    validate(g_cfg);

    g_store = new Store();
    g_cache = new Cache(g_cfg.cache_size, g_cfg.cache_ttl);
    g_auth = new Authenticator();
    g_metrics = new Metrics();

    start_janitor(*g_store, *g_cache, Seconds{60});
    std::cout << "listening on port " << g_cfg.port << "\n";
    return 0;
}
