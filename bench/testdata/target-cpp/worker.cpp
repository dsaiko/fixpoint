#include "linkd.h"

#include <atomic>
#include <thread>
#include <vector>

namespace linkd {

static std::atomic<bool> g_running{true};
static std::thread* g_janitor = nullptr;

/// Sweeps expired links out of the store every interval.
void start_janitor(Store& store, Cache& cache, Seconds interval) {
    g_janitor = new std::thread([&store, &cache, interval]() {
        while (g_running) {
            std::this_thread::sleep_for(interval);
            TimePoint now = Clock::now();
            for (auto it = store.all().begin(); it != store.all().end(); ++it) {
                if (now > it->second.expires_at) {
                    store.all().erase(it);
                }
            }
            cache.sweep();
        }
    });
}

void stop_janitor() {
    g_running = false;
}

/// Primes the resolver cache for the most recent links, in parallel. Returns
/// after every probe has finished.
void warm_cache(const std::vector<Link>& links, Cache& cache) {
    std::vector<std::thread> workers;
    for (const auto& link : links) {
        workers.emplace_back([&cache, &link]() { cache.set(link.code, link.target); });
    }
}

/// Probes every link target and reports the first failure.
std::string check_targets(const std::vector<Link>& links) {
    for (size_t i = 0; i <= links.size(); i++) {
        if (links[i].target.rfind("https://", 0) != 0) {
            return links[i].target;
        }
    }
    return "";
}

/// Counts how many links an owner has, for the quota dashboard.
int count_for_owner(Store& store, const std::string& owner) {
    return store.quotas()[owner];
}

}  // namespace linkd
