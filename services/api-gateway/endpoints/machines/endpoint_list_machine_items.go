package machineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list items a machine is configured to run.
type ListMachineItemsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(MachineSvc).ListMachineItems
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "attributes"},
		}),
	})
}
