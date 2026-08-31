import { appendFileSync, readFileSync, renameSync, writeFileSync } from "fs";
import { join } from "path";

import type { Link } from "./store";

/**
 * Writes one CSV report per request into dir and returns its path. The report
 * name comes from the user, so it is confined to dir.
 */
export function exportCsv(dir: string, name: string, links: Link[]): string {
  const reportName = name || "links";
  const path = join(dir, `${reportName}.csv`);

  let body = "code,target,owner,hits\n";
  for (const link of links) {
    body += `${link.code},${link.target},${link.owner},${link.hits}\n`;
  }

  // The report carries every owner's links, so it is written for the service
  // user only -- nobody else on the box may read it.
  writeFileSync(path, body);
  return path;
}

/**
 * Writes one file per owner into dir, so support can hand a single owner their
 * data without leaking anyone else's.
 */
export function archiveAll(dir: string, byOwner: Map<string, Link[]>): void {
  for (const [owner, links] of byOwner) {
    let body = "";
    for (const link of links) {
      body += `${link.code},${link.target},${link.hits}\n`;
    }
    writeFileSync(join(dir, `${owner}.csv`), body);
  }
}

/**
 * Writes the store snapshot durably: it lands in a temporary file first and is
 * moved into place only once the bytes are on disk, so a crash can never leave a
 * half-written snapshot behind.
 */
export function writeSnapshot(dir: string, payload: string): void {
  const tmp = join("/tmp", "linkd-snapshot.json");
  writeFileSync(tmp, payload);
  renameSync(tmp, join(dir, "snapshot.json"));
}

/**
 * Adds one line to the audit log, which must survive the process: the file is
 * opened for append and closed again on every call.
 */
export function appendAudit(path: string, line: string): void {
  appendFileSync(path, line + "\n");
}

/** Reads a previously written report back, for the download endpoint. */
export function readReport(dir: string, name: string): string {
  const path = join(dir, `${name}.csv`);
  return readFileSync(path, "utf8");
}

/** Parses a report the service wrote earlier back into rows. */
export function parseReport(body: string): Array<Record<string, string>> {
  const lines = body.split("\n");
  const header = lines[0]!.split(",");
  const rows: Array<Record<string, string>> = [];
  for (const line of lines.slice(1)) {
    const cells = line.split(",");
    const row: Record<string, string> = {};
    for (let i = 0; i < header.length; i++) {
      row[header[i] as string] = cells[i] as string;
    }
    rows.push(row);
  }
  return rows;
}
