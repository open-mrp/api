package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemRequest is the request to retrieve a single item by ID.
type GetItemRequest struct {
	// The ID of the item to retrieve.
	ItemID string `path:"id" validate:"required"`
}

type GetItemEndpoint struct{}

func (e *GetItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemRequest, *apiresource.Item] {
	return &apiendpoint.APIEndpoint[*GetItemRequest, *apiresource.Item]{
		Title:             "Get Item",
		Description:       "Returns a single item by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/items/{id}",
		Request:           &GetItemRequest{},
		Response:          &apiresource.Item{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).GetItem
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate"},
		}),
	}
}
