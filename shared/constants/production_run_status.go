package constants

// ProductionRunStatus filters a list of production runs by whether the run has completed.
type ProductionRunStatus string

const (
	// ProductionRunStatusOpen returns runs that still have batches left to scan.
	ProductionRunStatusOpen ProductionRunStatus = "open"
	// ProductionRunStatusClosed returns runs whose batches have all been scanned or deleted.
	ProductionRunStatusClosed ProductionRunStatus = "closed"
)

func (s ProductionRunStatus) IsValid() bool {
	switch s {
	case ProductionRunStatusOpen, ProductionRunStatusClosed:
		return true
	default:
		return false
	}
}

func (s ProductionRunStatus) EnumValues() []string {
	return []string{
		string(ProductionRunStatusOpen),
		string(ProductionRunStatusClosed),
	}
}
