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
type RetrievePartRequest struct {
	// Part ID.
	ItemID string `path:"id" validate:"required"`
}

type RetrievePartEndpoint struct{}

func (e *RetrievePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*RetrievePartRequest, *apiresource.Part]{
		Title:             "Retrieve Part",
		Description:       "Returns a part by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/{id}",
		Request:           &RetrievePartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).GetPart
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}
}
