package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeMaterialsRequest is the request to analyze material inventory and demand.
type AnalyzeMaterialsRequest struct {
	// Optional sales order IDs to filter by.
	SalesOrderIDs []string `json:"sales_order_ids,omitempty"`
	// Optional supplier IDs to filter by.
	SupplierIDs []string `json:"supplier_ids,omitempty"`
}

// Returns material inventory and demand analytics per material, including quantities, unit groups, and supplier information.
//
// The quantity in inventory is available to promise: available receipts less reserved and open issues, each net of its allocations. The quantity in demand is the open issues. Both are converted to the order point's unit, or the item's base unit for a material without an order point.
type AnalyzeMaterialsEndpoint struct{}

func (e *AnalyzeMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeMaterialsRequest, *apiresource.AnalyzeMaterialsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeMaterialsRequest, *apiresource.AnalyzeMaterialsResponse]{
		Title:               "Analyze Materials",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/materials",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeMaterialsRequest) (*apiresource.AnalyzeMaterialsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeMaterials
		},
	})
}
