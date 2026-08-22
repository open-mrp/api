package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to add an attribute to an item.
type AddItemAttributeRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// ID of the attribute to assign to the item.
	AttributeID string `path:"attribute_id" validate:"required"`
}

// Assigns an attribute to an item and returns the updated item.
//
// The attribute's property must be one the item's category carries, so link the property to the category before assigning any of its attributes.
//
// Adding an attribute the item already carries succeeds and changes nothing, so the call is safe to repeat.
type AddItemAttributeEndpoint struct{}

func (e *AddItemAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemAttributeRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*AddItemAttributeRequest, *apiresource.Item]{
		Title:               "Add Item Attribute",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/attributes/{attribute_id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).AddItemAttribute
		},
		ObjectType: constants.ObjectTypeItem,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
