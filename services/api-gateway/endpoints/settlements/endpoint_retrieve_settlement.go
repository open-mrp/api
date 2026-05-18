package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveSettlementRequest is the request to get a settlement.
type RetrieveSettlementRequest struct {
	// Settlement ID.
	SettlementID string `path:"id" validate:"required"`
	// Sub-resources to include in the response.
	Includes []string `include:"true"`
}

// Returns a settlement by ID.
type RetrieveSettlementEndpoint struct{}

func (e *RetrieveSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*RetrieveSettlementRequest, *apiresource.Settlement]{
		Title:             "Retrieve Settlement",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements/{id}",
		Request:           &RetrieveSettlementRequest{},
		Response:          &apiresource.Settlement{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).GetSettlement
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSettlement,
			Fields:     []string{"allocations"},
		}),
	}).WithDocSource(e)
}
