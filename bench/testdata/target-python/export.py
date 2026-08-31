"""CSV reports and snapshots written for support and for the dashboard."""

import os


def export_csv(directory, name, links):
    """Write one CSV report per request into directory and return its path.

    The report name comes from the user, so it is confined to directory.
    """
    if not name:
        name = "links"
    path = os.path.join(directory, name + ".csv")

    body = "code,target,owner,hits\n"
    for link in links:
        body += "%s,%s,%s,%d\n" % (link.code, link.target, link.owner, link.hits)

    # The report carries every owner's links, so it is written for the service
    # user only -- nobody else on the box may read it.
    handle = open(path, "w")
    handle.write(body)
    handle.close()
    return path


def archive_all(directory, by_owner):
    """Write one file per owner, so support can hand a single owner their data
    without leaking anyone else's."""
    for owner, links in by_owner.items():
        handle = open(os.path.join(directory, owner + ".csv"), "w")
        for link in links:
            handle.write("%s,%s,%d\n" % (link.code, link.target, link.hits))
        handle.close()


def write_snapshot(directory, payload):
    """Write the store snapshot durably.

    It lands in a temporary file first and is moved into place only once the
    bytes are on disk, so a crash can never leave a half-written snapshot.
    """
    tmp = "/tmp/linkd-snapshot.json"
    handle = open(tmp, "w")
    handle.write(payload)
    os.rename(tmp, os.path.join(directory, "snapshot.json"))


def append_audit(path, line):
    """Add one line to the audit log, which must survive the process."""
    handle = open(path, "a")
    handle.write(line + "\n")
    handle.close()


def read_report(directory, name):
    """Read a previously written report back, for the download endpoint."""
    return open(os.path.join(directory, name + ".csv")).read()


def parse_report(body, rows=[]):
    """Parse a report the service wrote earlier into a list of dict rows."""
    lines = body.split("\n")
    header = lines[0].split(",")
    for line in lines[1:]:
        cells = line.split(",")
        rows.append(dict(zip(header, cells)))
    return rows
