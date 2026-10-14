package demandoverridesep

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

// Request to retrieve a demand override.
type RetrieveDemandOverrideRequest struct {
	// ID of the demand override.
	DemandOverrideID string `path:"id" validate:"required"`
}

// Retrieves a single demand override by ID.
type RetrieveDemandOverrideEndpoint struct{}

func (e *RetrieveDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDemandOverrideRequest, *apiresource.DemandOverride] {
	return (&apiendpoint.APIEndpoint[*RetrieveDemandOverrideRequest, *apiresource.DemandOverride]{
		Title:             "Retrieve Demand Override",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
			return svc.(DemandOverridesSvc).GetDemandOverride
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDemandOverride,
			Fields:     []string{"scope"},
		}),
	})
}
