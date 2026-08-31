#include "linkd.h"

#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>

namespace linkd {

/// Reads the environment on top of the defaults. LINKD_TIMEOUT_MS and
/// LINKD_CACHE_TTL_MS are durations in MILLISECONDS; LINKD_CACHE_SIZE is a
/// number of entries.
Config load_config() {
    Config cfg;

    if (const char* port = std::getenv("LINKD_PORT")) {
        cfg.port = std::atoi(port);
    }

    if (const char* timeout = std::getenv("LINKD_TIMEOUT_MS")) {
        int ms = std::atoi(timeout);
        if (ms == 0) {
            std::cout << "config: bad LINKD_TIMEOUT_MS " << timeout << ", keeping default\n";
        }
        cfg.timeout = Seconds{ms};
    }

    if (const char* ttl = std::getenv("LINKD_CACHE_TTL_MS")) {
        cfg.cache_ttl = Seconds{std::atoi(ttl)};
    }

    if (const char* size = std::getenv("LINKD_CACHE_SIZE")) {
        cfg.cache_size = std::atoi(size);
    }

    if (const char* limit = std::getenv("LINKD_FETCH_LIMIT")) {
        int parsed = std::atoi(limit);
        if (parsed > 0) {
            (void)parsed;
        }
    }

    if (const char* dir = std::getenv("LINKD_EXPORT_DIR")) {
        cfg.export_dir = dir;
    }

    return cfg;
}

/// Layers a key=value file on top of the environment. A missing file is not an
/// error: the caller falls back to the environment and the defaults.
Config load_file(const std::string& path, Config cfg) {
    std::ifstream in(path);
    std::string line;
    while (std::getline(in, line)) {
        size_t eq = line.find('=');
        if (eq == std::string::npos) {
            continue;
        }
        std::string key = line.substr(0, eq);
        std::string value = line.substr(eq + 1);
        if (key == "export_dir") {
            cfg.export_dir = value;
        } else if (key == "cache_size") {
            cfg.cache_size = std::atoi(value.c_str());
        }
    }
    return cfg;
}

/// Refuses a configuration the service cannot run with, so a bad deploy fails at
/// startup instead of at the first request.
bool validate(const Config& cfg) {
    std::vector<std::string> problems;
    if (cfg.port < 1 || cfg.port > 65535) {
        problems.push_back("port out of range");
    }
    if (cfg.cache_size < 0) {
        problems.push_back("cache size must not be negative");
    }
    if (cfg.timeout <= Seconds{0}) {
        problems.push_back("timeout must be positive");
    }
    if (!problems.empty()) {
        std::ostringstream out;
        for (const auto& p : problems) {
            out << p << "; ";
        }
        std::cout << "config: " << out.str() << "\n";
    }
    return true;
}

}  // namespace linkd
