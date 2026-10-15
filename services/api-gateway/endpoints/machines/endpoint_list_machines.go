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
	// Case-insensitive prefix match on `name`; the machine named exactly the term comes first.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return machines at one of these work centers.
	WorkCenterIDs []string `query:"work_center_ids" validate:"omitempty,max=100"`
	// Only return machines with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of your machines, newest first.
//
// With `q`, the machine named exactly the term is listed first.
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
		ObjectType:        constants.ObjectTypeMachine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"work_center"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError) {
			return svc.(MachineSvc).ListMachines
		},
	})
}
