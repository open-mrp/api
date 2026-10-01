package constants

// ProductionRunBatchScope chooses which batches a production run's batch list returns.
type ProductionRunBatchScope string

const (
	// ProductionRunBatchScopeFlow returns the run's batches plus the in-progress batch flow around them.
	ProductionRunBatchScopeFlow ProductionRunBatchScope = "flow"
	// ProductionRunBatchScopeRun returns only the batches created under the run.
	ProductionRunBatchScopeRun ProductionRunBatchScope = "run"
)

func (s ProductionRunBatchScope) IsValid() bool {
	switch s {
	case ProductionRunBatchScopeFlow, ProductionRunBatchScopeRun:
		return true
	default:
		return false
	}
}

func (s ProductionRunBatchScope) EnumValues() []string {
	return []string{
		string(ProductionRunBatchScopeFlow),
		string(ProductionRunBatchScopeRun),
	}
}
