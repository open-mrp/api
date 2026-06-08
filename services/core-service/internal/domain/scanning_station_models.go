package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
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
	OperatorRequirement constants.OperatorRequirement
	DepartmentID        string
	Includes            []string
}

type UpdateScanningStationParams struct {
	AccountID           string
	ScanningStationID   string
	Name                *string
	Notes               field.Clearable[string]
	LabelSizeCode       *string
	LabelTypeCode       *string
	OperatorRequirement *constants.OperatorRequirement
	Includes            []string
}

type DeleteScanningStationParams struct {
	AccountID         string
	ScanningStationID string
}

type ConnectProductionStepsByNameParams struct {
	AccountID         string
	ScanningStationID string
	Name              string
}
