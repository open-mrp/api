package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetDCLocationRequest is the request to retrieve a single DC location.
type GetDCLocationRequest struct {
	// The ID of the DC location to retrieve.
	DCLocationID string `path:"id" validate:"required"`
}

type GetDCLocationEndpoint struct{}

func (e *GetDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetDCLocationRequest, *apiresource.DCLocation] {
	return &apiendpoint.APIEndpoint[*GetDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Get DC Location",
		Description:       "Returns a single DC location by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		Request:           &GetDCLocationRequest{},
		Response:          &apiresource.DCLocation{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).GetDCLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDCLocation,
			Fields:     []string{"customer"},
		}),
	}
}
