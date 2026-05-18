package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a unit.
type RetrieveUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
}

// Returns a unit by ID, including both account-owned and global system units.
type RetrieveUnitEndpoint struct{}

func (e *RetrieveUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*RetrieveUnitRequest, *apiresource.Unit]{
		Title:             "Retrieve Unit",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).GetUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
