# NoteSync: offline-first note synchronization

NoteSync keeps a user's notes available and editable on every device, with or
without a network, and reconciles the copies when connectivity returns. This
document describes the sync protocol, the client storage layer, and the server.

## 1. Goals

- A note edited offline is never lost.
- Sync completes within 30 seconds of connectivity returning.
- The server is stateless above its database; any instance can serve any user.
- One code path for first sync, incremental sync and recovery.

## 2. Client storage

Each device keeps a single SQLite file, `notes.db`, holding the note table and
the outbox. The desktop client and the CLI tool both open this file directly;
whichever process starts first creates it. Notes carry `id`, `body`,
`modified_at` (client wall clock, milliseconds) and `deleted` flag.

Edits made while offline are appended to the **outbox** table. The outbox is
drained oldest-first when sync runs. There is no cap on the outbox: a device
that stays offline simply accumulates rows until sync succeeds.

## 3. The sync protocol

Sync is a single POST of the outbox batch to `/sync`, answered by the server's
changes since the client's last cursor:

1. Client sends `{cursor, edits: [...]}` — the entire outbox in one request.
2. Server applies each edit in order, writing rows as it goes.
3. Server responds `{cursor', changes: [...]}`.
4. Client applies the server changes, advances the cursor, and clears the
   outbox.

If the request fails — timeout, 5xx, connection drop — the client retries the
same POST after a backoff. Retries repeat until the batch is acknowledged.

When both sides changed the same note, the copy with the newer `modified_at`
wins; the other copy is discarded. Devices with skewed clocks are expected to
be rare, and NTP handles the common case.

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

Load balancing is round-robin; because the service is stateless, a retry may
land on any instance.

## 6. Failure handling

The design's failure posture is: **no acknowledged edit is ever lost**. An
edit is acknowledged when the client receives the server's response for the
batch containing it.

- A crash mid-sync on the client: the outbox still holds the batch (it is
  cleared only on acknowledgment), so the next sync resends it.
- A crash mid-sync on the server: the transaction rolls back and the client
  retries.
- Prolonged offline: the outbox grows and drains when connectivity returns.

## 7. Operations

Each instance exposes `/healthz`, which returns 200 while the process is
running. The load balancer removes instances that fail three consecutive
probes. Database connectivity, replication lag and disk headroom are dashboard
metrics; they do not gate `/healthz`, keeping the check fast and dependency-
free.

Deploys are rolling. Schema migrations run before the new version starts;
old and new versions never run concurrently against the same schema because
deploys are quick.

## 8. Out of scope

Sharing notes between users, end-to-end encryption, and search are later
phases. The protocol reserves a `shared_with` field on the note row so the
schema will not need another migration for sharing.
