package machinestatusep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request for the state of the floor.
type ListMachineStatusRequest struct {
	// Only include machines in these departments.
	DepartmentIDs []string `query:"department_ids"`
	// The moment to read the floor at. Defaults to now.
	AsOf *time.Time `query:"as_of"`
}

// Returns what every machine is running right now, how much is left on it, and what is queued behind that.
//
// Assembled from the published schedule, the batches the floor has scanned against each campaign, and any open downtime. A campaign is `current` once its week is released and while it still has batches to scan; when the last one is scanned it hands over to the next, so this advances on its own as a shift progresses.
//
// A machine with an open stoppage reads `down` even when it has a released campaign, because a broken machine is not producing whatever the plan says. A machine with nothing released reads `idle`, which is a state worth seeing rather than an absence from the list.
//
// Reads the published version rather than the newest draft: the floor works to what was committed, and a draft regenerating underneath a wall display would make machines appear to change job on their own. With nothing published every machine reads idle rather than the request failing.
type ListMachineStatusEndpoint struct{}

func (e *ListMachineStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachineStatusRequest, *apiresource.List[apiresource.MachineStatus]] {
	return (&apiendpoint.APIEndpoint[*ListMachineStatusRequest, *apiresource.List[apiresource.MachineStatus]]{
		Title:               "List Machine Status",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/machine-status",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		AgentTool:           true,
		ObjectType:          constants.ObjectTypeMachineStatus,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMachines, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachineStatusRequest) (*apiresource.List[apiresource.MachineStatus], *apierror.APIError) {
			return svc.(MachineStatusSvc).ListMachineStatus
		},
	})
}
