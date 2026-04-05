package domain

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
	Production        StepProduction
	Consumptions      []StepConsumption
	InStepIDs         []string
	OutStepIDs        []string
	ScanningStationID *string
	LevelingFactor    string
	Allowances        string
	LaborRate         *FlowRate
	LaborTime         *FlowRate
	OverheadRate      *FlowRate
}

// FlowRate represents a rate value with numerator and denominator unit references.
type FlowRate struct {
	ID                string
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}
