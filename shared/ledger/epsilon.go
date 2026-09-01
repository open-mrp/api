// Package ledger holds the tolerances the inventory ledger's arithmetic is compared against.
//
// It exists so that the allocator and the tools that audit it cannot disagree. A guard that permits
// a draw the monitor later alarms on produces an alert nobody can act on, and a guard stricter than
// the monitor fails real work for a difference nobody considers a breach.
package ledger

import "github.com/shopspring/decimal"

// Epsilon is the largest difference in base units that is not a discrepancy.
//
// Every ledger comparison crosses at least one unit ratio, and those are DECIMAL(65,30) divisions —
// a receipt in pairs drawn by an issue in eaches does not come back to exactly zero. This is the
// value cmd/inventory-invariant-check and cmd/repair-overallocated-receipts have always defaulted
// to; it is a constant here so the allocator uses the same one rather than a second copy.
var Epsilon = decimal.RequireFromString("0.000001")

// EpsilonFlagDefault is Epsilon as the string the repair commands expose as a --epsilon default.
const EpsilonFlagDefault = "0.000001"
