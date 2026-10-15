package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleCustomerID = "ac_opnlh43ymyee"
const SampleCustomerName = "Acme Inc."
const SampleCustomerNumber = "100042"
const SampleCustomerRelationID = "acre_f9nhgnzecfjm"

// A business you sell to: its contact details, the defaults new orders take, and its credit and freight policies.
type Customer struct {
	// Customer ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// Whether the customer can receive new orders.
	//
	// `archived` customers stay on their existing records.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// The customer's credit standing.
	//
	// `normal`: no restrictions. `hold_shipment`: hold their shipments; orders can still be placed. `hold_all`: hold all activity. Holds flag orders but never reject them.
	Standing constants.CustomerStanding `json:"standing" validate:"required"`
	// The customer's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required"`
	// Customer number, unique within your account.
	Number string `json:"number" validate:"required"`
	// Where the customer sits in an account hierarchy.
	//
	// `standalone`, `parent` (has `child_customers`) or `child` (has a `parent_customer`). Read-only.
	RelationshipType constants.CustomerRelationshipType `json:"relationship_type" validate:"required"`
	// Your team's standing instructions for this customer's orders, such as who receives invoices.
	//
	// Copied onto new orders that don't set their own. Always `null` for customer and supplier portal users.
	Instructions *string `json:"instructions" sensitive:"internal"`
	// Whether orders from this customer earn sales commission.
	//
	// `applied` or `exempt`. A customer is exempt if it or its group is `exempt`. Exempt customers never have a sales rep assigned automatically. Always `null` for portal users.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy" validate:"required" sensitive:"internal"`
	// Whether this customer is billed for freight.
	//
	// Freight is waived if this, the customer's group, or the product line of an ordered item is `free`.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Credit extended to this customer.
	//
	// Orders are flagged once the outstanding balance passes it, never rejected. `null` means no limit.
	CreditLimit *Money `json:"credit_limit"`
	// How to reach the customer's business.
	Contact *CustomerContact `json:"contact" validate:"required"`
	// Who pays the carrier.
	//
	// `sender`: you pay. `third_party`: the carrier bills `account_number`.
	CarrierBilling *CustomerCarrierBilling `json:"carrier_billing" validate:"required"`
	// Values copied onto a new order for this customer when the order doesn't set them.
	Defaults *CustomerDefaults `json:"defaults" validate:"required"`
	// The account group that classifies this customer, such as `Distributors`. `null` unless included.
	Group *AccountGroup `json:"group" expandable:"true"`
	// Pricing account groups this customer belongs to. `null` unless included.
	PriceGroups *List[AccountGroup] `json:"price_groups" expandable:"true" visibility:"internal"`
	// The customer this one belongs to, when it is a child customer. `null` unless included.
	ParentCustomer *Customer `json:"parent_customer" expandable:"true"`
	// The customers that belong to this one. `null` unless included.
	ChildCustomers *List[Customer] `json:"child_customers" expandable:"true"`
	// Who receives this customer's documents by email.
	//
	// Each recipient is one of the customer's users and the document types they get. New orders copy these as their `email_recipients`. `null` unless included.
	NotificationRecipients *List[EmailRecipient] `json:"notification_recipients" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// How to reach a customer's business.
type CustomerContact struct {
	// Email address.
	Email *string `json:"email"`
	// Phone number.
	Phone *string `json:"phone"`
	// Website URL.
	URL *string `json:"url"`
}

// Who pays the carrier for a customer's shipments.
type CustomerCarrierBilling struct {
	// `sender`: you pay the carrier. `third_party`: the carrier bills `account_number`.
	Type constants.CarrierBillingType `json:"type" validate:"required"`
	// The carrier account billed when `type` is `third_party`.
	AccountNumber *string `json:"account_number"`
}

// Values copied onto a new order for a customer when the order doesn't set them.
type CustomerDefaults struct {
	// Carrier for the customer's orders. `null` unless included.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level of the default carrier. `null` unless included.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Payment term for the customer's orders. `null` unless included.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping term for the customer's orders. `null` unless included.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Your account user credited as the sales rep on the customer's orders. `null` unless included.
	//
	// Used when an order is created without a sales rep, unless the customer is commission-exempt. With none set, the rep comes from the sales territory matching the order's ship-to address.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Billing address for the customer's orders. `null` unless included.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Shipping address for the customer's orders. `null` unless included.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Days the customer's dock accepts freight. `null` uses your account default.
	ReceiveCalendar *OperatingCalendar `json:"receive_calendar" expandable:"true" visibility:"internal"`
	// Business days (on your working calendar) from an order being issued to it being due to ship.
	//
	// `null` inherits from the parent customer, then the group, then your account default. Get Customer Lead Time shows the result.
	LeadTime *Quantity `json:"lead_time"`
	// Where the customer's order lines are supplied from when the item doesn't say.
	//
	// `stock`: taken from inventory. `make`: produced for the order. `buy`: purchased for the order. `null` uses the item's supply.
	Supply *constants.Supply `json:"supply"`
}

// The ship-by lead time a new order for a customer would be committed to, and where it came from.
type CustomerLeadTime struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_lead_time"`
	// The customer the lead time was resolved for. `null` unless included.
	Customer *Customer `json:"customer" expandable:"true"`
	// Business days from an order being issued to it being due to ship.
	//
	// `0` means same-day: an order issued today would be due to ship today.
	LeadTime *Quantity `json:"lead_time" validate:"required"`
	// Which rule in the chain produced the lead time.
	//
	// `customer`: set on the customer. `parent_customer`: inherited from its parent customer. `account_group`: inherited from its group. `account`: your account default.
	Source constants.LeadTimeSource `json:"source" validate:"required"`
	// The account group the lead time was inherited from, only when `source` is `account_group`. `null` unless included.
	AccountGroup *AccountGroup `json:"account_group" expandable:"true"`
	// The parent customer the lead time was inherited from, only when `source` is `parent_customer`. `null` unless included.
	ParentCustomer *Customer `json:"parent_customer" expandable:"true"`
}

