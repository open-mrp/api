package locationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to get a location type.
type RetrieveLocationTypeRequest struct {
	// Location type ID or code, such as `building` or `bin`.
	Identifier string `path:"id" validate:"required"`
}

// Returns a location type by ID or code.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge has no such route.
type RetrieveLocationTypeEndpoint struct{}

func (e *RetrieveLocationTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType] {
	return (&apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType]{
		Title:             "Retrieve Location Type",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/location-types/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError) {
			return svc.(LocationSvc).GetLocationType
		},
	})
}
