package supplierep

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(
		field.Optional[SupplierContactInput]{},
		field.Optional[SupplierCarrierBillingInput]{},
		field.Optional[CreateSupplierDefaultsInput]{},
		field.Optional[UpdateSupplierDefaultsInput]{},
	)
}

// supplierIncludeFields are the include keys every supplier endpoint accepts.
var supplierIncludeFields = []string{
	"defaults.carrier", "defaults.carrier.service_levels", "defaults.service_level", "defaults.payment_term",
	"defaults.shipping_term", "defaults.bill_to_address", "defaults.ship_to_address",
}

// How to reach a supplier's business.
type SupplierContactInput struct {
	// Email address. Send `null` to clear it.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number. Send `null` to clear it.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL. Send `null` to clear it.
	URL field.Clearable[string] `json:"url,omitzero" validate:"omitempty,max=255"`
}

// Who pays the carrier for freight from a supplier.
type SupplierCarrierBillingInput struct {
	// `sender`: the supplier pays the carrier. `third_party`: the carrier bills `account_number`. Defaults to `sender`.
	Type field.Optional[constants.CarrierBillingType] `json:"type,omitzero"`
	// The carrier account billed when `type` is `third_party`. Send `null` to clear it.
	AccountNumber field.Clearable[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
}

// Values copied onto a new purchase order from the supplier when the order doesn't set them.
type CreateSupplierDefaultsInput struct {
	// ID of the carrier for freight from the supplier.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero"`
	// ID of a service level of the carrier.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero"`
	// ID of the payment term for the supplier's orders.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero"`
	// ID of the shipping term for the supplier's orders.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
}

// The supplier's defaults to change. Only fields you send are changed.
type UpdateSupplierDefaultsInput struct {
	// ID of the carrier for freight from the supplier. Send `null` to clear it.
	CarrierID field.Clearable[string] `json:"carrier_id,omitzero"`
	// ID of a service level of the carrier. Send `null` to clear it.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero"`
	// ID of the payment term for the supplier's orders. Send `null` to clear it.
	PaymentTermID field.Clearable[string] `json:"payment_term_id,omitzero"`
	// ID of the shipping term for the supplier's orders. Send `null` to clear it.
	ShippingTermID field.Clearable[string] `json:"shipping_term_id,omitzero"`
	// ID of one of the supplier's addresses to bill its orders to. Send `null` to clear it.
	BillToAddressID field.Clearable[string] `json:"bill_to_address_id,omitzero"`
	// ID of one of the supplier's addresses that its goods ship from. Send `null` to clear it.
	ShipToAddressID field.Clearable[string] `json:"ship_to_address_id,omitzero"`
}
