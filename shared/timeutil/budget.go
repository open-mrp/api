package timeutil

import (
	"context"
	"time"
)

// FanOutReserve is the headroom [BudgetedContext] leaves between the end of a fan-out and the caller's own deadline: enough to assemble and serialize a response, and to survive the gap between when a goroutine observes cancellation and when the handler actually returns.
const FanOutReserve = 1500 * time.Millisecond

// BudgetedContext derives a context that expires far enough before ctx's own deadline for the caller to still return a response.
//
// It exists for fan-outs over slow dependencies — a live carrier-rating round trip, a per-type search RPC — where an individual participant failing is already handled by dropping its contribution. Without a budget those fan-outs inherit the request deadline exactly, so one stalled participant spends the whole of it and the caller returns a 504 with nothing in it, discarding every result that did arrive. Cutting the fan-out short instead turns a total failure into a partial one.
//
// reserve is the headroom to leave; pass [FanOutReserve] unless the assembly step after the fan-out is unusually expensive. When ctx carries no deadline, or when less than reserve remains (the request is already over budget and about to be abandoned), ctx is returned with a no-op cancel and the caller runs unbounded — a fan-out that cannot be shortened usefully should not be truncated to nothing.
func BudgetedContext(ctx context.Context, reserve time.Duration) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ctx, func() {}
	}

	budget := time.Until(deadline) - reserve
	if budget <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, budget)
}
