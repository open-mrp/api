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

// RetrieveItemLotDefaultRequest is the request to resolve an item's lot.
type RetrieveItemLotDefaultRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns the lot this item is made in — how many, counted in what.
//
// A lot is a doff, a pallet, a batch: the quantity production is issued in. The unit is what makes it meaningful, since 60 pairs and 60 eaches are different lots, so `quantity` should never be read without `unit`.
//
// Resolved through the same chain the production schedule uses, most specific first: a per-item override, then the item's own product line, then the product lines of the finished goods it becomes, then the account-wide default. `source` names which rule applied. Intermediate items like greige are not sold and have no product line of their own, which is why they inherit from what they become.
//
// `quantity` is `0` when nothing in the chain supplies a lot. That means the item has no lot convention, not that its lot is zero.
type RetrieveItemLotDefaultEndpoint struct{}

func (e *RetrieveItemLotDefaultEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemLotDefaultRequest, *apiresource.ItemLotDefault] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemLotDefaultRequest, *apiresource.ItemLotDefault]{
		Title:               "Retrieve Item Lot Default",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/lot-default",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemLotDefault,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemLotDefaultRequest) (*apiresource.ItemLotDefault, *apierror.APIError) {
			return svc.(ItemSvc).GetItemLotDefault
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemLotDefault,
			Fields:     []string{"unit"},
		}),
	})
}
