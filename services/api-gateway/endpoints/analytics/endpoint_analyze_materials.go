package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeMaterialsRequest is the request to analyze material inventory and demand.
type AnalyzeMaterialsRequest struct {
	// Optional sales order IDs to filter by.
	SalesOrderIDs []string `json:"sales_order_ids,omitempty"`
	// Optional supplier IDs to filter by.
	SupplierIDs []string `json:"supplier_ids,omitempty"`
}

type AnalyzeMaterialsEndpoint struct{}

func (e *AnalyzeMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeMaterialsRequest, *apiresource.AnalyzeMaterialsResponse] {
	return &apiendpoint.APIEndpoint[*AnalyzeMaterialsRequest, *apiresource.AnalyzeMaterialsResponse]{
		Title:             "Analyze Materials",
		Description:       "Returns material inventory and demand analytics per material, including quantities, unit groups, and supplier information.",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/materials",
		ContentType:       "application/json",
		Request:           &AnalyzeMaterialsRequest{},
		Response:          &apiresource.AnalyzeMaterialsResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeMaterialsRequest) (*apiresource.AnalyzeMaterialsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeMaterials
		},
	}
}
