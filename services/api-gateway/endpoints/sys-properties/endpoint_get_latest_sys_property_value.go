package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get the latest value for a system property type.
type GetLatestSysPropertyValueRequest struct {
	// Code identifying which number series to read.
	//
	// - `transaction_number`: numbering for financial transactions such as payments, credit memos, adjustments, and rebates.
	// - `settlement_number`: numbering for settlements that apply transactions to invoices.
	// - `sales_order_number`: numbering for sales orders.
	// - `purchase_order_number`: numbering for purchase orders.
	// - `customer_number`: identifiers assigned to new customers.
	// - `supplier_number`: identifiers assigned to new suppliers.
	// - `production_run_number`: numbering for production runs.
	// - `sscc_count`: serial component of the GS1 SSCC-18 codes assigned to shipping cases.
	TypeCode string `path:"type_code" validate:"required"`
}

// Returns the next available counter value for a system property type.
//
// Initializes the counter at `1` if it does not yet exist for the account. If the current value is already used by an existing record (for example, a transaction with that number), the counter is incremented before the value is returned. The `sscc_count` counter is returned as-is, without a duplicate check.
type GetLatestSysPropertyValueEndpoint struct{}

func (e *GetLatestSysPropertyValueEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLatestSysPropertyValueRequest, *apiresource.SysPropertyValue] {
	return (&apiendpoint.APIEndpoint[*GetLatestSysPropertyValueRequest, *apiresource.SysPropertyValue]{
		Title:             "Get Latest System Property Value",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/properties/{type_code}/latest-value",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSystemProperties, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetLatestSysPropertyValueRequest) (*apiresource.SysPropertyValue, *apierror.APIError) {
			return svc.(SysPropertySvc).GetLatestSysPropertyValue
		},
	})
}
