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

## 4. Comments may organize a long file

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

## Auditing a diff's comments (audit mode)

Reviewing every comment in a large diff is a fan-out job, not a read-through: extract each
added comment, verify its claim against the code it annotates, and report. Run it like this.

1. **Extract** added/changed comment lines, grouped by file:

   ```bash
   git diff <base> -- '*.go' '*.sql' '*.proto' | awk '
     /^diff --git/ {f=$3; sub("a/","",f)}
     /^\+[[:space:]]*(\/\/|--)/ {print f}' | sort | uniq -c | sort -rn
   ```

   Skip generated files (`*.pb.go`, `sqlc/*.sql.go`, `*_mock.go`) — their comments are copied
   from the `.proto`/`.sql`/source, so audit the source and the copy is covered.

2. **Fan out** one subagent per area (services, SQL, tests, loaders/domain). Each reads the
   annotated code *fully* and returns a verdict per comment. Auditing is read-only — agents
   report, they do not edit.

3. **Verdict per comment**, then the style flags:
   - TRUE — matches the code.
   - FALSE — contradicts the code (wrong column, wrong return, wrong condition).
   - MISLEADING — defensible but reads wrong. **This is the dominant failure, not FALSE.**
   - STALE — describes code that no longer exists as described.
   - Style: change-narration (§1), how-not-why (§2), too-long (§3), name-prefix.

**What MISLEADING looks like in practice** (every one of these was a real finding):

- **A test comment that over-claims the assertion.** "a replay must not report zero" on an
  assertion that is trivially `0 == 0`; "the header links to order, pick and invoice" above a
  block that only checks order and pick. The comment states the *goal* — verify it against what
  the code actually asserts, not against what it was trying to prove.
- **A count or list a later hunk invalidates.** "Two dedicated shipments" when the same change
  adds a third under the same section.
- **A parenthetical the next line contradicts.** "(not a loader Target)" directly above
  `Target: ...`.
- **"Never reached" / "short-circuits" that overstate.** The path is unwired entirely, not
  merely skipped in one mode.

**"Legacy" is usually a why, not change-narration.** A comment citing the external system being
migrated from ("matching legacy's canCreateInvoice") states a standing reason and is fine. A
comment narrating *this file's own* old→new format ("Legacy returned a bare number; v2 wraps it")
is §1 change-narration. Test: does "legacy" name another system, or a prior version of this code?

## The check before you keep a comment

1. Would this read as sensible to someone seeing the file for the first time, with no
   knowledge of what it looked like before? If no → delete or restate as a standing fact.
2. Does it say something the code cannot? If no → delete.
3. Is it explaining *how*? If yes → fix the code instead.
4. Can it lose half its words without losing meaning? If yes → do that.

Deleting a comment is a normal outcome. A wrong or stale comment is worse than none, so
if the code moves, the comment moves with it in the same change.
