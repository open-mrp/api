package apiresource

// CheckDuplicateResult represents the result of a duplicate check action.
type CheckDuplicateResult struct {
	// Whether the record number is a duplicate.
	IsDuplicate bool `json:"is_duplicate" validate:"required"`
	// A human-readable message if the record is a duplicate.
	Message *string `json:"message"`
}

var exampleCheckDuplicateResult = &CheckDuplicateResult{
	IsDuplicate: true,
	Message:     ptrString("This invoice number INV-001 already exists"),
}

func (*CheckDuplicateResult) SchemaExample() any {
	return exampleCheckDuplicateResult
}

func ptrString(s string) *string {
	return &s
}
