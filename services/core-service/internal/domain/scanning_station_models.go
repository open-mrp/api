package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

type ScanningStation struct {
	ID                    string
	Name                  string                        `audit:"name"`
	Notes                 *string                       `audit:"notes"`
	Type                  constants.ScanningStationType `audit:"type"`
	LabelSizeCode         *string                       `audit:"label_size_code"`
	LabelTypeCode         *string                       `audit:"label_type_code"`
	MaterialCheckRequired bool                          `audit:"material_check_required"`
	DepartmentID          string                        `audit:"department_id"`
	DepartmentName        string
	ProductionSteps       []LightRef
	AccountID             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ListScanningStationsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListScanningStationsResult struct {
	ScanningStations []*ScanningStation
	PageInfo         pagination.PageInfo
}

type GetScanningStationParams struct {
	AccountID         string
	ScanningStationID string
}

type CreateScanningStationParams struct {
	AccountID             string
	Name                  string
	Notes                 *string
	Type                  constants.ScanningStationType
	MaterialCheckRequired bool
	DepartmentID          string
}

type UpdateScanningStationParams struct {
	AccountID             string
	ScanningStationID     string
	Name                  *string
	Notes                 *string
	LabelSizeCode         *string
	LabelTypeCode         *string
	MaterialCheckRequired *bool
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
