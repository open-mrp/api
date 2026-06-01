package domain

import "time"

// StepEdge represents a parent→child edge in the production step graph.
type StepEdge struct {
	ParentStepID string `audit:"parent_step_id"`
	ChildStepID  string `audit:"child_step_id"`
}

// ProductionFlowStep represents a single step in the production flow with all
// associated data needed for flow display.
type ProductionFlowStep struct {
	ID                string
	Name              string
	Notes             *string
	Production        StepProduction
	Consumptions      []StepConsumption
	InStepIDs         []string
	OutStepIDs        []string
	ScanningStationID *string
	DepartmentID      *string
	MachineIDs        []string
	LevelingFactor    string
	Allowances        string
	LaborRate         *FlowRate
	LaborTime         *FlowRate
	OverheadRate      *FlowRate
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// FlowRate represents a rate value with numerator and denominator unit references.
type FlowRate struct {
	ID                string
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}
