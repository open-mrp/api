package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveItemRequest is the request to get an item by ID.
type RetrieveItemRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

func (*RetrieveItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&RetrieveItemRequest{
		ItemID: apiresource.SampleItemID,
	})
}

// Returns an item by ID.
type RetrieveItemEndpoint struct{}

func (e *RetrieveItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item]{
		Title:             "Retrieve Item",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).GetItem
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
