---
name: main-delegates-to-run
description: >-
  main() is a thin wrapper that calls Run(ctx, getenv, stdin, stdout, stderr). Use when
  adding or changing a cmd/main.go, cmd/run.go, service entry point, or service
  integration test that starts the process.
---

# `main()` delegates to `Run(...)`

`main()` creates the root context, wires OS deps, calls `Run`, and is the only `os.Exit`. `Run` loads config, initializes deps, starts servers/loops, handles signals, and **returns an error**. Human spec: `docs/patterns/main-delegates-to-run-pattern.md`.

```go
func main() {
    ctx := context.Background()
    if err := Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
}

func Run(
    ctx context.Context,
    getenv func(string) string,
    stdin io.Reader,
    stdout, stderr io.Writer,
) error
```

- `Run` owns `signal.NotifyContext` and `defer stop()`. Long-running work derives from `ctx`.
- Log to the injected writers, not `os.Stdout` / `os.Stderr`.
- Tests: map-backed `getenv`, `bytes.Buffer` for output, start `Run` in a goroutine, cancel `ctx` to shut down. Do not spawn subprocesses for the happy path.
