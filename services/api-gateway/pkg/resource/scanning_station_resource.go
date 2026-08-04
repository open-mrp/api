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

// A station on the production floor where operators scan batches to perform a batch operation, such as initializing or moving a batch.
type ScanningStation struct {
	// Scanning station ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_station"`
	// Display name of the scanning station.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Free-form notes about the scanning station.
	Notes *string `json:"notes"`
	// Scanning station type, determining which batch operation an operator performs when they scan here.
	//
	// - `init_batch`: starts a new batch at the beginning of a production flow.
	// - `merge_batch`: combines several scanned batches into one.
	// - `move_batch`: advances a batch through a production step connected to this station.
	// - `split_batch`: divides a batch into several batches.
	//
	// Fixed when the station is created.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Size of the labels printed at this station, given as width-by-height (for example, `1x1`).
	LabelSizeCode *constants.LabelSizeCode `json:"label_size"`
	// Type of label printed at this station.
	//
	// - `tag`: a label attached to the physical product.
	// - `traveler`: a routing sheet that accompanies the batch through every production step.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type"`
	// Whether operators must perform a material check at this station.
	//
	// - `none`: no additional operator check is required.
	// - `material_check`: a material check is expected before the operation.
	OperatorRequirement constants.OperatorRequirement `json:"operator_requirement"`
	// The department this scanning station belongs to.
	//
	// Assigned when the station is created and cannot be reassigned afterward.
	Department *Department `json:"department" expandable:"true"`
	// Production steps connected to this station.
	//
	// A production step can be connected to at most one scanning station, so connecting a step here disconnects it from any other station. Manage the connections with Connect Production Steps to Scanning Station.
	ProductionSteps *List[ProductionStep] `json:"production_steps" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleScanningStationNotes = "Initializes batches at the start of the packaging line."

var SampleScanningStation = &ScanningStation{
	ID:                  SampleScanningStationID,
	Object:              constants.ObjectTypeScanningStation,
	Name:                SampleScanningStationName,
	Notes:               &sampleScanningStationNotes,
	Type:                constants.ScanningStationTypeInitBatch,
	LabelSizeCode:       new(constants.LabelSizeCodeTwoByFour),
	LabelTypeCode:       new(constants.LabelTypeCodeTraveler),
	OperatorRequirement: constants.OperatorRequirementNone,
	Department:          nil,
	ProductionSteps:     nil,
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ScanningStation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScanningStation)
}
