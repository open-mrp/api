package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to get a location type.
type RetrieveLocationTypeRequest struct {
	// Location type ID or code, such as `building` or `bin`.
	Identifier string `path:"id" validate:"required"`
}

// Returns a location type by ID or code.
type RetrieveLocationTypeEndpoint struct{}

func (e *RetrieveLocationTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType] {
	return (&apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType]{
		Title:               "Retrieve Location Type",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/location-types/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError) {
			return svc.(LocationSvc).GetLocationType
		},
	})
}
