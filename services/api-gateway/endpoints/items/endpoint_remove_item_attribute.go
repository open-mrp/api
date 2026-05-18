package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RemoveItemAttributeRequest is the request to remove an attribute from an item.
type RemoveItemAttributeRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"attribute_id" validate:"required"`
}

// Removes an attribute from an item.
type RemoveItemAttributeEndpoint struct{}

func (e *RemoveItemAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemAttributeRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*RemoveItemAttributeRequest, *apiresource.Item]{
		Title:             "Remove Item Attribute",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/attributes/{attribute_id}",
		Request:           &RemoveItemAttributeRequest{},
		Response:          &apiresource.Item{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).RemoveItemAttribute
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	}).WithDocSource(e)
}
