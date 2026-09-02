// Package ledger defines a shared tolerance, `Epsilon`, for tiny rounding differences in inventory calculations. Both the allocator and audit/repair tools use this value so they agree on what constitutes a discrepancy.
package ledger

import "github.com/shopspring/decimal"

// Epsilon is the largest difference in base units that is not a discrepancy. Every ledger comparison crosses at least one unit ratio, and those are DECIMAL(65,30) divisions — a receipt in pairs drawn by an issue in eaches does not come back to exactly zero.
var Epsilon = decimal.RequireFromString("0.000001")

// EpsilonFlagDefault is Epsilon as the string the repair commands expose as a --epsilon default.
const EpsilonFlagDefault = "0.000001"
