package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CheckDuplicateRequest is the request to check for a duplicate record number.
type CheckDuplicateRequest struct {
	// The type of duplicate check to perform: invoice_number, order_number, or customer_po_number.
	Type string `json:"type" validate:"required"`
	// The record number to check.
	RecordNumber string `json:"record_number" validate:"required"`
	// The customer ID, required for customer_po_number checks.
	CustomerID *string `json:"customer_id"`
}

var exampleCheckDuplicateRequest = &CheckDuplicateRequest{
	Type:         "invoice_number",
	RecordNumber: "INV-001",
}

func (*CheckDuplicateRequest) SchemaExample() any {
	return exampleCheckDuplicateRequest
}

type CheckDuplicateEndpoint struct{}

func (e *CheckDuplicateEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckDuplicateRequest, *apiresource.CheckDuplicateResult] {
	return &apiendpoint.APIEndpoint[*CheckDuplicateRequest, *apiresource.CheckDuplicateResult]{
		Title:             "Check Duplicate",
		Description:       "Checks whether a record number already exists for the given type (invoice number, order number, or customer PO number).",
		Method:            http.MethodPut,
		Route:             "/v1/core/actions/check-duplicates",
		ContentType:       "application/json",
		Request:           &CheckDuplicateRequest{},
		Response:          &apiresource.CheckDuplicateResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckDuplicateRequest) (*apiresource.CheckDuplicateResult, *apierror.APIError) {
			return svc.(UtilsSvc).CheckDuplicate
		},
	}
}
