package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Result of a duplicate check.
type CheckDuplicateResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=check_duplicate_result"`
	// Whether a record with the given number already exists on the account.
	IsDuplicate bool `json:"is_duplicate" validate:"required"`
	// Human-readable message describing the duplicate.
	//
	// Populated only when `is_duplicate` is `true`; names the type and value that already exists.
	Message *string `json:"message"`
}

var SampleCheckDuplicateResult = &CheckDuplicateResult{
	Object:      constants.ObjectTypeCheckDuplicateResult,
	IsDuplicate: true,
	Message:     new("This invoice number INV-001 already exists"),
}

func (*CheckDuplicateResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCheckDuplicateResult)
}
