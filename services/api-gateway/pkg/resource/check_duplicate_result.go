package apiresource

// Result of a duplicate check.
type CheckDuplicateResult struct {
	// Whether the record number is a duplicate.
	IsDuplicate bool `json:"is_duplicate" validate:"required"`
	// Human-readable message if the record is a duplicate.
	Message *string `json:"message"`
}

var exampleCheckDuplicateResult = &CheckDuplicateResult{
	IsDuplicate: true,
	Message:     new("This invoice number INV-001 already exists"),
}

func (*CheckDuplicateResult) SchemaExample() any {
	return exampleCheckDuplicateResult
}
