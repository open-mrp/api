---
name: code-comments
description: How to write comments in this repo — why over how, no change narration, ruthless brevity. Use whenever writing or editing a comment in Go, SQL, or proto, and when reviewing a diff for comment quality.
---

# Comments

Comments exist for **human readability** — nothing else. They are not a changelog, not a
transcript of the code, and not a place to compensate for code that does not read well.

`docs/patterns/comment-conventions.md` is the repo's convention and still governs
(business intent, side effects, ordering, idempotency, failure modes; `(required)` /
`(optional; default: …)` on config fields; actionable TODOs). This skill covers what goes
wrong in practice.

## 1. A comment must be ignorant of every prior version of the file

The reader has never seen the diff, the old code, or the discussion. Anything that only
makes sense as an annotation on a *change* is noise in a month.

Banned outright: "was briefly…", "this replaced…", "previously…", "they drifted once…",
"now traced rather than dropped", "deliberately NOT one of them any more", "this used to".

The only exception: the code breaks a convention so plainly that a reader will try to
"fix" it back. Then state the constraint as a standing fact, not as history.

```go
// Bad — the reader is being told a story about a bug that is already gone.
// The update's guard and Job.IsTerminal have to name the same states, and nothing in
// the type system couples them. They drifted once — the guard also required
// failed_at IS NULL, which froze a failed job against every later transition.

// Good — the standing reason, stated once.
// Nothing couples the SQL guard to Job.IsTerminal, so pin them here: a failed job stays retryable.
```

## 2. Why, not how — and needing to explain *how* means the code is wrong

The code says what it does. Comment the reason it exists, the side effect, the
constraint that is not visible locally. Never narrate the next line.

**If you find yourself explaining how the code works, stop and fix the code.** Extract
the block into a named function, name the variable properly, drop the cleverness. The
comment you were about to write is a bug report against the code, and reaching for it
buries the problem instead.

The one allowance: a **top-of-function** line orienting a reader to a genuinely long
body. If that line cannot be short, the function is doing too much — split it.

```go
// Bad — narrates mechanics the code already states, at four times the length.
// Deliberately not returned: the rows are committed and the job is completed, so there
// is nothing left to roll back or re-drive. Traced rather than dropped — a completed job
// whose derived side effects quietly did not happen is invisible otherwise. The job this
// ran for is on the consumer's parent span, so the trace names it without repeating it.

// Good — the one fact the code cannot show.
// The job is already completed, so this has nothing to roll back; the span is where it surfaces.
```

## 3. Be short — shorter than feels sufficient

Measured across `services/`: **22,809** comment lines are ≤100 characters; **895** are
longer. Short is the house style, not an aspiration.

Budgets, and treat them as hard:

| kind | budget |
|---|---|
| inline comment | **1 line.** 2 only if the second earns its place |
| doc comment on a func/type | **1–2 lines.** 3+ means the thing is doing too much, or you are explaining a change |
| config field | 1 line, leading with `(required)` / `(optional; default: …)` |

Never hard-wrap a paragraph across `//` lines (conventions doc, rule 1) — but a comment
long enough for that question to arise is usually one that should be cut instead. Wrapped
prose in a diff is the signal that rule 3 was broken before rule 1 was.

## 4. Comments may organise a long file

Navigation is a readability job, so section markers earn their place where a file has
enough in it to get lost in. The forms already in use:

```go
// ---------------------------------------------------------------------------
// Department — full department resource
// ---------------------------------------------------------------------------
```

```go
// --- Accept phase ---
// --- idempotency helpers ---
```

A marker names a region; it does not explain it. If a section needs a paragraph to
justify its existence, it probably wants its own file.

## The check before you keep a comment

1. Would this read as sensible to someone seeing the file for the first time, with no
   knowledge of what it looked like before? If no → delete or restate as a standing fact.
2. Does it say something the code cannot? If no → delete.
3. Is it explaining *how*? If yes → fix the code instead.
4. Can it lose half its words without losing meaning? If yes → do that.

Deleting a comment is a normal outcome. A wrong or stale comment is worse than none, so
if the code moves, the comment moves with it in the same change.
