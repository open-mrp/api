package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// The outcome of checking whether a record number is already in use.
type CheckDuplicateResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=check_duplicate_result"`
	// Whether a record with the submitted number already exists.
	//
	// Invoice and sales order numbers are matched across the whole account; a customer PO number is matched only against the orders of the customer given in the request, so the same PO number may exist on another customer's orders without being reported here.
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
