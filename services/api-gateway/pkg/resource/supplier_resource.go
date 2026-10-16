package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleSupplierItemID = "suit_k2m9q4w7r1xa"

// A business you buy from: its contact details and the defaults your purchase orders from it take.
type Supplier struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// Whether the supplier can be named on new records.
	//
	// `archived` suppliers stay on their existing purchase orders and supplier items.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// The supplier's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required"`
	// Supplier number, unique within your account.
	Number string `json:"number" validate:"required"`
	// Your team's standing instructions for this supplier's orders, such as who confirms deliveries.
	//
	// Copied onto new orders that don't set their own. Always `null` for customer and supplier portal users.
	Instructions *string `json:"instructions" sensitive:"internal"`
	// How to reach the supplier's business.
	Contact *SupplierContact `json:"contact" validate:"required"`
	// Who pays the carrier for freight from this supplier.
	//
	// `sender`: the supplier pays. `third_party`: the carrier bills `account_number`.
	CarrierBilling *SupplierCarrierBilling `json:"carrier_billing" validate:"required"`
	// Values copied onto a new purchase order from this supplier when the order doesn't set them.
	Defaults *SupplierDefaults `json:"defaults" validate:"required"`
	// How many supplier items the supplier has. Only 1.0.forge-preview.9 callers read it.
	MaterialCount *int64 `json:"material_count" visibility:"internal"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// How to reach a supplier's business.
type SupplierContact struct {
	// Email address.
	Email *string `json:"email"`
	// Phone number.
	Phone *string `json:"phone"`
	// Website URL.
	URL *string `json:"url"`
}

// Who pays the carrier for freight from a supplier.
type SupplierCarrierBilling struct {
	// `sender`: the supplier pays the carrier. `third_party`: the carrier bills `account_number`.
	Type constants.CarrierBillingType `json:"type" validate:"required"`
	// The carrier account billed when `type` is `third_party`.
	AccountNumber *string `json:"account_number"`
}

// Values copied onto a new purchase order from a supplier when the order doesn't set them.
type SupplierDefaults struct {
	// Carrier for freight from the supplier. `null` unless included.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level of the default carrier. `null` unless included.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Payment term for the supplier's orders. `null` unless included.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping term for the supplier's orders. `null` unless included.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// The supplier's billing address. `null` unless included.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The address the supplier ships from. `null` unless included.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
}

var sampleSupplierEmail = "orders@acmesupplies.com"
var sampleSupplierPhone = "555-987-6543"
var sampleSupplierURL = "https://acmesupplies.com"
var sampleSupplierInstructions = "Confirm every delivery date by email."

var SampleSupplier = &Supplier{
	ID:           SampleSupplierID,
	Object:       constants.ObjectTypeSupplier,
	Status:       constants.ArchiveStatusActive,
	Name:         SampleSupplierName,
	Number:       SampleSupplierNumber,
	Instructions: &sampleSupplierInstructions,
	Contact: &SupplierContact{
		Email: &sampleSupplierEmail,
		Phone: &sampleSupplierPhone,
		URL:   &sampleSupplierURL,
	},
	CarrierBilling: &SupplierCarrierBilling{Type: constants.CarrierBillingTypeSender},
	Defaults: &SupplierDefaults{
		Carrier:       SampleCarrier,
		ServiceLevel:  SampleServiceLevel,
		PaymentTerm:   SamplePaymentTerm,
		ShippingTerm:  SampleShippingTerm,
		BillToAddress: SampleAddress,
		ShipToAddress: SampleAddress,
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Supplier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplier)
}

// An item a supplier provides, with the supplier's own code and description for it.
//
// A supplier has at most one supplier item per item.
type SupplierItem struct {
	// Supplier item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_item"`
	// Whether you still buy this item from the supplier.
	//
	// `archived` supplier items stay listed for reference.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// The supplier that provides the item. `null` unless included.
	Supplier *Supplier `json:"supplier" expandable:"true"`
	// The item the supplier provides. `null` unless included.
	Item *Item `json:"item" expandable:"true"`
	// The supplier's own code for this item, as it appears on their documents.
	SupplierSKU string `json:"supplier_sku" validate:"required"`
	// The supplier's own description of this item.
	SupplierDescription *string `json:"supplier_description"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleSupplierItemDescription = "Cold-rolled steel sheet, 16 gauge, 48x96 in."

var SampleSupplierItem = &SupplierItem{
	ID:                  SampleSupplierItemID,
	Object:              constants.ObjectTypeSupplierItem,
	Status:              constants.ArchiveStatusActive,
	Supplier:            SampleSupplier,
	Item:                SampleItem,
	SupplierSKU:         "SUP-PART-001",
	SupplierDescription: &sampleSupplierItemDescription,
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupplierItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierItem)
}
