package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove an attribute from an item.
type RemoveItemAttributeRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// ID of the attribute to unassign from the item.
	AttributeID string `path:"attribute_id" validate:"required"`
}

// Unassigns an attribute from an item and returns the updated item.
//
// Returns a not-found error if the attribute is not currently assigned to the item, so unlike adding an attribute, this call is not safe to repeat blindly. The attribute itself is not deleted and stays available for other items.
type RemoveItemAttributeEndpoint struct{}

func (e *RemoveItemAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemAttributeRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*RemoveItemAttributeRequest, *apiresource.Item]{
		Title:               "Remove Item Attribute",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/attributes/{attribute_id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemAttributeRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).RemoveItemAttribute
		},
		ObjectType: constants.ObjectTypeItem,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
