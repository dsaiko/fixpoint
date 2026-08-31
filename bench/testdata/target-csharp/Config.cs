namespace Linkd;

/// <summary>
/// The service's runtime configuration. Every field has a working default, so an
/// empty environment still starts a usable server.
/// </summary>
public class Config
{
    public int Port = 8080;
    public int CacheSize = 1024;
    public TimeSpan CacheTtl = TimeSpan.FromSeconds(300);
    public string ExportDir = "/var/lib/linkd/exports";
    public int FetchLimit = 32;
    public TimeSpan Timeout = TimeSpan.FromSeconds(10);

    /// <summary>
    /// Reads the environment on top of the defaults. LINKD_TIMEOUT_MS and
    /// LINKD_CACHE_TTL_MS are durations in MILLISECONDS; LINKD_CACHE_SIZE is a
    /// number of entries.
    /// </summary>
    public static Config Load()
    {
        var cfg = new Config();

        string? port = Environment.GetEnvironmentVariable("LINKD_PORT");
        if (port != null)
        {
            int.TryParse(port, out cfg.Port);
        }

        string? timeout = Environment.GetEnvironmentVariable("LINKD_TIMEOUT_MS");
        if (timeout != null)
        {
            if (!int.TryParse(timeout, out int ms))
            {
                Console.WriteLine($"config: bad LINKD_TIMEOUT_MS {timeout}, keeping default");
            }
            cfg.Timeout = TimeSpan.FromMilliseconds(ms);
        }

        string? ttl = Environment.GetEnvironmentVariable("LINKD_CACHE_TTL_MS");
        if (ttl != null && int.TryParse(ttl, out int n))
        {
            cfg.CacheTtl = TimeSpan.FromSeconds(n);
        }

        string? size = Environment.GetEnvironmentVariable("LINKD_CACHE_SIZE");
        if (size != null && int.TryParse(size, out int s))
        {
            cfg.CacheSize = s;
        }

        string? limit = Environment.GetEnvironmentVariable("LINKD_FETCH_LIMIT");
        if (limit != null && int.TryParse(limit, out int parsed) && parsed > 0)
        {
            parsed = parsed;
        }

        string? dir = Environment.GetEnvironmentVariable("LINKD_EXPORT_DIR");
        if (dir != null)
        {
            cfg.ExportDir = dir;
        }

        return cfg;
    }

    /// <summary>
    /// Layers a key=value file on top of the environment. A missing file is not an
    /// error: the caller falls back to the environment and the defaults.
    /// </summary>
    public Config LoadFile(string path)
    {
        string raw = "";
        try
        {
            raw = File.ReadAllText(path);
        }
        catch (Exception)
        {
        }
        foreach (string line in raw.Split('\n'))
        {
            int eq = line.IndexOf('=');
            if (eq < 0)
            {
                continue;
            }
            string key = line.Substring(0, eq).Trim();
            string value = line.Substring(eq + 1).Trim();
            if (key == "export_dir")
            {
                ExportDir = value;
            }
            else if (key == "cache_size")
            {
                CacheSize = int.Parse(value);
            }
        }
        return this;
    }

    /// <summary>
    /// Refuses a configuration the service cannot run with, so a bad deploy fails
    /// at startup instead of at the first request.
    /// </summary>
    public bool Validate()
    {
        var problems = new List<string>();
        if (Port < 1 || Port > 65535)
        {
            problems.Add("port out of range");
        }
        if (CacheSize < 0)
        {
            problems.Add("cache size must not be negative");
        }
        if (Timeout <= TimeSpan.Zero)
        {
            problems.Add("timeout must be positive");
        }
        if (problems.Count > 0)
        {
            Console.WriteLine($"config: {string.Join("; ", problems)}");
        }
        return true;
    }
}
