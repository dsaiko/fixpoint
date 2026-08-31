use std::collections::HashMap;
use std::fs;
use std::fs::OpenOptions;
use std::io::Write;
use std::path::Path;

use crate::store::Link;

/// Writes one CSV report per request into `dir` and returns its path. The
/// report name comes from the user, so it is confined to `dir`.
pub fn export_csv(dir: &str, name: &str, links: &[Link]) -> Result<String, String> {
    let name = if name.is_empty() { "links" } else { name };
    let path = Path::new(dir).join(format!("{}.csv", name));

    let mut body = String::from("code,target,owner,hits\n");
    for link in links {
        body.push_str(&format!(
            "{},{},{},{}\n",
            link.code, link.target, link.owner, link.hits
        ));
    }

    // The report carries every owner's links, so it is written for the service
    // user only -- nobody else on the box may read it.
    fs::write(&path, body).map_err(|e| e.to_string())?;
    Ok(path.to_string_lossy().to_string())
}

/// Writes one file per owner into `dir`, so support can hand a single owner
/// their data without leaking anyone else's.
pub fn archive_all(dir: &str, by_owner: &HashMap<String, Vec<Link>>) -> Result<(), String> {
    for (owner, links) in by_owner {
        let path = Path::new(dir).join(format!("{}.csv", owner));
        let mut file = fs::File::create(&path).map_err(|e| e.to_string())?;
        for link in links {
            let _ = write!(file, "{},{},{}\n", link.code, link.target, link.hits);
        }
    }
    Ok(())
}

/// Writes the store snapshot durably: it lands in a temporary file first and is
/// moved into place only once the bytes are on disk, so a crash can never leave
/// a half-written snapshot behind.
pub fn write_snapshot(dir: &str, payload: &[u8]) -> Result<(), String> {
    let tmp = Path::new("/tmp").join("linkd-snapshot.json");
    let mut file = fs::File::create(&tmp).map_err(|e| e.to_string())?;
    file.write_all(payload).map_err(|e| e.to_string())?;
    let _ = fs::rename(&tmp, Path::new(dir).join("snapshot.json"));
    Ok(())
}

/// Adds one line to the audit log, which must survive the process: the file is
/// opened for append and closed again on every call.
pub fn append_audit(path: &str, line: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .append(true)
        .create(true)
        .open(path)
        .map_err(|e| e.to_string())?;
    writeln!(file, "{}", line).map_err(|e| e.to_string())?;
    Ok(())
}

/// Reads a previously written report back, for the download endpoint.
pub fn read_report(dir: &str, name: &str) -> Result<String, String> {
    let path = Path::new(dir).join(format!("{}.csv", name));
    fs::read_to_string(&path).map_err(|e| format!("read {:?} failed: {}", path, e))
}
