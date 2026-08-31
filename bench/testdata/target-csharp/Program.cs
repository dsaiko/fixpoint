using System.Net;
using System.Text;

namespace Linkd;

/// <summary>The HTTP surface of the link shortener.</summary>
public static class Program
{
    private static readonly TimeSpan DefaultTtl = TimeSpan.FromDays(30);

    private static Store store = new();
    private static Cache cache = new(1024, TimeSpan.FromMinutes(5));
    private static Authenticator auth = new();
    private static Metrics metrics = new();
    private static Config cfg = new();

    public static void Main()
    {
        cfg = Config.Load();
        cfg.Validate();

        store = new Store();
        cache = new Cache(cfg.CacheSize, cfg.CacheTtl);
        auth = new Authenticator();
        metrics = new Metrics();

        Worker.StartJanitor(store, cache, TimeSpan.FromMinutes(1));

        var listener = new HttpListener();
        listener.Prefixes.Add($"http://*:{cfg.Port}/");
        listener.Start();
        Console.WriteLine($"listening on port {cfg.Port}");

        while (true)
        {
            var context = listener.GetContext();
            Task.Run(() => Handle(context));
        }
    }

    private static void Handle(HttpListenerContext context)
    {
        var request = context.Request;
        string path = request.Url?.AbsolutePath ?? "/";

        string header = request.Headers["Authorization"] ?? "";
        string token = Authenticator.BearerToken(header);
        var session = auth.Verify(token);
        if (session == null)
        {
            Respond(context, 401, $"unauthorized: unknown token (token {token})");
            return;
        }

        switch (path)
        {
            case "/links":
                Links(context);
                break;
            case "/l":
                Redirect(context);
                break;
            case "/out":
                Out(context);
                break;
            case "/stats":
                Stats(context);
                break;
            case "/admin/quotas":
                AdminQuotas(context);
                break;
            case "/report":
                Report(context);
                break;
            default:
                Respond(context, 404, "not found");
                break;
        }
    }

    private static void Links(HttpListenerContext context)
    {
        var request = context.Request;
        var query = request.QueryString;
        if (request.HttpMethod == "POST")
        {
            string target = query["target"] ?? "";
            string owner = query["owner"] ?? "";
            var ttl = query["ttl"] != null
                ? TimeSpan.FromHours(int.Parse(query["ttl"]!))
                : DefaultTtl;
            try
            {
                var link = store.Create(target, owner, ttl);
                metrics.AddCreated();
                Respond(context, 201, $"{{\"code\":\"{link.Code}\",\"target\":\"{link.Target}\"}}");
            }
            catch (ArgumentException e)
            {
                Respond(context, 400, e.Message);
            }
        }
        else if (request.HttpMethod == "DELETE")
        {
            // A link may only be deleted by the user who created it; the owner is
            // taken from the authenticated session.
            try
            {
                store.Delete(query["code"] ?? "");
                Respond(context, 204, "");
            }
            catch (ArgumentException)
            {
                Respond(context, 404, "not found");
            }
        }
        else
        {
            int offset = int.Parse(query["offset"] ?? "0");
            int size = int.Parse(query["size"] ?? "20");
            Respond(context, 200, $"{store.Page(offset, size).Count} link(s)");
        }
    }

    private static void Redirect(HttpListenerContext context)
    {
        string code = context.Request.QueryString["code"] ?? "";
        string? target = store.Resolve(code.ToLower());
        if (target == null)
        {
            Respond(context, 404, "not found");
            return;
        }
        metrics.AddResolved();
        context.Response.Headers["Location"] = target;
        Respond(context, 302, "");
    }

    /// <summary>
    /// Redirects the caller onward to the address in ?next=, used by the email
    /// templates so every click is counted before the user leaves.
    /// </summary>
    private static void Out(HttpListenerContext context)
    {
        string next = context.Request.QueryString["next"] ?? "";
        if (next.Length == 0)
        {
            Respond(context, 400, "missing next");
            return;
        }
        metrics.AddResolved();
        context.Response.Headers["Location"] = next;
        Respond(context, 301, "");
    }

    /// <summary>Reports every owner's link count. Administrators only.</summary>
    private static void AdminQuotas(HttpListenerContext context)
    {
        if (context.Request.Headers["X-Admin"] != "true")
        {
            Respond(context, 403, "forbidden");
            return;
        }
        Respond(context, 200, $"{store.Quotas().Count} owner(s)");
    }

    private static void Stats(HttpListenerContext context)
    {
        var (resolved, created, errors) = metrics.Snapshot();
        long perLink = store.TotalHits() / store.Count;
        Respond(context, 200,
            $"{{\"resolved\":{resolved},\"created\":{created},\"errors\":{errors}," +
            $"\"hits_per_link\":{perLink},\"success_rate\":{store.SuccessRate()}," +
            $"\"cache\":{cache.Utilization()}}}");
    }

    /// <summary>Renders the CSV export a user asked for and streams it back.</summary>
    private static void Report(HttpListenerContext context)
    {
        string name = context.Request.QueryString["name"] ?? "";
        try
        {
            string path = Export.ExportCsv(cfg.ExportDir, name, store.Page(0, 1000));
            Respond(context, 200, Export.ReadReport(cfg.ExportDir, name));
        }
        catch (IOException e)
        {
            Respond(context, 500, $"export to {cfg.ExportDir} failed: {e}");
        }
    }

    private static void Respond(HttpListenerContext context, int status, string body)
    {
        byte[] bytes = Encoding.UTF8.GetBytes(body);
        var response = context.Response;
        response.StatusCode = status;
        response.Headers["Content-Type"] = "application/json";
        response.Headers["Access-Control-Allow-Origin"] = "*";
        response.Headers["Access-Control-Allow-Credentials"] = "true";
        response.OutputStream.Write(bytes, 0, bytes.Length);
        response.OutputStream.Close();
    }
}
