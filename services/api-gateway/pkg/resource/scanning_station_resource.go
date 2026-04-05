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

// ScanningStation represents a scanning station with all its details.
type ScanningStation struct {
	// The unique identifier for the scanning station.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_station"`
	// The display name of the scanning station.
	Name string `json:"name" validate:"required"`
	// Optional notes about the scanning station.
	Notes *string `json:"notes"`
	// The type of scanning station.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// The label size code for the scanning station.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size_code"`
	// The label type code for the scanning station.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type_code"`
	// Whether material check is required at this station.
	MaterialCheckRequired bool `json:"material_check_required"`
	// The department this scanning station belongs to.
	Department *Department `json:"department" expandable:"true"`
	// The production steps connected to this scanning station.
	ProductionSteps *List[ProductionStep] `json:"production_steps" expandable:"true"`
	// The timestamp when the scanning station was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the scanning station was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleScanningStation = &ScanningStation{
	ID:                    SampleScanningStationID,
	Object:                constants.ObjectTypeScanningStation,
	Name:                  SampleScanningStationName,
	Type:                  constants.ScanningStationTypeInitBatch,
	MaterialCheckRequired: false,
	Department:            nil,
	ProductionSteps:       nil,
	CreatedAt:             timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:             timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ScanningStation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScanningStation)
}
