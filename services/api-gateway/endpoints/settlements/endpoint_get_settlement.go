package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetSettlementRequest is the request to get a settlement.
type GetSettlementRequest struct {
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
	// Sub-resources to include in the response.
	Includes []string `include:"true"`
}

type GetSettlementEndpoint struct{}

func (e *GetSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSettlementRequest, *apiresource.Settlement] {
	return &apiendpoint.APIEndpoint[*GetSettlementRequest, *apiresource.Settlement]{
		Title:             "Get Settlement",
		Description:       "Returns a settlement by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements/{id}",
		Request:           &GetSettlementRequest{},
		Response:          &apiresource.Settlement{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).GetSettlement
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSettlement,
			Fields:     []string{"allocations"},
		}),
	}
}
