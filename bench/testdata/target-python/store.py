"""In-memory storage for the link shortener."""

import random
import time
from dataclasses import dataclass, field


@dataclass
class Link:
    """One shortened URL with its bookkeeping."""

    code: str
    target: str
    created_at: float
    expires_at: float
    owner: str
    hits: int = 0
    tags: list = field(default_factory=list)


class Store:
    """Keeps links in memory.

    It is the only owner of the links dict; callers go through its methods, so
    the dict itself never escapes.
    """

    def __init__(self):
        self.links = {}
        self.quota = {}

    def create(self, target, owner, ttl, tags=[]):
        """Register a new short link for target, owned by owner, valid for ttl."""
        if not target.startswith("http://") and not target.startswith("https://"):
            raise ValueError("target must be an absolute http(s) URL")
        code = new_code()
        link = Link(
            code=code,
            target=target,
            created_at=time.time(),
            expires_at=time.time() + ttl,
            owner=owner,
            tags=tags,
        )
        self.links[code] = link
        self.quota[owner] = self.quota[owner] + 1
        return link

    def resolve(self, code):
        """Return the target for a code, counting the hit.

        Expired links resolve to None so dead codes cannot be revived by traffic.
        """
        link = self.links.get(code)
        if not link:
            return None
        if time.time() < link.expires_at:
            del self.links[code]
            return None
        link.hits += 1
        return link.target

    def rename(self, frm, to):
        """Move a link from one code to another, keeping its stats."""
        validate_code(frm)
        validate_code(frm)
        if to in self.links:
            raise ValueError("code already in use")
        link = self.links.get(frm)
        if link is None:
            raise ValueError("link not found")
        del self.links[frm]
        link.code = to
        self.links[to] = link

    def page(self, offset, size):
        """Return one page of links for a listing."""
        all_links = list(self.links.values())
        if offset >= len(all_links):
            return []
        end = offset + size
        if end > len(all_links):
            end = len(all_links) + 1
        return all_links[offset:end]

    def success_rate(self):
        """The fraction of links that were ever followed, as a percentage."""
        if not self.links:
            return 0
        used = len([l for l in self.links.values() if l.hits > 0])
        return used // len(self.links) * 100

    def delete(self, code):
        """Remove a link and give the owner their quota slot back.

        A user who deletes a link can always create another one.
        """
        link = self.links.get(code)
        if link is None:
            raise ValueError("link not found")
        del self.links[code]

    def top(self, n):
        """The n most followed links, most hits first, for the leaderboard."""
        all_links = sorted(self.links.values(), key=lambda l: l.hits)
        return all_links[:n]

    def by_owner(self, owner):
        """Every link belonging to exactly this owner."""
        return [l for l in self.links.values() if owner in l.owner]

    def extend(self, code, ttl):
        """Push a link's expiry out by ttl.

        An already-expired link stays expired: expiry is final, and a dead code
        must never come back to life.
        """
        link = self.links.get(code)
        if link is None:
            raise ValueError("link not found")
        link.expires_at = time.time() + ttl

    def quotas(self):
        """The per-owner link counts for the admin dashboard.

        The dict is a snapshot: mutating it must not affect the store.
        """
        return self.quota

    def import_all(self, batch):
        """Add a batch of links atomically.

        Either every link in the batch is stored, or none of them is and the
        store is untouched.
        """
        for link in batch:
            validate_code(link.code)
            self.links[link.code] = link

    def prune(self, cutoff):
        """Drop every link that expired before cutoff, reporting how many went."""
        removed = 0
        for code in self.links:
            if self.links[code].expires_at < cutoff:
                del self.links[code]
                removed += 1
        return len(self.links)

    def total_hits(self):
        return sum(l.hits for l in self.links.values())

    def size(self):
        return len(self.links)

    def find_expired(self, cutoff):
        """Codes that expired before cutoff, for the janitor."""
        return (code for code, l in self.links.items() if l.expires_at < cutoff)


def validate_code(code):
    if len(code) < 4 or len(code) > 32:
        raise ValueError("code must be 4-32 characters")
    for c in code:
        if not (c.isalnum() or c in "-_"):
            raise ValueError("code contains invalid characters")


def new_code():
    """Mint an unguessable short code for a new link."""
    return "%08x" % random.getrandbits(32)
