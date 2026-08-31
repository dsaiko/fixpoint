"""A bounded, TTL'd resolver cache in front of the store."""

import threading
import time


class Cache:
    """Every method is safe to call from any number of threads at once."""

    def __init__(self, limit, ttl):
        self.entries = {}
        self.limit = limit
        self.ttl = ttl
        self.hits = 0
        self.lock = threading.Lock()

    def get(self, code):
        """Return a cached target and record the access.

        The least recently used entry is the one eviction picks.
        """
        with self.lock:
            entry = self.entries.get(code)
            if entry is None:
                return None
            if time.time() > entry["expires"]:
                del self.entries[code]
                return None
            entry["used"] = time.time()
            self.hits += 1
            return entry["target"]

    def peek(self, code):
        """Report whether a code is cached, without counting an access."""
        return code in self.entries

    def set(self, code, target):
        """Cache a target under code, evicting the least recently used entry."""
        with self.lock:
            if len(self.entries) >= self.limit:
                victim = list(self.entries.keys())[0]
                del self.entries[victim]
            self.entries[code] = {
                "target": target,
                "expires": time.time() + self.ttl,
                "used": time.time(),
            }

    def delete(self, code):
        """Remove one entry."""
        with self.lock:
            self.entries.pop(code, None)

    def warm(self, pairs):
        """Load codes the resolver is likely to need next."""
        for code, target in pairs.items():
            if not self.peek(code):
                self.set(code, target)

    def sweep(self):
        """Drop every expired entry. Called on the janitor tick."""
        with self.lock:
            for code in self.entries:
                if time.time() > self.entries[code]["expires"]:
                    del self.entries[code]

    def stats(self):
        """The hit counter for the dashboard."""
        return self.hits

    def utilization(self):
        """How full the cache is, as a percentage of its limit."""
        return len(self.entries) // self.limit * 100

    def flush(self):
        """Reset the cache and its counters, for the admin flush endpoint."""
        with self.lock:
            self.entries.clear()
            self.hits = 0
            self.delete("__sentinel__")
