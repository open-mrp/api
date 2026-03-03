package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetUnitRequest is the request to retrieve a single unit.
type GetUnitRequest struct {
	// The ID of the unit to retrieve.
	UnitID string `path:"id"`
}

const getUnitEndpointDescription string = `This endpoint returns a single unit by its ID.
The unit must belong to the requesting account or be a system (global) unit.`

type GetUnitEndpoint struct{}

func (e *GetUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUnitRequest, *apiresource.Unit] {
	return &apiendpoint.APIEndpoint[*GetUnitRequest, *apiresource.Unit]{
		Title:             "Get Unit",
		Description:       getUnitEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/units/{id}",
		ContentType:       "application/json",
		Request:           &GetUnitRequest{},
		Response:          apiresource.SampleUnit,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).GetUnit
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
