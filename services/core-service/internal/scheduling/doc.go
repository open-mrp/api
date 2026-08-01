// Package scheduling is the production schedule solver.
//
// It is a pure port of the knit-scheduling script
// (dashboard/apps/api/src/scripts/knit-scheduling-merz.ts): no database access, no
// clock, no randomness. Everything it needs is loaded up front into SolverInput, and
// Solve returns the plan. That boundary is what makes the parity gate against the
// script possible — the same input must produce the same plan.
//
// The pipeline, in order:
//
//	measure    per-item run rate, cost, lot size and machine affinity from history
//	changeover calibrate the yarn-driven setup time model
//	demand     pool finished-goods demand back onto the constraint item
//	policy     EOQ, safety stock, reorder point, ABC class
//	levelling  the capacity-levelled (s,S) sweep across the horizon
//	explode    push the constraint plan downstream through the process DAG
//
// # Determinism
//
// The script leans on JavaScript's insertion-ordered maps and stable sort. Go's map
// iteration is randomized, so anything that iterates a map here MUST sort first or the
// plan will differ between identical runs. TestSolve_Deterministic guards this.
package scheduling

// SolverVersion is stamped onto every generated schedule so a plan can be traced back
// to the algorithm that produced it. Bump it whenever the output changes for the same
// input.
const SolverVersion = "v1"
