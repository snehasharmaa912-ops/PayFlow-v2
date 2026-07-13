# PayFlow v2 — Design Notes

## The problem this project actually solves

A payments system has one non-negotiable job: money can never appear or
disappear. A charges list that "looks right when you test it by hand
once" doesn't prove that. This project is built around mechanisms that
catch you when something goes wrong — not code that merely looks correct.

## Why three languages

- **Go** for the core service, because correctness under concurrency and
  failure needs real threads, an explicit locking model, and a standard
  library built for networked services.
- **Scala** for the risk rules, because pattern matching over immutable
  case classes makes business logic readable and hard to corrupt by
  accident when new rules get added later.
- **Ruby** for the CLI, because a human-facing audit tool should be fast
  to write and read almost like English.

They only ever talk through well-defined interfaces — HTTP between Ruby
and Go, JSON over stdin/stdout between Go and Scala. Nothing shares
memory or is tightly coupled across the language boundary.

## The bug — and why it's the most important part of this project

Step 1 shipped a `CreateCharge` that checked the idempotency key under an
`RLock`, released it, then wrote under a separate `Lock`. That gap between
check and write is a textbook TOCTOU (time-of-check-to-time-of-use) race:
two goroutines can both observe "key not seen yet" before either one
finishes writing.

It passed every test through Step 3, because every test up to that point
was single-threaded. A single-threaded test proves nothing about
correctness under concurrency — you can write a charge, read it back,
watch it look perfect, and still be carrying a bug that only shows up
when two requests land in the same microsecond.

Step 4 built a test that fires 50 identical requests, same idempotency
key, genuinely concurrent via goroutines and a `WaitGroup`. First run: it
failed. More than one charge got created for the same key, and different
callers got back different charge IDs for what should have been one
logical charge.

The fix was to collapse the check and the write into a single critical
section — one `Lock()` held across both the idempotency lookup and the
map writes, with the event log append happening inside that same section
so the log and the in-memory state can never disagree about what
happened first.

This is the single best story in the whole project: not "I built an
API," but "I had to decide where the idempotency lock should scope to, I
got it wrong the first time, my load test caught it, and here's exactly
what was happening and how I fixed it." That's what separates
understanding failure modes from just writing happy-path code.

## Write-ahead logging

Every charge-creation event is written to a durable log file *before* the
in-memory ledger and charge maps are mutated, and the write is fsync'd
before the request is allowed to return. If the process crashes between
"decide to create a charge" and "update in-memory state," the log already
has a durable record of the decision. `ReplayFromLog` proves this
actually works: it throws the live in-memory store away entirely and
rebuilds it from nothing but the log file. If a replayed ledger doesn't
match what was live, the log isn't capturing everything it needs to —
and the `GET /debug/verify-replay` endpoint checks this on demand rather
than trusting it silently.

## Double-entry bookkeeping

Every charge posts two ledger entries, not one: a debit against the
customer's account and an equal credit to platform revenue. A single flat
balance can silently drift if a bug double-counts or drops a charge and
nothing will look wrong. Double-entry gives a mechanical, checkable
invariant instead: the sum of every entry across the whole ledger must
always equal zero. `Ledger.Verify()` checks exactly that, and it's called
from both the `/ledger` endpoint and the replay-comparison endpoint.

One deliberate exception: a **declined** charge does not post to the
ledger at all, because a declined payment never actually moved money.
Posting it and then reversing it would be more complex and more failure-
prone than simply not posting it in the first place.

## The risk engine as a real polyglot boundary

For charges over `LargeChargeThreshold`, Go shells out to a Scala jar via
`os/exec`, passing the charge as one line of JSON on stdin and reading a
decision back off stdout. The Go side depends on a `RiskEvaluator`
interface, not directly on `os/exec` — so the store's decision-handling
logic (approve → succeeds and posts to the ledger; review → succeeds but
flagged; decline → doesn't post) is fully unit-testable with a fake
evaluator, and the real subprocess path gets its own separate, opt-in
integration test that only runs when a built jar is provided. If the risk
engine is unreachable or errors, the charge fails safe to `pending_review`
rather than either silently succeeding or crashing the request.

## What "top of the resume pile" actually means here

It's not line count. It's being able to say, unprompted, in an interview:
*"I had to decide where an idempotency lock should scope to. I got it
wrong the first time. My concurrency test caught it — here's what was
actually happening and how I fixed it."* That's a real engineering story
about a failure mode, not a description of an API surface.
