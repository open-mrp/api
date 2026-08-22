package machinedowntimeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list downtime reasons.
type ListMachineDowntimeReasonsRequest struct{}

// Returns the downtime reasons available when logging a stoppage.
//
// The list is the same for every account and is ordered for display, so it can be rendered straight into a reason picker. Each reason carries the OEE term its stoppages charge, which is what makes the choice of reason matter beyond labeling.
type ListMachineDowntimeReasonsEndpoint struct{}

func (e *ListMachineDowntimeReasonsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachineDowntimeReasonsRequest, *apiresource.List[apiresource.MachineDowntimeReason]] {
	return (&apiendpoint.APIEndpoint[*ListMachineDowntimeReasonsRequest, *apiresource.List[apiresource.MachineDowntimeReason]]{
		Title:             "List Machine Downtime Reasons",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-reasons",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeReason,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachineDowntimeReasonsRequest) (*apiresource.List[apiresource.MachineDowntimeReason], *apierror.APIError) {
			return svc.(MachineDowntimeSvc).ListDowntimeReasons
		},
	})
}
