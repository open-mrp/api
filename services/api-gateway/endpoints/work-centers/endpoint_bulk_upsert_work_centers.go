package workcenterep

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
	"github.com/open-mrp/apikit/field"
)

// UpsertWorkCenterInput is one work center in a bulk upsert.
type UpsertWorkCenterInput struct {
	// Display name of the work center, which matches an existing one (case-insensitive) to update; otherwise a new
	// work center is created.
	Name string `json:"name" validate:"required,max=255"`
	// Location where the work center operates, referenced by `id` or `name`. Must be active. Omitted, an existing work
	// center keeps its location.
	Location field.Optional[apirequest.ObjectIdentifier] `json:"location,omitzero"`
}

// Request to bulk upsert work centers.
type BulkUpsertWorkCentersRequest struct {
	// Work centers to create or update, matched by name (case-insensitive).
	WorkCenters []UpsertWorkCenterInput `json:"work_centers" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertWorkCentersRequest = &BulkUpsertWorkCentersRequest{
	WorkCenters: []UpsertWorkCenterInput{
		{Name: apiresource.SampleWorkCenterName},
	},
}

func (*BulkUpsertWorkCentersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertWorkCentersRequest)
}

// Creates or updates many work centers at once.
//
// Rows are matched to your existing work centers by name (case-insensitive). The writes run in the background: the response is a 202 with an async job to poll for each row's result. Machines and scanning stations name their work center in their own bulk upserts.
type BulkUpsertWorkCentersEndpoint struct{}

func (e *BulkUpsertWorkCentersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertWorkCentersRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertWorkCentersRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Work Centers",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDepartments, Action: types.ActionCreate}, {Domain: types.PermissionDomainDepartments, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(WorkCenterSvc).BulkUpsertWorkCenters
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
