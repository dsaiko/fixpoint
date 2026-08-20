# NoteSync: offline-first note synchronization

NoteSync keeps a user's notes available and editable on every device, with or
without a network, and reconciles the copies when connectivity returns. This
document describes the sync protocol, the client storage layer, the server, and
the operational posture we will ship with.

## 1. Goals

- A note edited offline is never lost.
- Sync completes within 30 seconds of connectivity returning.
- The server is stateless above its database; any instance can serve any user.
- One code path for first sync, incremental sync and recovery.
- A support engineer can answer "why does this device not have my note" from
  the dashboards alone.

## 2. Client storage

Each device keeps a single SQLite file, `notes.db`, holding the note table and
the outbox. The desktop client and the CLI tool both open this file directly;
whichever process starts first creates it. Notes carry `id`, `body`,
`modified_at` (client wall clock, milliseconds) and `deleted` flag.

Note `id` is generated on the device that creates the note, as a random 32-bit
integer. The server stores notes under its own `note_id BIGSERIAL` primary key
and keeps the client id in a secondary column for lookup.

`body` is free text. We do not cap its length on the client: a note is whatever
the user typed, and truncating a user's writing is not acceptable.

Edits made while offline are appended to the **outbox** table. The outbox is
drained oldest-first when sync runs. There is no cap on the outbox: a device
that stays offline simply accumulates rows until sync succeeds. Each outbox row
carries a full copy of the note body as it stood at the edit, so a sync can be
assembled without reading the note table.

`notes.db` is stored in the application's data directory with default file
permissions. Device-level disk encryption is the platform's job, so the file
itself is written in the clear.

## 3. The sync protocol

Sync is a single POST of the outbox batch to `/sync`, answered by the server's
changes since the client's last cursor:

1. Client sends `{cursor, edits: [...]}` — the entire outbox in one request.
2. Server applies each edit in order, writing rows as it goes.
3. Server responds `{cursor', changes: [...]}`.
4. Client applies the server changes, advances the cursor, and clears the
   outbox.

The cursor is the `modified_at` of the newest change the client has seen. A
client asks for everything strictly newer than its cursor. One cursor is stored
per user account.

If the request fails — timeout, 5xx, connection drop — the client retries the
same POST after a backoff. Retries repeat until the batch is acknowledged. The
backoff doubles from one second up to a minute, and every client uses that same
schedule.

When both sides changed the same note, the copy with the newer `modified_at`
wins; the other copy is discarded. Devices with skewed clocks are expected to
be rare, and NTP handles the common case.

The wire format is JSON with the field names above. Adding a field later is
backwards compatible, so the protocol carries no version number.

## 4. Auth

Clients hold a refresh token and rotate a short-lived access token. When a
refresh fails with 401 — revocation, password change, or token corruption —
the client signs the user out and wipes local state, including `notes.db`, so
a stolen device cannot read notes after a remote revocation. The user signs in
again on the next launch and performs a first sync.

## 5. Server

The server is a stateless HTTP service over Postgres. `/sync` runs in a single
transaction per request on the server side. Batches are applied row by row;
if row N fails validation, rows 1..N-1 stay applied and the response reports
the failing row, so a client never has to resend work the server already has.

The changes query is `SELECT * FROM notes WHERE user_id = $1 AND modified_at >
$2 ORDER BY modified_at`, returning every matching row. Postgres is fast and
the working set per user is small, so no pagination is needed and no index
beyond the primary key is planned for the first release.

Load balancing is round-robin; because the service is stateless, a retry may
land on any instance. There is no request timeout on the server: a sync is
allowed to take as long as it takes, because failing a large first sync would
be worse than holding a connection.

The server keeps only the latest version of each note. Superseded bodies are
overwritten in place, which keeps the table small.

## 6. Failure handling

The design's failure posture is: **no acknowledged edit is ever lost**. An
edit is acknowledged when the client receives the server's response for the
batch containing it.

- A crash mid-sync on the client: the outbox still holds the batch (it is
  cleared only on acknowledgment), so the next sync resends it.
- A crash mid-sync on the server: the transaction rolls back and the client
  retries.
- Prolonged offline: the outbox grows and drains when connectivity returns.
- Postgres unavailable: `/sync` returns 500 and the client retries on its
  backoff schedule. There is no degraded read-only mode; the clients hold the
  data anyway.

A conflict discards the losing copy immediately. We do not keep the discarded
body: the winning copy is what the user asked for, and keeping both would
require a merge UI we are not building.

## 7. Operations

Each instance exposes `/healthz`, which returns 200 while the process is
running. The load balancer removes instances that fail three consecutive
probes. Database connectivity, replication lag and disk headroom are dashboard
metrics; they do not gate `/healthz`, keeping the check fast and dependency-
free.

We track request rate, error rate and p99 latency per endpoint. Sync failures
show up as 5xx on that dashboard. A client that never manages to sync is a
client that stops sending requests, which is not itself an alertable signal;
we rely on support tickets for that class of problem.

Deploys are rolling. Schema migrations run before the new version starts;
old and new versions never run concurrently against the same schema because
deploys are quick. In-flight requests are not drained on shutdown: the client
retry loop covers them.

Backups are nightly `pg_dump` to object storage, retained for 14 days. Restore
has not been rehearsed; the procedure is "load the dump into a fresh instance".

## 8. Scale and capacity

The first release targets 50k users with a few hundred notes each. Sync runs
whenever the app is foregrounded and every 15 minutes on a timer while it is
running. On mobile the OS may suspend the process; when it resumes, the timer
fires and sync happens then.

When all clients come back after a regional outage, they sync at once. The
server scales horizontally and the database has headroom, so we expect the
spike to be absorbed by adding instances.

There is no per-user rate limit. A misbehaving client that syncs in a tight
loop is a support conversation, not a mechanism.

## 9. Data lifecycle

Deleting a note sets `deleted = true` and syncs the flag, so every device
learns about the deletion. Rows with `deleted = true` stay in both databases
forever; they are small, and keeping them is what makes deletion converge.

Account deletion removes the user row. The notes rows carry `user_id` and are
left in place; a nightly job could clean them up later if the table grows.

There is no audit trail of syncs. The server writes no record of which device
sent which batch, because the note rows carry everything a user can ask about.

Attachments (images, voice memos) are a later phase. They will live in object
storage with the note row holding a URL.

## 10. Out of scope

Sharing notes between users, end-to-end encryption, and search are later
phases. The protocol reserves a `shared_with` field on the note row so the
schema will not need another migration for sharing.
