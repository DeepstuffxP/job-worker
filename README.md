# Job Worker

A background job processing system built in Go. Jobs are submitted via a REST API, persisted to PostgreSQL, and processed concurrently by a pool of worker goroutines. Workers claim jobs using `SELECT FOR UPDATE SKIP LOCKED` — a Postgres row-level locking mechanism that guarantees no two workers ever process the same job simultaneously. Currently supports email delivery jobs via Gmail SMTP, with graceful shutdown via context cancellation.

## How It Works

```
POST /jobs → save to Postgres (status: pending)
                    ↓
     Worker pool (5 goroutines, running continuously)
                    ↓
     Worker claims job atomically (FOR UPDATE SKIP LOCKED)
     → status: processing
                    ↓
     Worker sends email via SMTP
     → status: done / failed
                    ↓
GET /jobs/:id → returns current status
```

## Stack

- **Go** — application runtime
- **Gin** — HTTP router
- **PostgreSQL** — job persistence and status tracking
- **`database/sql`** — raw SQL for precise control over the claim query
- **`net/smtp`** — email delivery via Gmail SMTP
- **`context.Context`** — graceful shutdown and worker cancellation


## Project Structure

```
job-worker/
├── main.go      — Gin server, routes, graceful shutdown
├── worker.go    — worker pool, job processing loop
├── db.go        — Postgres connection, job queries
├── email.go     — SMTP email sending, payload parsing
├── models.go    — shared structs

```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/jobs` | Submit a new job |
| GET | `/jobs/:id` | Get job status |

### Submit a Job
```
POST /jobs
Body: {"payload": "send_email:recipient@gmail.com:Subject:Body"}
```

Response:
```json
{
  "id": 1,
  "payload": "send_email:recipient@gmail.com:Order Filled:Your order has been processed",
  "status": "pending",
  "attempts": 0,
  "created_at": "2026-07-03T14:32:00Z",
  "updated_at": "2026-07-03T14:32:00Z"
}
```

### Check Job Status
```
GET /jobs/1
```

Response:
```json
{
  "id": 1,
  "status": "done",
  "attempts": 0,
  ...
}
```

## Payload Format

```
send_email:<to>:<subject>:<body>
```

Examples:
```
send_email:user@gmail.com:Order Filled:Your order ORD-001 for RELIANCE 10 shares at Rs 2450.50 has been filled
send_email:user@gmail.com:Price Alert:NIFTY50 has crossed 24000 — your target price has been reached
send_email:user@gmail.com:Account Alert:Login detected from new device in Mumbai at 14:32 IST
```

## The Concurrency Safety Problem

Without locking, 5 workers querying `SELECT * FROM jobs WHERE status = 'pending' LIMIT 1` simultaneously would all read the same row before any of them updates it — the same job gets processed 5 times.

The fix is a single atomic query:

```sql
UPDATE jobs SET status = 'processing'
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending'
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;
```

`FOR UPDATE` locks the selected row. `SKIP LOCKED` tells each worker to skip rows already locked by another worker and grab the next available one. The `UPDATE` and `SELECT` happen atomically — there is no gap between "I found a pending job" and "I claimed it" where another worker could sneak in. Each of the 5 workers gets a guaranteed unique job.

## Graceful Shutdown

On `Ctrl+C` or `SIGTERM`:
1. OS signal is caught on a channel
2. `cancel()` is called — cancels the shared `context.Context`
3. Every worker's `select` sees `ctx.Done()` become ready
4. Workers exit cleanly — jobs mid-processing are marked `failed` so they can be retried
5. No jobs are left stuck in `processing` state permanently

## Running Locally

**Prerequisites:** Docker, Go 1.22+, Gmail account with App Password enabled.

**1. Start Postgres:**
```bash
docker run -d --name jobworker-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=jobworker \
  -p 5432:5432 postgres:15
```

**2. Set environment variables:**
```bash
# PowerShell
$env:SMTP_EMAIL="youremail@gmail.com"
$env:SMTP_PASSWORD="your16charapppassword"
$env:DATABASE_URL="postgres://postgres:password@localhost:5432/jobworker?sslmode=disable"
```

**3. Run:**
```bash
go run .
```

**4. Submit a job:**
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"payload": "send_email:recipient@gmail.com:Test:Hello from job worker"}'
```

**5. Check status:**
```bash
curl http://localhost:8080/jobs/1
```

## Getting a Gmail App Password

1. Go to myaccount.google.com → Security
2. Enable 2-Step Verification
3. Search "App passwords" → create one named "job-worker"
4. Copy the 16-character password — use it as `SMTP_PASSWORD`
