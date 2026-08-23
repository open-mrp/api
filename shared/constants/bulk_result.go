package constants

// BulkResultStatus is the outcome of one row in a bulk create response.
type BulkResultStatus string

const (
	// BulkResultStatusCreated means the row was written.
	BulkResultStatusCreated BulkResultStatus = "created"
	// BulkResultStatusFailed means the row was rejected; the result carries the reason.
	BulkResultStatusFailed BulkResultStatus = "failed"
)

func (s BulkResultStatus) IsValid() bool {
	switch s {
	case BulkResultStatusCreated, BulkResultStatusFailed:
		return true
	default:
		return false
	}
}

func (s BulkResultStatus) EnumValues() []string {
	return []string{
		string(BulkResultStatusCreated),
		string(BulkResultStatusFailed),
	}
}

// BulkResultAction is what a bulk create did to the record behind one row, which may be an update
// rather than an insert when the row matched something that already existed.
type BulkResultAction string

const (
	// BulkResultActionCreated means a new record was inserted.
	BulkResultActionCreated BulkResultAction = "created"
	// BulkResultActionUpdated means an existing record was updated in place.
	BulkResultActionUpdated BulkResultAction = "updated"
	// BulkResultActionSkipped means nothing was written for the row.
	BulkResultActionSkipped BulkResultAction = "skipped"
)

func (a BulkResultAction) IsValid() bool {
	switch a {
	case BulkResultActionCreated, BulkResultActionUpdated, BulkResultActionSkipped:
		return true
	default:
		return false
	}
}

func (a BulkResultAction) EnumValues() []string {
	return []string{
		string(BulkResultActionCreated),
		string(BulkResultActionUpdated),
		string(BulkResultActionSkipped),
	}
}
