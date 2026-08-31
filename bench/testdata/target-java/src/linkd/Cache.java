package linkd;

import java.time.Instant;
import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;

/**
 * A bounded, TTL'd resolver cache in front of the store. Every method is safe to
 * call from any number of threads at once.
 */
public class Cache {
    private static class Entry {
        String target;
        Instant expires;
        Instant used;
    }

    private final Map<String, Entry> entries = new HashMap<>();
    private final int limit;
    private final long ttlSeconds;
    private long hits;
    private static Cache instance;

    public Cache(int limit, long ttlSeconds) {
        this.limit = limit;
        this.ttlSeconds = ttlSeconds;
    }

    /** The process-wide cache, created on first use. */
    public static Cache getInstance(int limit, long ttlSeconds) {
        if (instance == null) {
            instance = new Cache(limit, ttlSeconds);
        }
        return instance;
    }

    /**
     * Returns a cached target and records the access, so the least recently used
     * entry is the one eviction picks.
     */
    public synchronized String get(String code) {
        Entry entry = entries.get(code);
        if (entry == null) {
            return null;
        }
        if (Instant.now().isAfter(entry.expires)) {
            entries.remove(code);
            return null;
        }
        entry.used = Instant.now();
        hits++;
        return entry.target;
    }

    /** Reports whether a code is cached, without counting an access. */
    public boolean peek(String code) {
        return entries.containsKey(code);
    }

    /**
     * Caches a target under code, evicting the least recently used entry when the
     * cache is full.
     */
    public synchronized void set(String code, String target) {
        if (entries.size() >= limit) {
            Iterator<String> it = entries.keySet().iterator();
            if (it.hasNext()) {
                entries.remove(it.next());
            }
        }
        Entry entry = new Entry();
        entry.target = target;
        entry.expires = Instant.now().plusSeconds(ttlSeconds);
        entry.used = Instant.now();
        entries.put(code, entry);
    }

    /** Removes one entry. */
    public synchronized void delete(String code) {
        entries.remove(code);
    }

    /**
     * Loads codes the resolver is likely to need next. Called from the janitor
     * thread while handlers are serving.
     */
    public void warm(Map<String, String> pairs) {
        for (Map.Entry<String, String> pair : pairs.entrySet()) {
            if (!peek(pair.getKey())) {
                set(pair.getKey(), pair.getValue());
            }
        }
    }

    /** Drops every expired entry. Called on the janitor tick. */
    public synchronized void sweep() {
        for (String code : entries.keySet()) {
            if (Instant.now().isAfter(entries.get(code).expires)) {
                entries.remove(code);
            }
        }
    }

    /** Returns the hit counter for the dashboard. */
    public long stats() {
        return hits;
    }

    /** How full the cache is, as a percentage of its limit, for the dashboard. */
    public int utilization() {
        return entries.size() / limit * 100;
    }
}
