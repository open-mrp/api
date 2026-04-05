package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type Machine struct {
	ID               string
	Name             string  `audit:"name"`
	SerialNumber     string  `audit:"serial_number"`
	Notes            *string `audit:"notes"`
	DepartmentID     *string
	DepartmentName   *string `audit:"department_name"`
	ProductionStepID *string `audit:"production_step_id"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListMachinesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListMachinesResult struct {
	Machines []*Machine
	PageInfo pagination.PageInfo
}

type GetMachineParams struct {
	AccountID string
	MachineID string
}

type CreateMachineParams struct {
	AccountID    string
	Name         string
	SerialNumber string
	Notes        *string
	DepartmentID string
}

type UpdateMachineParams struct {
	AccountID    string
	MachineID    string
	Name         *string
	SerialNumber *string
	Notes        *string
}

type DeleteMachineParams struct {
	AccountID string
	MachineID string
}
