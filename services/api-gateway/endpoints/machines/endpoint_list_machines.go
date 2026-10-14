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

// Request to list machines.
type ListMachinesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of machines in your account, most recently created first.
//
// The search term matches the start of the machine name, and a machine named exactly the term is listed
// first.
type ListMachinesEndpoint struct{}

func (e *ListMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachinesRequest, *apiresource.List[apiresource.Machine]] {
	return (&apiendpoint.APIEndpoint[*ListMachinesRequest, *apiresource.List[apiresource.Machine]]{
		Title:             "List Machines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMachine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError) {
			return svc.(MachineSvc).ListMachines
		},
	})
}
