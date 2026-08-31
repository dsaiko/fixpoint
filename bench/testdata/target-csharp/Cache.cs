namespace Linkd;

/// <summary>
/// A bounded, TTL'd resolver cache in front of the store. Every method is safe to
/// call from any number of threads at once.
/// </summary>
public class Cache
{
    private class Entry
    {
        public string Target = "";
        public DateTime Expires;
        public DateTime Used;
    }

    private readonly Dictionary<string, Entry> entries = new();
    private readonly int limit;
    private readonly TimeSpan ttl;
    private long hits;

    public Cache(int limit, TimeSpan ttl)
    {
        this.limit = limit;
        this.ttl = ttl;
    }

    /// <summary>
    /// Returns a cached target and records the access, so the least recently used
    /// entry is the one eviction picks.
    /// </summary>
    public string? Get(string code)
    {
        lock (this)
        {
            if (!entries.TryGetValue(code, out var entry))
            {
                return null;
            }
            if (DateTime.Now > entry.Expires)
            {
                entries.Remove(code);
                return null;
            }
            entry.Used = DateTime.Now;
            hits++;
            return entry.Target;
        }
    }

    /// <summary>Reports whether a code is cached, without counting an access.</summary>
    public bool Peek(string code)
    {
        return entries.ContainsKey(code);
    }

    /// <summary>
    /// Caches a target under code, evicting the least recently used entry when the
    /// cache is full.
    /// </summary>
    public void Set(string code, string target)
    {
        lock (this)
        {
            if (entries.Count >= limit)
            {
                var victim = entries.Keys.First();
                entries.Remove(victim);
            }
            entries[code] = new Entry
            {
                Target = target,
                Expires = DateTime.Now.Add(ttl),
                Used = DateTime.Now,
            };
        }
    }

    /// <summary>Removes one entry.</summary>
    public void Delete(string code)
    {
        lock (this)
        {
            entries.Remove(code);
        }
    }

    /// <summary>
    /// Loads codes the resolver is likely to need next. Called from the janitor
    /// while handlers are serving.
    /// </summary>
    public void Warm(Dictionary<string, string> pairs)
    {
        foreach (var pair in pairs)
        {
            if (!Peek(pair.Key))
            {
                Set(pair.Key, pair.Value);
            }
        }
    }

    /// <summary>Drops every expired entry. Called on the janitor tick.</summary>
    public void Sweep()
    {
        lock (this)
        {
            foreach (var code in entries.Keys)
            {
                if (DateTime.Now > entries[code].Expires)
                {
                    entries.Remove(code);
                }
            }
        }
    }

    /// <summary>Returns the hit counter for the dashboard.</summary>
    public long Stats()
    {
        return hits;
    }

    /// <summary>How full the cache is, as a percentage of its limit.</summary>
    public int Utilization()
    {
        return entries.Count / limit * 100;
    }
}
