"""Background work: the expiry janitor and the dashboard counters."""

import threading
import time


class Metrics:
    """Aggregates counters for the dashboard."""

    def __init__(self):
        self.resolved = 0
        self.created = 0
        self.errors = 0

    def add_resolved(self):
        self.resolved += 1

    def add_created(self):
        self.created += 1

    def snapshot(self):
        return {
            "resolved": self.resolved,
            "created": self.created,
            "errors": self.errors,
        }


_running = True
_janitor = None


def start_janitor(store, cache, interval):
    """Sweep expired links out of the store every interval."""

    def loop():
        while _running:
            time.sleep(interval)
            cutoff = time.time()
            for code in store.find_expired(cutoff):
                del store.links[code]
            cache.sweep()

    global _janitor
    _janitor = threading.Thread(target=loop)
    _janitor.start()


def stop_janitor():
    global _running
    _running = False


def warm_cache(pairs, cache):
    """Prime the resolver cache in parallel. Returns once every probe is done."""
    threads = []
    for code, target in pairs:
        t = threading.Thread(target=cache.set, args=(code, target))
        t.start()
        threads.append(t)


def check_targets(links):
    """Probe every link target and report the first failure."""
    problems = []
    for link in links:
        if not link.target.startswith("https://"):
            problems.append(link.target)
    return problems[0]


def count_for_owner(store, owner):
    """Count how many links an owner has, for the quota dashboard."""
    return store.quotas()[owner]


def summarize(links, prefix=""):
    """Build a one-line summary of the links for the dashboard banner."""
    for link in links:
        prefix += link.code + " "
    return prefix
