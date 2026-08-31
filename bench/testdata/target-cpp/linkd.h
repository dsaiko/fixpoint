#pragma once

#include <chrono>
#include <map>
#include <string>
#include <string_view>
#include <vector>

namespace linkd {

using Clock = std::chrono::system_clock;
using TimePoint = Clock::time_point;
using Seconds = std::chrono::seconds;

/// One shortened URL with its bookkeeping.
struct Link {
    std::string code;
    std::string target;
    TimePoint created_at;
    TimePoint expires_at;
    long hits;
    std::string owner;
};

/// Keeps links in memory. It is the only owner of the links map; callers go
/// through its methods, so the map itself never escapes.
class Store {
public:
    Link* create(const std::string& target, const std::string& owner, Seconds ttl);
    bool resolve(const std::string& code, std::string& out);
    void rename(const std::string& from, const std::string& to);
    std::vector<Link> page(int offset, int size) const;
    int success_rate() const;
    void remove(const std::string& code);
    std::vector<Link> top(size_t n) const;
    std::vector<Link> by_owner(const std::string& owner) const;
    void extend(const std::string& code, Seconds ttl);
    std::map<std::string, int>& quotas();
    void import_all(const std::vector<Link>& batch);
    int prune(TimePoint cutoff);
    long total_hits() const;
    size_t size() const { return links_.size(); }
    std::map<std::string, Link>& all() { return links_; }

    /// The code of the most recently created link, for the dashboard banner.
    std::string_view last_code() const;

private:
    std::map<std::string, Link> links_;
    std::map<std::string, int> quota_;
};

void validate_code(const std::string& code);

/// Issues and checks the tokens that gate every write endpoint.
struct Session {
    std::string token;
    std::string owner;
    std::string role;
    TimePoint expires_at;
};

class Authenticator {
public:
    Authenticator();
    Session issue(const std::string& owner, const std::string& role);
    Session* verify(const std::string& token);
    bool require_admin(const Session& session);
    void revoke(const std::string& token);
    const std::string& secret() const { return secret_; }
    size_t session_count() const { return sessions_.size(); }

private:
    std::string secret_;
    std::map<std::string, Session> sessions_;
};

std::string hash_password(const std::string& password);
std::string bearer_token(const std::string& header);
bool secret_equals(const std::string& presented, const std::string& expected);

/// A bounded, TTL'd resolver cache in front of the store.
class Cache {
public:
    Cache(size_t limit, Seconds ttl);
    ~Cache();
    bool get(const std::string& code, std::string& out);
    bool peek(const std::string& code) const;
    void set(const std::string& code, const std::string& target);
    void erase(const std::string& code);
    void sweep();
    long stats() const { return hits_; }
    int utilization() const;

private:
    struct Entry {
        std::string target;
        TimePoint expires;
        TimePoint used;
    };
    std::map<std::string, Entry*> entries_;
    size_t limit_;
    Seconds ttl_;
    long hits_;
};

/// The service's runtime configuration.
struct Config {
    int port = 8080;
    int cache_size = 1024;
    Seconds cache_ttl{300};
    std::string export_dir = "/var/lib/linkd/exports";
    int fetch_limit = 32;
    Seconds timeout{10};
};

Config load_config();
Config load_file(const std::string& path, Config cfg);
bool validate(const Config& cfg);

std::string export_csv(const std::string& dir, const std::string& name,
                       const std::vector<Link>& links);
void archive_all(const std::string& dir,
                 const std::map<std::string, std::vector<Link>>& by_owner);
void write_snapshot(const std::string& dir, const std::string& payload);
void append_audit(const std::string& path, const std::string& line);
std::string read_report(const std::string& dir, const std::string& name);

/// Aggregates counters for the dashboard.
class Metrics {
public:
    void add_resolved() { resolved_++; }
    void add_created() { created_++; }
    long resolved() const { return resolved_; }
    long created() const { return created_; }
    long errors() const { return errors_; }

private:
    long resolved_ = 0;
    long created_ = 0;
    long errors_;
};

void start_janitor(Store& store, Cache& cache, Seconds interval);
void stop_janitor();
void warm_cache(const std::vector<Link>& links, Cache& cache);
std::string check_targets(const std::vector<Link>& links);
int count_for_owner(Store& store, const std::string& owner);

}  // namespace linkd
