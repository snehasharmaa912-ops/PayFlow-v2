# PayFlow v2

Money can never just appear or disappear. That's the one rule this whole project is built around.

Most "payment system" projects you see on GitHub are basically a charges table with a REST wrapper around it. This one started that way too, and then I found a real bug in it — a race condition that let the same idempotency key create two charges instead of one — and everything after that was built to make sure that kind of thing gets caught instead of getting lucky.

**Live:** [https://payflow-api-ulqi.onrender.com/healthz](https://payflow-api-ulqi.onrender.com/healthz)
(free-tier hosting, so give it ~50 seconds to wake up if it's been idle)

## Why three languages

I didn't pick Go, Scala, and Ruby to show off. Each one is doing the job it's actually good at:

- **Go** runs the core service. Payments code lives or dies by how it behaves under concurrency, and Go's concurrency model plus its standard library made that easy to get right (eventually — see below).
- **Scala** runs the risk engine. Risk rules are the kind of logic that grows more `if` statements every quarter as new edge cases show up, and pattern matching over case classes keeps that from turning into spaghetti.
- **Ruby** runs the CLI. Nobody wants to read raw JSON output when they're auditing charges by hand. Ruby made that part fast to write and pleasant to read.

They only talk to each other over well-defined boundaries — HTTP between Ruby and Go, JSON over stdin/stdout between Go and Scala. No shared memory, no tight coupling.

## The bug that actually matters here

Early on, `CreateCharge` checked whether an idempotency key had been seen before under a read lock, released that lock, and then wrote the new charge under a separate write lock. That gap between checking and writing is a classic TOCTOU race — two requests can both check at the same time, both see "not seen yet," and both go ahead and create a charge.

Every test passed. Every single one. Because every test I'd written up to that point ran one request at a time, and a single-threaded test can't prove anything about what happens when two requests land in the same microsecond.

So I wrote a test that fires 50 identical requests — same idempotency key — genuinely concurrently, using goroutines and a WaitGroup. First run, it failed immediately. Different callers got back different charge IDs for what should have been one logical charge.

The fix was to stop treating "check" and "write" as two separate steps and put them both inside one lock, with the event log write happening in that same critical section too — so the log and the in-memory state can never disagree about what actually happened first.

This is the part of the project I'd actually want to talk about in an interview. Not "I built an API." More like: I had to decide where a lock should scope to, got it wrong the first time, wrote a test that caught it, and know exactly why the fix works.

## How correctness is actually enforced, not just hoped for

**Write-ahead logging.** Every charge gets written to a durable log file before anything in memory changes, and that write is fsync'd before the request returns. If the process dies mid-request, the log already has the decision recorded. There's a `ReplayFromLog` function that throws away the live store completely and rebuilds it from nothing but that log file — and a `/debug/verify-replay` endpoint that checks live state against a fresh replay on demand, instead of just trusting it silently.

**Double-entry bookkeeping.** Every charge posts two ledger entries — a debit from the customer and an equal credit to platform revenue — instead of one flat balance. A flat balance can silently drift if a bug double-counts or drops something and nothing looks wrong until it's too late. With double-entry, there's a hard invariant to check: everything in the ledger has to sum to zero, always. Declined charges are the one exception — they never post at all, since no money actually moved.

**Fail-safe risk checks.** For charges over a threshold, Go shells out to the Scala risk engine. If that engine is unreachable, slow, or errors out, the charge doesn't silently succeed and it doesn't crash the request — it falls back to `pending_review`.

## What got added after the core was solid

Once the ledger and concurrency story were done, I added the parts that make this feel closer to something you'd actually run in production, not just a demo:

- **Idempotency-Key header** — send it as a proper HTTP header (`Idempotency-Key: ...`) the way Stripe's own API does, and a retried request with the same key just returns the original charge instead of creating a new one. Still falls back to the old body field if you don't send the header.
- **API key auth** — bearer token auth on every route except `/healthz`.
- **Rate limiting** — a token bucket per caller, so one client can't hammer the API.
- **Signed webhooks** — the code supports firing `charge.succeeded` / `charge.failed` events at a configured URL, signed with HMAC-SHA256 in a `Payflow-Signature` header, with retries and exponential backoff on failure. Not enabled on the live deployment yet — see the note in Deploying below.

## Endpoints

| Method | Path | What it does |
|---|---|---|
| POST | `/charges` | Create a charge (send `Idempotency-Key` header) |
| GET | `/charges` | List every charge |
| GET | `/charges/{id}` | Get one charge |
| GET | `/ledger` | Ledger balances + whether it's balanced |
| GET | `/debug/verify-replay` | Compare live state against a fresh log replay |
| GET | `/healthz` | Health check, no auth required |

## Running it yourself

cd go
go build -o payflow ./cmd/server
PORT=8080 ./payflow

Environment variables, all optional — the app runs fine with none of these set, each one just falls back to a safe default:

| Variable | What it's for | If it's not set | Set on live deployment? |
|---|---|---|---|
| `PAYFLOW_LOG_PATH` | Where the write-ahead log lives | defaults to `payflow.log` | Yes |
| `PAYFLOW_RISK_JAR` | Path to the Scala risk-engine jar (needs a JRE on the machine) | risk engine is skipped | Set, but not yet functional — see Deploying below |
| `PAYFLOW_API_KEYS` | Comma-separated valid keys | auth stays disabled, API stays open | Yes |
| `PAYFLOW_RATE_LIMIT_RPS` | Requests per second per caller | defaults to 5 | Yes |
| `PAYFLOW_WEBHOOK_URL` | Where to send charge events | webhooks never fire | Not yet |
| `PAYFLOW_WEBHOOK_SECRET` | Secret used to sign the webhook payloads | irrelevant without a webhook URL | Not yet |

## Deploying

Runs on Render off `render.yaml` and a `Dockerfile` — only the Go service gets deployed; Ruby and Scala are tools, not services.

Things worth knowing about the current live deployment:

- The event log sits on Render's ephemeral disk by default, so charge history doesn't survive a restart or redeploy unless a persistent Disk is attached and `PAYFLOW_LOG_PATH` points into it.
- `PAYFLOW_RISK_JAR` is set as an environment variable, but the Docker build doesn't yet compile the Scala jar (`sbt assembly`) or include a JRE in the runtime image, so the path it points to doesn't actually exist in the container. This is a known, intentional gap, not a bug — the app fails safe: large charges just fall back to `pending_review` instead of crashing or silently skipping the check. Wiring it up for real means updating the `Dockerfile` to build the jar and add a JRE.
- Webhooks aren't enabled live because there's nothing set up yet to receive them. Setting `PAYFLOW_WEBHOOK_URL` without a working receiver just means every charge retries delivery a few times and gives up quietly.

## Testing

cd go
go test ./... -race

The -race flag isn't optional here. It's literally what would have caught the concurrency bug before the load test did.

