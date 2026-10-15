package scanningstationep

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

// Input for a single scanning station in a bulk upsert operation.
type UpsertScanningStationInput struct {
	// Display name of the scanning station. Rows are matched against existing stations
	// by name (case-insensitive): a match updates that station, no match creates a new
	// one.
	Name string `json:"name" validate:"required,max=255"`
	// Scanning station type, determining which batch operation the station performs.
	//
	// - `init_batch`: initializes a new batch.
	// - `merge_batch`: merges multiple batches into one.
	// - `move_batch`: moves a batch to another location or step.
	// - `split_batch`: splits a batch into multiple batches.
	//
	// The type cannot be changed after creation — rows updating an existing station
	// must state that station's current type.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Whether operators must perform a material check at this station.
	//
	// - `none`: no additional operator check is required.
	// - `material_check`: a material check is expected before the operation.
	OperatorRequirement constants.OperatorRequirement `json:"operator_requirement" validate:"required"`
	// Size of the labels printed at this station, given as width-by-height (for
	// example, `1x1`). Preserved when omitted on update; send `null` or an empty
	// string to clear.
	LabelSizeCode field.Clearable[constants.LabelSizeCode] `json:"label_size,omitzero"`
	// Type of label printed at this station. Preserved when omitted on update; send
	// `null` or an empty string to clear.
	//
	// - `tag`: a label attached to the physical product.
	// - `traveler`: a routing sheet that accompanies the batch through every production step.
	LabelTypeCode field.Clearable[constants.LabelTypeCode] `json:"label_type,omitzero"`
	// Work center the station belongs to, referenced by `id` or `name`. A row matching an
	// existing station must name the station's current work center; move stations with
	// Update Scanning Station.
	WorkCenter apirequest.ObjectIdentifier `json:"work_center" validate:"required"`
}

// Request to bulk upsert scanning stations.
type BulkUpsertScanningStationsRequest struct {
	// Scanning stations to create or update, matched by name (case-insensitive) within
	// the account.
	ScanningStations []UpsertScanningStationInput `json:"scanning_stations" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertScanningStationsRequest = &BulkUpsertScanningStationsRequest{
	ScanningStations: []UpsertScanningStationInput{
		{
			Name:                apiresource.SampleScanningStationName,
			Type:                constants.ScanningStationTypeInitBatch,
			OperatorRequirement: constants.OperatorRequirementNone,
			WorkCenter: apirequest.ObjectIdentifier{
				ID:   apiresource.SampleWorkCenterID,
				Name: apiresource.SampleWorkCenterName,
			},
		},
	},
}

func (*BulkUpsertScanningStationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertScanningStationsRequest)
}

// Creates or updates many scanning stations at once.
//
// Rows are matched to your existing stations by name (case-insensitive). The writes run in the background: the response is a 202 with an async job to poll for each row's result.
type BulkUpsertScanningStationsEndpoint struct{}

func (e *BulkUpsertScanningStationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertScanningStationsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertScanningStationsRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Scanning Stations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionCreate}, {Domain: types.PermissionDomainScanningStations, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertScanningStationsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ScanningStationSvc).BulkUpsertScanningStations
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
