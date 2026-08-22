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

// Request to retrieve an item by ID.
type RetrieveItemRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns a single item by ID.
type RetrieveItemEndpoint struct{}

func (e *RetrieveItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item]{
		Title:               "Retrieve Item",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).GetItem
		},
		ObjectType: constants.ObjectTypeItem,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
