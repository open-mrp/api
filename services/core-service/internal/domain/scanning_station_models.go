package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

type ScanningStation struct {
	ID                  string
	Name                string                        `audit:"name"`
	Notes               *string                       `audit:"notes"`
	Type                constants.ScanningStationType `audit:"type"`
	LabelSizeCode       *string                       `audit:"label_size_code"`
	LabelTypeCode       *string                       `audit:"label_type_code"`
	OperatorRequirement constants.OperatorRequirement `audit:"operator_requirement"`
	DepartmentID        string                        `audit:"department_id"`
	DepartmentName      string
	DepartmentCreatedAt *time.Time
	DepartmentUpdatedAt *time.Time
	ProductionSteps     []ProductionStepRef
	AccountID           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ProductionStepRef is a reference to a production step with the minimal required fields.
type ProductionStepRef struct {
	ID             string
	Name           string
	LevelingFactor string
	Allowances     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListScanningStationsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
}

type ListScanningStationsResult struct {
	ScanningStations []*ScanningStation
	PageInfo         pagination.PageInfo
}

// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportScanningStationsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type GetScanningStationParams struct {
	AccountID         string
	ScanningStationID string
	Includes          []string
}

type CreateScanningStationParams struct {
	AccountID           string
	Name                string
	Notes               *string
	Type                constants.ScanningStationType
	LabelSizeCode       *string
	LabelTypeCode       *string
	OperatorRequirement constants.OperatorRequirement
	DepartmentID        string
	Includes            []string
}

type UpdateScanningStationParams struct {
	AccountID           string
	ScanningStationID   string
	Name                *string
	Notes               field.Clearable[string]
	LabelSizeCode       field.Clearable[string]
	LabelTypeCode       field.Clearable[string]
	OperatorRequirement *constants.OperatorRequirement
	Includes            []string
}

type DeleteScanningStationParams struct {
	AccountID         string
	ScanningStationID string
}

type UpsertScanningStationParams struct {
	Name                string
	Notes               *string
	Type                constants.ScanningStationType
	LabelSizeCode       field.Clearable[string]
	LabelTypeCode       field.Clearable[string]
	OperatorRequirement constants.OperatorRequirement
	Department          ObjectIdentifier
}

type BulkUpsertScanningStationsParams struct {
	ScanningStations []UpsertScanningStationParams
}

// ResolvedUpsertScanningStationRow is a scanning station upsert row with its department
// reference resolved to an id. The engine round-trips job_items against this type, so it
// carries no JSON tags — except the Clearable label fields, whose MarshalJSON errors on an
// unset value and so require `omitzero` to be skipped.
type ResolvedUpsertScanningStationRow struct {
	Name                string
	Notes               *string
	Type                constants.ScanningStationType
	LabelSizeCode       field.Clearable[string] `json:",omitzero"`
	LabelTypeCode       field.Clearable[string] `json:",omitzero"`
	OperatorRequirement constants.OperatorRequirement
	DepartmentID        string
}

type ConnectProductionStepsByNameParams struct {
	AccountID         string
	ScanningStationID string
	Name              string
}
