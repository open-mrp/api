package machineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// UpsertMachineInput is the input for a single machine in a bulk upsert operation.
type UpsertMachineInput struct {
	// Display name of the machine. Rows match existing machines by name or serial number
	// (case-insensitive); a row whose name and serial match two different machines fails.
	Name string `json:"name" validate:"required,max=255"`
	// Serial number of the machine. Also used for matching — see `name`.
	SerialNumber string `json:"serial_number" validate:"required,max=255"`
	// Work center the machine belongs to, referenced by `id` or `name`. A row matching an existing machine must name
	// the machine's current work center; move machines with Update Machine.
	WorkCenter apirequest.ObjectIdentifier `json:"work_center" validate:"required"`
	// `active` or `archived`. Omitted, a new machine is active and an existing one keeps its status.
	Status *constants.ArchiveStatus `json:"status,omitempty"`
}

// BulkUpsertMachinesRequest is the request to bulk upsert machines.
type BulkUpsertMachinesRequest struct {
	// Machines to create or update, matched by name or serial number (case-insensitive)
	// within the account.
	Machines []UpsertMachineInput `json:"machines" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertMachinesRequest = &BulkUpsertMachinesRequest{
	Machines: []UpsertMachineInput{
		{
			Name:         apiresource.SampleMachineName,
			SerialNumber: apiresource.SampleMachineSerialNumber,
			WorkCenter: apirequest.ObjectIdentifier{
				ID:   apiresource.SampleWorkCenterID,
				Name: apiresource.SampleWorkCenterName,
			},
		},
	},
}

func (*BulkUpsertMachinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertMachinesRequest)
}

// Creates or updates many machines at once.
//
// Rows are matched to your existing machines by name or serial number (case-insensitive). The writes run in the background: the response is a 202 with an async job to poll for each row's result.
type BulkUpsertMachinesEndpoint struct{}

func (e *BulkUpsertMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertMachinesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertMachinesRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Machines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMachines, Action: types.ActionCreate}, {Domain: types.PermissionDomainMachines, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertMachinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(MachineSvc).BulkUpsertMachines
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
