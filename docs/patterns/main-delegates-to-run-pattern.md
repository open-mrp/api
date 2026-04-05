# Pattern: `main()` delegates to `Run(...)`

This repo uses a convention where the `main()` function is minimal and delegates all program behavior to a `Run(...)` function.

## Overview

`main()` should only:
- create the root context
- wire real OS dependencies (env, stdin/stdout/stderr)
- call `Run(...)`
- handle the final error/exit code

`Run(...)` is the true entry point and is responsible for:
- configuration loading + validation
- initialization of dependencies (db, messaging, tracing, clients)
- starting background loops / servers
- signal handling and graceful shutdown
- returning an error instead of calling `os.Exit`

## Example

```go
// main.go
package main

import (
  "context"
  "fmt"
  "os"
)

func main() {
  ctx := context.Background()
  if err := Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
    fmt.Fprintf(os.Stderr, "%s\n", err)
    os.Exit(1)
  }
}
```

```go
// run.go
func Run(
  ctx context.Context,
  getenv func(string) string,
  stdin io.Reader,
  stdout, stderr io.Writer,
) error {
  // ...
}
```

## Why we do this

<!-- ! NOTE: we are not doing this in integration tests -->
Separating `Run(...)` from `main()` enables integration tests for each service. It also provides several additional benefits:

### 1) Dependency injection without frameworks
Passing `getenv`, `stdin`, `stdout`, `stderr` (and potentially other deps) makes dependencies explicit and swappable in tests:
- override env reads without mutating global process env
- capture logs/output deterministically
- simulate stdin input where needed

### 2) Deterministic, testable lifecycle management
`Run(...)` returns an error instead of terminating the process. This allows tests to:
- start the service
- drive it through requests/events
- cancel the context or trigger shutdown
- assert graceful cleanup and final error behavior

### 3) Reliable shutdown semantics
By centralizing signal handling and shutdown into `Run(...)`, we ensure:
- a single place to manage `signal.NotifyContext`
- consistent server shutdown timeouts
- consistent ordering for stopping background goroutines (outbox, workers, etc.)
- fewer shutdown races and leaked goroutines

### 4) Easier fault injection and failure-path coverage
Tests can intentionally induce failures and assert behavior without spawning subprocesses:
- config validation errors
- db connection failures
- messaging broker failures
- upstream client readiness timeouts
- server listen/bind failures

## Conventions

### Function signature
Prefer a signature that passes OS dependencies explicitly:

```go
func Run(
  ctx context.Context,
  getenv func(string) string,
  stdin io.Reader,
  stdout, stderr io.Writer,
) error
```

### Error handling
- `main()` is the only place that should call `os.Exit`.

### Context and signals
- `Run(...)` is responsible for `signal.NotifyContext` and `defer stop()`.
- All long-running operations should derive from the passed `ctx`.

### Output and logging
- Logging should be configurable and write to `stdout` by default.
- Avoid writing directly to `os.Stdout`/`os.Stderr` inside `Run(...)`; use the injected writers.

## Testing guidance

Typical test flow:
1. Build env via a map-backed `getenv`.
2. Provide `bytes.Buffer` for `stdout`/`stderr`.
3. Start `Run(...)` in a goroutine.
4. Wait for readiness (health endpoint, log line, or explicit ready channel if available).
5. Exercise the service.
6. Cancel the context to trigger shutdown.
7. Assert on outputs and errors.