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
)

// Request to retrieve an item by ID.
type RetrieveItemRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Retrieves an item.
type RetrieveItemEndpoint struct{}

func (e *RetrieveItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemRequest, *apiresource.Item]{
		Title:             "Retrieve Item",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             itemRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).GetItem
		},
		ObjectType:    constants.ObjectTypeItem,
		IncludeConfig: itemIncludes(),
	})
}
