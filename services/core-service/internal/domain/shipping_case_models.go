package domain

import "time"

// ShippingCase represents a shipping case domain model with joined fields for reads.
type ShippingCase struct {
	ID                  string
	Number              string  `audit:"number"`
	SSCC                *string `audit:"sscc"`
	TrackingNumber      *string `audit:"tracking_number"`
	ShippoTransactionID *string
	ShippingLabelURL    *string
	ShippedAt           *time.Time `audit:"shipped_at"`
	// Freight amount quantity
	FreightAmountID               string
	FreightAmountValue            string `audit:"freight_amount_value"`
	FreightAmountUnitID           string `audit:"freight_amount_unit_id"`
	FreightAmountUnitName         string
	FreightAmountUnitAbbreviation string
	FreightAmountUnitType         string
	// Freight weight quantity
	FreightWeightID               string
	FreightWeightValue            string `audit:"freight_weight_value"`
	FreightWeightUnitID           string `audit:"freight_weight_unit_id"`
	FreightWeightUnitName         string
	FreightWeightUnitAbbreviation string
	FreightWeightUnitType         string
	// Relations
	ShipmentID  string
	CarrierID   string `audit:"carrier_id"`
	CarrierName string
	AccountID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UpdateShippingCaseParams holds the parameters for updating a shipping case.
type UpdateShippingCaseParams struct {
	AccountID           string
	ShippingCaseID      string
	TrackingNumber      *string
	FreightAmountValue  *string
	FreightAmountUnitID *string
	FreightWeightValue  *string
	FreightWeightUnitID *string
}
