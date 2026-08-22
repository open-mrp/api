// ! Note: this will be refactored in the future - okay to leave as is.
package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
)

// BulkCreateItemResult represents the result of creating a single item in a bulk operation.
type BulkCreateItemResult struct {
	// The SKU of the item.
	SKU string `json:"sku" validate:"required"`
	// Outcome of the create attempt: "created" or "failed".
	Status string `json:"status" validate:"required"`
	// The error message if the item failed to create.
	Error *string `json:"error"`
	// The ID of the created item.
	ItemID *string `json:"item_id"`
}

// BulkCreateItemsResponse represents the response from the bulk create items endpoint.
type BulkCreateItemsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The results of each item creation.
	Data []BulkCreateItemResult `json:"data" validate:"required"`
}

func (*BulkCreateItemsResponse) SchemaExample() any {
	return map[string]any{
		"object": "list",
		"data":   []any{},
	}
}

// BulkCreateProductionStepResult represents the result of creating a single production step.
type BulkCreateProductionStepResult struct {
	// The name of the production step.
	Name string `json:"name" validate:"required"`
	// Outcome of the operation for this step: "created" or "failed".
	Status string `json:"status" validate:"required"`
	// The error message if the step failed.
	Error *string `json:"error"`
	// The ID of the created or updated production step.
	ProductionStepID *string `json:"production_step_id"`
	// The action taken: "created", "updated", or "skipped".
	Action string `json:"action" validate:"required"`
}

// BulkCreateProductionStepsResponse represents the response from the bulk create production steps endpoint.
type BulkCreateProductionStepsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The results of each production step creation.
	Data []BulkCreateProductionStepResult `json:"data" validate:"required"`
}

func (*BulkCreateProductionStepsResponse) SchemaExample() any {
	return map[string]any{
		"object": "list",
		"data":   []any{},
	}
}
