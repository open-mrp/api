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
type RetrievePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
}

// Returns a property by ID.
type RetrievePropertyEndpoint struct{}

func (e *RetrievePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*RetrievePropertyRequest, *apiresource.Property]{
		Title:             "Retrieve Property",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/{id}",
		Request:           &RetrievePropertyRequest{},
		Response:          &apiresource.Property{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).GetProperty
		},
	}).WithDocSource(e)
}
