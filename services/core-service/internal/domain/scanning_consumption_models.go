package domain

// ScanningConsumption represents a single consumption demand entry for a scanning station.
type ScanningConsumption struct {
	SKU              string
	DemandMeasure    string
	DemandUnit       string
	InventoryMeasure string
	InventoryUnit    string
	Instructions     *string
}
