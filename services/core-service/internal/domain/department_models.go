package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// LightRef is a lightweight reference with an ID and Name, used for sub-resource lists
// (e.g., scanning stations, machines attached to a department).
type LightRef struct {
	ID   string
	Name string
}

type Department struct {
	ID               string
	Name             string     `audit:"name"`
	Notes            *string    `audit:"notes"`
	LocationID       *string    `audit:"location_id"`
	LocationName     *string    `audit:"location_name"`
	ScanningStations []LightRef `audit:"scanning_stations"`
	Machines         []LightRef `audit:"machines"`
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
