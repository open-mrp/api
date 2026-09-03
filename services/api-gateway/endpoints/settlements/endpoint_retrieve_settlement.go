package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a settlement.
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
		Title:               "Retrieve Settlement",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/finance/settlements/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSettlements, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeSettlement,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).GetSettlement
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSettlement,
			Fields:     []string{"responsible_user", "responsible_user.user", "allocations", "allocations.amount", "allocations.amount.unit", "allocations.transaction", "allocations.transaction.amount", "allocations.transaction.amount.unit"},
		}),
	})
}
