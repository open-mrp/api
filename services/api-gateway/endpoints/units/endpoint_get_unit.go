package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetUnitRequest is the request to retrieve a single unit.
type GetUnitRequest struct {
	// The ID of the unit to retrieve.
	UnitID string `path:"id" validate:"required"`
}

type GetUnitEndpoint struct{}

func (e *GetUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUnitRequest, *apiresource.Unit] {
	return &apiendpoint.APIEndpoint[*GetUnitRequest, *apiresource.Unit]{
		Title:             "Get Unit",
		Description:       "Returns a single unit by its ID, including both account-owned and global system units.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		Request:           &GetUnitRequest{},
		Response:          &apiresource.Unit{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).GetUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
