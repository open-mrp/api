package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ---------------------------------------------------------------------------
// ScanningStation — full scanning station resource
// ---------------------------------------------------------------------------

const SampleScanningStationName = "Packaging Line 1"

// Scanning station resource.
type ScanningStation struct {
	// Scanning station ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_station"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
	// Scanning station type, determining which batch operation the station performs.
	//
	// - `init_batch`: initializes a new batch.
	// - `merge_batch`: merges multiple batches into one.
	// - `move_batch`: moves a batch to another location or step.
	// - `split_batch`: splits a batch into multiple batches.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Label size printed at this station.
	//
	// `null` when no label size is configured.
	//
	// - `1x1`: 1x1 inch label.
	// - `1x3`: 1x3 inch label.
	// - `1x4`: 1x4 inch label.
	// - `2x4`: 2x4 inch label.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size"`
	// Label type printed at this station.
	//
	// `null` when no label type is configured.
	//
	// - `tag`: a tag label.
	// - `traveler`: a traveler label that accompanies the batch through production.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type"`
	// Operator requirement behavior for this station.
	//
	// - `none`: no operator action is required to complete a scan.
	// - `material_check`: the operator must perform a material check before the scan is accepted.
	OperatorRequirement constants.OperatorRequirement `json:"operator_requirement"`
	// Department.
	Department *Department `json:"department" expandable:"true"`
	// Connected production steps.
	ProductionSteps *List[ProductionStep] `json:"production_steps" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleScanningStation = &ScanningStation{
	ID:                  SampleScanningStationID,
	Object:              constants.ObjectTypeScanningStation,
	Name:                SampleScanningStationName,
	Type:                constants.ScanningStationTypeInitBatch,
	OperatorRequirement: constants.OperatorRequirementNone,
	Department:          nil,
	ProductionSteps:     nil,
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ScanningStation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScanningStation)
}
