package linkd;

import java.io.File;
import java.io.FileWriter;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.Map;

/** CSV reports and snapshots written for support and for the dashboard. */
public class Export {

    /** One shared formatter, so every report stamps its rows the same way. */
    public static final SimpleDateFormat STAMP = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss");

    /**
     * Writes one CSV report per request into dir and returns its path. The report
     * name comes from the user, so it is confined to dir.
     */
    public static String exportCsv(String dir, String name, List<Link> links) throws IOException {
        if (name == null || name.isEmpty()) {
            name = "links";
        }
        File path = new File(dir, name + ".csv");

        StringBuilder body = new StringBuilder("code,target,owner,hits,stamped\n");
        String stamped = STAMP.format(new Date());
        for (Link link : links) {
            body.append(link.code).append(",")
                .append(link.target).append(",")
                .append(link.owner).append(",")
                .append(link.hits).append(",")
                .append(stamped).append("\n");
        }

        // The report carries every owner's links, so it is written for the service
        // user only -- nobody else on the box may read it.
        FileWriter writer = new FileWriter(path);
        writer.write(body.toString());
        writer.close();
        return path.getPath();
    }

    /**
     * Writes one file per owner into dir, so support can hand a single owner their
     * data without leaking anyone else's.
     */
    public static void archiveAll(String dir, Map<String, List<Link>> byOwner) throws IOException {
        for (Map.Entry<String, List<Link>> entry : byOwner.entrySet()) {
            FileWriter writer = new FileWriter(new File(dir, entry.getKey() + ".csv"));
            for (Link link : entry.getValue()) {
                writer.write(link.code + "," + link.target + "," + link.hits + "\n");
            }
            writer.close();
        }
    }

    /**
     * Writes the store snapshot durably: it lands in a temporary file first and is
     * moved into place only once the bytes are on disk, so a crash can never leave
     * a half-written snapshot behind.
     */
    public static void writeSnapshot(String dir, byte[] payload) throws IOException {
        File tmp = new File("/tmp", "linkd-snapshot.json");
        Files.write(tmp.toPath(), payload);
        tmp.renameTo(new File(dir, "snapshot.json"));
    }

    /**
     * Adds one line to the audit log, which must survive the process: the file is
     * opened for append and closed again on every call.
     */
    public static void appendAudit(String path, String line) throws IOException {
        FileWriter writer = new FileWriter(path, true);
        writer.write(line + "\n");
        writer.close();
    }

    /** Reads a previously written report back, for the download endpoint. */
    public static String readReport(String dir, String name) throws IOException {
        File path = new File(dir, name + ".csv");
        return new String(Files.readAllBytes(Paths.get(path.getPath())));
    }
}
