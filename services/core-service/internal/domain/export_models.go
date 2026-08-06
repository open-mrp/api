package domain

// bounds an export so one oversized account cannot exhaust the worker's memory.
// Queries fetch one row beyond it, so an overflow is detected rather than silently truncated.
const ExportRowLimit = 50_000

// carries a rendered export (name is derived from the job)
type Export struct {
	ContentType string
	Body        []byte
	RowCount    int32
}
