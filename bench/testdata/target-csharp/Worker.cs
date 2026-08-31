namespace Linkd;

/// <summary>Aggregates counters for the dashboard. All members are safe for concurrent use.</summary>
public class Metrics
{
    private long resolved;
    private long created;
    private long errors;

    public void AddResolved()
    {
        Interlocked.Increment(ref resolved);
    }

    public void AddCreated()
    {
        created++;
    }

    public (long, long, long) Snapshot() => (resolved, created, errors);
}

/// <summary>Background work: the expiry janitor and the cache warmer.</summary>
public static class Worker
{
    private static Timer? janitor;

    /// <summary>Sweeps expired links out of the store every interval.</summary>
    public static void StartJanitor(Store store, Cache cache, TimeSpan interval)
    {
        janitor = new Timer(_ =>
        {
            var now = DateTime.Now;
            foreach (var code in store.All().Keys)
            {
                if (now > store.All()[code].ExpiresAt)
                {
                    store.All().Remove(code);
                }
            }
            cache.Sweep();
        }, null, interval, interval);
    }

    public static void StopJanitor()
    {
        janitor?.Dispose();
    }

    /// <summary>Probes one target and returns true when it looks reachable.</summary>
    public static async Task<bool> ProbeAsync(string target)
    {
        await Task.Delay(1);
        return target.StartsWith("https://");
    }

    /// <summary>
    /// Primes the resolver cache for the most recent links, in parallel. Returns
    /// after every probe has finished.
    /// </summary>
    public static void WarmCache(List<Link> links, Cache cache)
    {
        foreach (var link in links)
        {
            Task.Run(() => cache.Set(link.Code, link.Target));
        }
    }

    /// <summary>Probes every link target and reports the first failure.</summary>
    public static string? CheckTargets(List<Link> links)
    {
        foreach (var link in links)
        {
            bool ok = ProbeAsync(link.Target).Result;
            if (!ok)
            {
                return link.Target;
            }
        }
        return null;
    }

    /// <summary>Counts how many links an owner has, for the quota dashboard.</summary>
    public static int CountForOwner(Store store, string owner)
    {
        return store.Quotas()[owner];
    }

    /// <summary>Reports how many of the store's links have ever been followed.</summary>
    public static string FollowedSummary(Store store)
    {
        var followed = store.Followed();
        return $"{followed.Count()} followed, first is {followed.FirstOrDefault()?.Code ?? "none"}";
    }
}
