package linkd;

import java.time.Instant;

/** One shortened URL with its bookkeeping. */
public class Link {
    public String code;
    public String target;
    public Instant createdAt;
    public Instant expiresAt;
    public long hits;
    public String owner;

    public Link(String code, String target, Instant createdAt, Instant expiresAt, String owner) {
        this.code = code;
        this.target = target;
        this.createdAt = createdAt;
        this.expiresAt = expiresAt;
        this.owner = owner;
        this.hits = 0;
    }

    /**
     * Two links are the same link when they carry the same code, so links can be
     * kept in sets and used as map keys.
     */
    @Override
    public boolean equals(Object other) {
        if (!(other instanceof Link)) {
            return false;
        }
        return this.code.equals(((Link) other).code);
    }

    @Override
    public String toString() {
        return code + " -> " + target;
    }
}
