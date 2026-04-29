package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a part by ID.
type GetPartRequest struct {
	// Part ID.
	ItemID string `path:"id" validate:"required"`
}

type GetPartEndpoint struct{}

func (e *GetPartEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*GetPartRequest, *apiresource.Part]{
		Title:             "Get Part",
		Description:       "Returns a part by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/parts/{id}",
		Request:           &GetPartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).GetPart
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}
}
