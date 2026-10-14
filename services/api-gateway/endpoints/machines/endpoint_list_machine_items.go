package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// Request to list items a machine is configured to run.
type ListMachineItemsRequest struct {
	apiresource.PaginationRequest
	// ID of the machine.
	MachineID string `path:"id" validate:"required"`
}

// Returns the items a machine is configured to run, newest association first.
//
// `q` matches against item SKU and description.
type ListMachineItemsEndpoint struct{}

func (e *ListMachineItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachineItemsRequest, *apiresource.List[apiresource.Item]] {
	return (&apiendpoint.APIEndpoint[*ListMachineItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:             "List Machine Items",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}/items",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeItem,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(MachineSvc).ListMachineItems
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "attributes"},
		}),
	})
}
