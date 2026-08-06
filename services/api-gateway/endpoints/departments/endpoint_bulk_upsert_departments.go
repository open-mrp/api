package departmentep

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

// UpsertDepartmentInput is the input for a single department in a bulk upsert
// operation.
type UpsertDepartmentInput struct {
	// Display name of the department, used to match existing departments within the
	// account (case-insensitive). If it exists the department is updated in place;
	// otherwise a new department is created.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the department. Preserved when omitted on update.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Location where this department operates, referenced by `id` or `name`. Preserved
	// when omitted on update.
	Location field.Optional[apirequest.ObjectIdentifier] `json:"location,omitzero"`
}

// BulkUpsertDepartmentsRequest is the request to bulk upsert departments.
type BulkUpsertDepartmentsRequest struct {
	// Departments to create or update, matched by name (case-insensitive) within the
	// account.
	Departments []UpsertDepartmentInput `json:"departments" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertDepartmentsRequest = &BulkUpsertDepartmentsRequest{
	Departments: []UpsertDepartmentInput{
		{Name: apiresource.SampleDepartmentName},
	},
}

func (*BulkUpsertDepartmentsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertDepartmentsRequest)
}

// Creates or updates multiple departments matched by name (case-insensitive), then writes
// asynchronously — 202 with a job to poll. Scanning stations and machines are not assigned here.
type BulkUpsertDepartmentsEndpoint struct{}

func (e *BulkUpsertDepartmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertDepartmentsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertDepartmentsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Departments",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertDepartmentsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(DepartmentSvc).BulkUpsertDepartments
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
