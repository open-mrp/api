package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a service level.
type RetrieveServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

type RetrieveServiceLevelEndpoint struct{}

func (e *RetrieveServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveServiceLevelRequest, *apiresource.ServiceLevel] {
	return &apiendpoint.APIEndpoint[*RetrieveServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Retrieve Service Level",
		Description:       "Returns a service level by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		Request:           &RetrieveServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).GetServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
