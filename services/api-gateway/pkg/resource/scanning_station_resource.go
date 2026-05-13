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
	// Scanning station type.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Label size code.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size"`
	// Label type code.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type"`
	// Operator requirement behavior for this station.
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
