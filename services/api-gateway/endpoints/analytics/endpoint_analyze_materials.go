package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeMaterialsRequest is the request to analyze material inventory and demand.
type AnalyzeMaterialsRequest struct {
	// Optional sales order IDs to filter by.
	SalesOrderIDs []string `json:"sales_order_ids,omitempty"`
	// Optional supplier IDs to filter by.
	SupplierIDs []string `json:"supplier_ids,omitempty"`
}

// Returns material inventory and demand analytics per material, including quantities, unit groups, and supplier information.
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
