package carrierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a carrier by ID.
type RetrieveCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Returns a carrier by ID.
type RetrieveCarrierEndpoint struct{}

func (e *RetrieveCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*RetrieveCarrierRequest, *apiresource.Carrier]{
		Title:             "Retrieve Carrier",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		Preview:           true,
		ObjectType:        constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).GetCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	})
}