// SampleDayUnit is the day unit every lead time is counted in.
var SampleDayUnit = &Unit{
	ID:                "day",
	Object:            constants.ObjectTypeUnit,
	Name:              "Day",
	Abbreviation:      "d",
	Dimension:         constants.UnitDimensionTime,
	RatioNumerator:    "24",
	RatioDenominator:  "1",
	OffsetNumerator:   "0",
	OffsetDenominator: "1",
	Owner:             SampleOwnerSystem,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleCustomerLeadTime = &CustomerLeadTime{
	Object:   constants.ObjectTypeCustomerLeadTime,
	LeadTime: &Quantity{Value: "10", Unit: SampleDayUnit},
	Source:   constants.LeadTimeSourceAccount,
}

func (*CustomerLeadTime) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomerLeadTime)
}

// --- Sample data ---

var sampleCustomerEmail = "orders@acme.com"
var sampleCustomerPhone = "555-123-4567"
var sampleCustomerURL = "https://acme.com"
var sampleCustomerInstructions = "Send invoices to accounts payable."
var sampleCarrierBillingAccount = "123456789"
var sampleCustomerSupply = constants.SupplyStock

var SampleCustomer = &Customer{
	ID:               SampleCustomerID,
	Object:           constants.ObjectTypeCustomer,
	Status:           constants.ArchiveStatusActive,
	Standing:         constants.CustomerStandingNormal,
	Name:             SampleCustomerName,
	Number:           SampleCustomerNumber,
	RelationshipType: constants.CustomerRelationshipTypeStandalone,
	Instructions:     &sampleCustomerInstructions,
	CommissionPolicy: new(constants.CommissionPolicyApplied),
	FreightPolicy:    constants.FreightPolicyBilled,
	CreditLimit:      SampleMoney,
	Contact: &CustomerContact{
		Email: &sampleCustomerEmail,
		Phone: &sampleCustomerPhone,
		URL:   &sampleCustomerURL,
	},
	CarrierBilling: &CustomerCarrierBilling{
		Type:          constants.CarrierBillingTypeThirdParty,
		AccountNumber: &sampleCarrierBillingAccount,
	},
	Defaults: &CustomerDefaults{
		Carrier:       SampleCarrier,
		ServiceLevel:  SampleServiceLevel,
		PaymentTerm:   SamplePaymentTerm,
		ShippingTerm:  SampleShippingTerm,
		SalesRep:      SampleAccountUser,
		BillToAddress: SampleAddress,
		ShipToAddress: SampleAddress,
		LeadTime:      &Quantity{Value: "10", Unit: SampleDayUnit},
		Supply:        &sampleCustomerSupply,
	},
	Group:     SampleAccountGroup,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Customer) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomer)
}

// An item a customer orders regularly, derived from their sales order history.
type FrequentlyOrderedProduct struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=frequently_ordered_product"`
	// The item the customer ordered.
	Item *Item `json:"item" validate:"required"`
	// Number of sales order lines on which this customer ordered the item.
	OrderCount int32 `json:"order_count" validate:"required"`
}

var SampleFrequentlyOrderedProduct = &FrequentlyOrderedProduct{
	Object:     constants.ObjectTypeFrequentlyOrderedProduct,
	Item:       SampleItem,
	OrderCount: 42,
}

func (*FrequentlyOrderedProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFrequentlyOrderedProduct)
}
