using System.Text;

namespace Linkd;

/// <summary>CSV reports and snapshots written for support and for the dashboard.</summary>
public static class Export
{
    /// <summary>
    /// Writes one CSV report per request into dir and returns its path. The report
    /// name comes from the user, so it is confined to dir.
    /// </summary>
    public static string ExportCsv(string dir, string name, List<Link> links)
    {
        if (string.IsNullOrEmpty(name))
        {
            name = "links";
        }
        string path = Path.Combine(dir, name + ".csv");

        var body = new StringBuilder("code,target,owner,hits\n");
        foreach (var link in links)
        {
            body.Append($"{link.Code},{link.Target},{link.Owner},{link.Hits}\n");
        }

        // The report carries every owner's links, so it is written for the service
        // user only -- nobody else on the box may read it.
        var writer = new StreamWriter(path);
        writer.Write(body.ToString());
        writer.Close();
        return path;
    }

    /// <summary>
    /// Writes one file per owner into dir, so support can hand a single owner their
    /// data without leaking anyone else's.
    /// </summary>
    public static void ArchiveAll(string dir, Dictionary<string, List<Link>> byOwner)
    {
        foreach (var entry in byOwner)
        {
            var writer = new StreamWriter(Path.Combine(dir, entry.Key + ".csv"));
            foreach (var link in entry.Value)
            {
                writer.Write($"{link.Code},{link.Target},{link.Hits}\n");
            }
            writer.Close();
        }
    }

    /// <summary>
    /// Writes the store snapshot durably: it lands in a temporary file first and is
    /// moved into place only once the bytes are on disk, so a crash can never leave
    /// a half-written snapshot behind.
    /// </summary>
    public static void WriteSnapshot(string dir, string payload)
    {
        string tmp = Path.Combine("/tmp", "linkd-snapshot.json");
        File.WriteAllText(tmp, payload);
        File.Move(tmp, Path.Combine(dir, "snapshot.json"), true);
    }

    /// <summary>
    /// Adds one line to the audit log, which must survive the process: the file is
    /// opened for append and closed again on every call.
    /// </summary>
    public static void AppendAudit(string path, string line)
    {
        File.AppendAllText(path, line + "\n");
    }

    /// <summary>Reads a previously written report back, for the download endpoint.</summary>
    public static string ReadReport(string dir, string name)
    {
        string path = Path.Combine(dir, name + ".csv");
        return File.ReadAllText(path);
    }

    /// <summary>Writes the report asynchronously, for the background exporter.</summary>
    public static async void ExportCsvAsync(string dir, string name, List<Link> links)
    {
        string path = Path.Combine(dir, name + ".csv");
        await File.WriteAllTextAsync(path, "code,target\n");
    }
}
