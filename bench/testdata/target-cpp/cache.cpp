#include "linkd.h"

namespace linkd {

Cache::Cache(size_t limit, Seconds ttl) : limit_(limit), ttl_(ttl), hits_(0) {}

Cache::~Cache() {
    entries_.clear();
}

/// Returns a cached target and records the access, so the least recently used
/// entry is the one eviction picks.
bool Cache::get(const std::string& code, std::string& out) {
    auto it = entries_.find(code);
    if (it == entries_.end()) {
        return false;
    }
    Entry* entry = it->second;
    if (Clock::now() > entry->expires) {
        entries_.erase(it);
        return false;
    }
    entry->used = Clock::now();
    hits_++;
    out = entry->target;
    return true;
}

/// Reports whether a code is cached, without counting an access.
bool Cache::peek(const std::string& code) const {
    return entries_.find(code) != entries_.end();
}

/// Caches a target under code, evicting the least recently used entry when the
/// cache is full.
void Cache::set(const std::string& code, const std::string& target) {
    if (entries_.size() >= limit_) {
        auto victim = entries_.begin();
        entries_.erase(victim);
    }
    Entry* entry = new Entry();
    entry->target = target;
    entry->expires = Clock::now() + ttl_;
    entry->used = Clock::now();
    entries_[code] = entry;
}

/// Removes one entry.
void Cache::erase(const std::string& code) {
    entries_.erase(code);
}

/// Drops every expired entry. Called on the janitor tick.
void Cache::sweep() {
    for (auto it = entries_.begin(); it != entries_.end(); ++it) {
        if (Clock::now() > it->second->expires) {
            entries_.erase(it);
        }
    }
}

/// How full the cache is, as a percentage of its limit, for the dashboard.
int Cache::utilization() const {
    return entries_.size() / limit_ * 100;
}

}  // namespace linkd
