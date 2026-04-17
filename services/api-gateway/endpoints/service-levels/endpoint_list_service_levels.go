package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list service levels.
type ListServiceLevelsRequest struct {
	apiresource.PaginationRequest
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
}

type ListServiceLevelsEndpoint struct{}

func (e *ListServiceLevelsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListServiceLevelsRequest, *apiresource.List[apiresource.ServiceLevel]] {
	return &apiendpoint.APIEndpoint[*ListServiceLevelsRequest, *apiresource.List[apiresource.ServiceLevel]]{
		Title:             "List Service Levels",
		Description:       "Returns a paginated list of service levels for a carrier.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		Request:           &ListServiceLevelsRequest{},
		Response:          &apiresource.List[apiresource.ServiceLevel]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError) {
			return svc.(ServiceLevelSvc).ListServiceLevels
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
