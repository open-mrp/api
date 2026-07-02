package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to check for a duplicate record number.
type CheckDuplicateRequest struct {
	// The kind of record number to check.
	//
	// - `invoice_number`: checks invoice numbers.
	// - `order_number`: checks sales order numbers.
	// - `customer_po_number`: checks customer PO numbers on sales orders; requires `customer_id`.
	Type string `json:"type" validate:"required"`
	// The record number to check for an existing match.
	RecordNumber string `json:"record_number" validate:"required"`
	// ID of the customer to scope the check to.
	//
	// Required when `type` is `customer_po_number`; ignored for other types.
	CustomerID field.Optional[string] `json:"customer_id,omitzero"`
}

var sampleCheckDuplicateRequest = &CheckDuplicateRequest{
	Type:         "invoice_number",
	RecordNumber: "INV-001",
}

func (*CheckDuplicateRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckDuplicateRequest)
}

// Checks whether a record number already exists on the account for the given type (invoice number, sales order number, or customer PO number).
type CheckDuplicateEndpoint struct{}

func (e *CheckDuplicateEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckDuplicateRequest, *apiresource.CheckDuplicateResult] {
	return (&apiendpoint.APIEndpoint[*CheckDuplicateRequest, *apiresource.CheckDuplicateResult]{
		Title:             "Check Duplicate",
		Method:            http.MethodPut,
		Route:             "/v1/core/actions/check-duplicates",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckDuplicateRequest) (*apiresource.CheckDuplicateResult, *apierror.APIError) {
			return svc.(UtilsSvc).CheckDuplicate
		},
	})
}
