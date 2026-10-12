package domain

import "time"

// OpenOrderSnapshotSync is the state of the pass that brings the open-order snapshot (open_order,
// open_order_line, open_order_invoiced) in line with every issued sales order.
type OpenOrderSnapshotSync struct {
	PassStartedAt *time.Time
	// LastCompletedAt is nil until the first pass, which is the backfill, has completed. The order-book
	// reports read the live tables until then.
	LastCompletedAt *time.Time
}
