package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// DepartmentScanningStation is a scanning station sub-resource attached to a department.
type DepartmentScanningStation struct {
	ID                  string
	Name                string
	Type                string
	OperatorRequirement string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DepartmentMachine is a machine sub-resource attached to a department.
type DepartmentMachine struct {
	ID           string
	Name         string
	SerialNumber string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Department struct {
	ID               string
	Name             string                      `audit:"name"`
	Notes            *string                     `audit:"notes"`
	LocationID       *string                     `audit:"location_id"`
	LocationName     *string                     `audit:"location_name"`
	LocationTypeCode *string                     `audit:"location_type_code"`
	ScanningStations []DepartmentScanningStation `audit:"scanning_stations"`
	Machines         []DepartmentMachine         `audit:"machines"`
	AccountID        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListDepartmentsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListDepartmentsResult struct {
	Departments []*Department
	PageInfo    pagination.PageInfo
}

type GetDepartmentParams struct {
	AccountID    string
	DepartmentID string
}

type CreateDepartmentParams struct {
	AccountID          string
	Name               string
	Notes              *string
	LocationID         *string
	ScanningStationIDs []string
	MachineIDs         []string
}

type UpdateDepartmentParams struct {
	AccountID          string
	DepartmentID       string
	Name               *string
	Notes              *string
	LocationID         *string
	ScanningStationIDs []string
	MachineIDs         []string
}

type DeleteDepartmentParams struct {
	AccountID    string
	DepartmentID string
}
