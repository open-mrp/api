package scanningstationep

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

// Input for a single scanning station in a bulk upsert operation.
type UpsertScanningStationInput struct {
	// Display name of the scanning station. Rows are matched against existing stations
	// by name (case-insensitive): a match updates that station, no match creates a new
	// one.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the scanning station. Preserved when omitted on update.
	Notes field.Optional[string] `json:"notes,omitzero"`
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
	// Department this station belongs to, referenced by `id` or `name`. Create-only: a row
	// updating an existing station must state that station's current department.
	Department apirequest.ObjectIdentifier `json:"department" validate:"required"`
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
			Department: apirequest.ObjectIdentifier{
				ID:   apiresource.SampleDepartmentID,
				Name: apiresource.SampleDepartmentName,
			},
		},
	},
}

func (*BulkUpsertScanningStationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertScanningStationsRequest)
}

// Creates or updates multiple scanning stations for the account, matched by name (case-insensitive).
// Validates and resolves synchronously, then writes asynchronously — 202 with a job to poll.
type BulkUpsertScanningStationsEndpoint struct{}

func (e *BulkUpsertScanningStationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertScanningStationsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertScanningStationsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Scanning Stations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertScanningStationsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ScanningStationSvc).BulkUpsertScanningStations
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
