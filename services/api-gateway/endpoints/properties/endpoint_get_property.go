package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a property.
type GetPropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
}

type GetPropertyEndpoint struct{}

func (e *GetPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPropertyRequest, *apiresource.Property] {
	return &apiendpoint.APIEndpoint[*GetPropertyRequest, *apiresource.Property]{
		Title:             "Get Property",
		Description:       "Returns a property by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/{id}",
		Request:           &GetPropertyRequest{},
		Response:          &apiresource.Property{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).GetProperty
		},
	}
}
