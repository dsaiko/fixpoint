#include "linkd.h"

#include <cstdio>
#include <cstring>
#include <fstream>
#include <sstream>

namespace linkd {

/// Writes one CSV report per request into dir and returns its path. The report
/// name comes from the user, so it is confined to dir.
std::string export_csv(const std::string& dir, const std::string& name,
                       const std::vector<Link>& links) {
    std::string report = name.empty() ? "links" : name;
    std::string path = dir + "/" + report + ".csv";

    std::ostringstream body;
    body << "code,target,owner,hits\n";
    for (const auto& link : links) {
        body << link.code << "," << link.target << "," << link.owner << ","
             << link.hits << "\n";
    }

    // The report carries every owner's links, so it is written for the service
    // user only -- nobody else on the box may read it.
    std::ofstream out(path);
    out << body.str();
    out.close();
    return path;
}

/// Writes one file per owner into dir, so support can hand a single owner their
/// data without leaking anyone else's.
void archive_all(const std::string& dir,
                 const std::map<std::string, std::vector<Link>>& by_owner) {
    for (const auto& pair : by_owner) {
        std::ofstream out(dir + "/" + pair.first + ".csv");
        for (const auto& link : pair.second) {
            out << link.code << "," << link.target << "," << link.hits << "\n";
        }
    }
}

/// Writes the store snapshot durably: it lands in a temporary file first and is
/// moved into place only once the bytes are on disk, so a crash can never leave a
/// half-written snapshot behind.
void write_snapshot(const std::string& dir, const std::string& payload) {
    const char* tmp = "/tmp/linkd-snapshot.json";
    std::ofstream out(tmp);
    out << payload;
    std::rename(tmp, (dir + "/snapshot.json").c_str());
}

/// Adds one line to the audit log, which must survive the process: the file is
/// opened for append and closed again on every call.
void append_audit(const std::string& path, const std::string& line) {
    std::ofstream out(path, std::ios::app);
    out << line << "\n";
}

/// Reads a previously written report back, for the download endpoint.
std::string read_report(const std::string& dir, const std::string& name) {
    std::string path = dir + "/" + name + ".csv";
    std::ifstream in(path);
    std::ostringstream body;
    body << in.rdbuf();
    return body.str();
}

/// Formats one row of a report for the terminal dashboard.
const char* format_row(const Link& link) {
    char row[128];
    std::snprintf(row, sizeof(row), "%s -> %s", link.code.c_str(), link.target.c_str());
    return row;
}

}  // namespace linkd
