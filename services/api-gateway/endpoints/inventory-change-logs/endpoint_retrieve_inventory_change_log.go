package inventorychangelogep

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

// Request to retrieve an inventory change log.
type RetrieveInventoryChangeLogRequest struct {
	// Inventory change log ID.
	InventoryChangeLogID string `path:"id" validate:"required"`
}

// Returns an inventory change log by ID.
type RetrieveInventoryChangeLogEndpoint struct{}

func (e *RetrieveInventoryChangeLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveInventoryChangeLogRequest, *apiresource.InventoryChangeLog] {
	return (&apiendpoint.APIEndpoint[*RetrieveInventoryChangeLogRequest, *apiresource.InventoryChangeLog]{
		Title:             "Retrieve Inventory Change Log",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeInventoryChangeLog,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).GetInventoryChangeLog
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "responsible_user", "responsible_scanning_station"},
		}),
	})
}
