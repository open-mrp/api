package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// AddItemAttributeRequest is the request to add an attribute to an item.
type AddItemAttributeRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"attribute_id" validate:"required"`
}

// Adds an attribute to an item. If the attribute is already associated with the item, this is a no-op.
type AddItemAttributeEndpoint struct{}

func (e *AddItemAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemAttributeRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*AddItemAttributeRequest, *apiresource.Item]{
		Title:             "Add Item Attribute",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/attributes/{attribute_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).AddItemAttribute
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
