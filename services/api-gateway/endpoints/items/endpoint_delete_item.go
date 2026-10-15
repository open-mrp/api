package itemep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an item.
type DeleteItemRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Deletes an item.
//
// Only while nothing references it: no order, inventory, production, price or supplier record. Archive an item that's in use by setting `status` to `archived`.
type DeleteItemEndpoint struct{}

func (e *DeleteItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteItemRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteItemRequest, *object.Deleted]{
		Title:             "Delete Item",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             itemRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionDelete}}},
		ObjectType:        constants.ObjectTypeItem,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteItemRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ItemSvc).DeleteItem, constants.ObjectTypeItem, "id")
		},
	})
}
