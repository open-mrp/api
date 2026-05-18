package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list machines.
type ListMachinesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of machines for the target account.
type ListMachinesEndpoint struct{}

func (e *ListMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachinesRequest, *apiresource.List[apiresource.Machine]] {
	return (&apiendpoint.APIEndpoint[*ListMachinesRequest, *apiresource.List[apiresource.Machine]]{
		Title:             "List Machines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachinesRequest) (*apiresource.List[apiresource.Machine], *apierror.APIError) {
			return svc.(MachineSvc).ListMachines
		},
	})
}
