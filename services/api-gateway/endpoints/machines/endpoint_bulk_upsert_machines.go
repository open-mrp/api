package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// UpsertMachineInput is the input for a single machine in a bulk upsert operation.
type UpsertMachineInput struct {
	// Display name of the machine. Rows match existing machines by name or serial number
	// (case-insensitive); a row whose name and serial match two different machines fails.
	Name string `json:"name" validate:"required,max=255"`
	// Serial number of the machine. Also used for matching — see `name`.
	SerialNumber string `json:"serial_number" validate:"required,max=255"`
	// Free-form notes about the machine. Preserved when omitted on update.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Department this machine belongs to, referenced by `id` or `name`. Create-only.
	Department apirequest.ObjectIdentifier `json:"department" validate:"required"`
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
			Department: apirequest.ObjectIdentifier{
				ID:   apiresource.SampleDepartmentID,
				Name: apiresource.SampleDepartmentName,
			},
		},
	},
}

func (*BulkUpsertMachinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertMachinesRequest)
}

// Creates or updates multiple machines for the account, matched by name or serial number
// (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertMachinesEndpoint struct{}

func (e *BulkUpsertMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertMachinesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertMachinesRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Machines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertMachinesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(MachineSvc).BulkUpsertMachines
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
