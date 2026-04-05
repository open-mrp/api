package domain

// ExecuteProductionStepEvent is the outbox event payload for executing a production step side-effect.
type ExecuteProductionStepEvent struct {
	ProductionStepID  string  `json:"production_step_id"`
	ScanningStationID string  `json:"scanning_station_id"`
	ItemID            string  `json:"item_id"`
	BatchQuantityID   string  `json:"batch_quantity_id"`
	BatchMeasure      string  `json:"batch_measure"`
	BatchUnitID       string  `json:"batch_unit_id"`
	ResponsibleUserID *string `json:"responsible_user_id,omitempty"`
	ProducedBatchID   *string `json:"produced_batch_id,omitempty"`
	ProduceInventory  bool    `json:"produce_inventory"`
}
