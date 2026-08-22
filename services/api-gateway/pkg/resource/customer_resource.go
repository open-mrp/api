package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleCustomerID = "ac_opnlh43ymyee"
const SampleCustomerName = "Acme Inc."
const SampleCustomerNumber = "100042"
const SampleCustomerRelationID = "acre_f9nhgnzecfjm"

// A business you sell to, with its contact details, default fulfillment settings, and order policies.
type Customer struct {
	// Customer ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// The customer's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required"`
	// Human-readable customer number used to identify the account, distinct from the `id`.
	//
	// Unique within your account.
	Number string `json:"number" validate:"required"`
	// The customer's account standing.
	//
	// - `normal`: standard account with no restrictions.
	// - `preferred`: account flagged for prioritized handling.
	// - `hold_shipment`: the customer's shipments should be held, typically over a credit problem, while orders can still be placed.
	// - `hold_all`: all activity for the customer should be held.
	//
	// The hold statuses are advisory: OpenMRP flags the customer's orders as being on credit hold, but requests to create orders or shipments for the customer are not rejected.
	Status constants.AccountStatusCode `json:"status" validate:"required"`
	// Whether EDI (Electronic Data Interchange) is enabled for exchanging orders and documents with this customer.
	EDIStatus constants.EDIStatus `json:"edi_status" validate:"required"`
	// The customer's position in the account hierarchy.
	//
	// - `standalone`: no parent or child accounts.
	// - `parent`: has one or more child accounts (see `child_accounts`).
	// - `child`: belongs to a parent account (see `parent_account`).
	RelationshipType constants.CustomerRelationshipType `json:"relationship_type" validate:"required"`
	// How sales commission applies to this customer's orders.
	//
	// - `commission_exempt`: this customer's orders are exempt from sales commission.
	// - `commission_applied`: sales commission is calculated on this customer's orders.
	//
	// The customer counts as exempt if this field, its `type` group, or any of its `price_groups` is `commission_exempt`. Exempt customers never have a sales rep assigned automatically when an order is created without one.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Free-form note about the customer.
	Note *string `json:"note"`
	// Maximum credit extended to this customer.
	//
	// Used to flag orders once the customer's outstanding balance approaches or passes the limit; orders that exceed it are not rejected.
	CreditLimit *Quantity `json:"credit_limit" expandable:"true"`
	// General contact details for the customer's business.
	ContactInfo *CustomerContactInfo `json:"contact_info" expandable:"true"`
	// Freight and carrier preferences applied to this customer's shipments.
	FreightPreferences *CustomerFreightPreferences `json:"freight_preferences" expandable:"true"`
	// Default settings applied to new orders for this customer.
	Defaults *CustomerDefaults `json:"defaults" expandable:"true"`
	// Which document emails this customer is set up to receive.
	NotificationPreferences *CustomerNotificationPreferences `json:"notification_preferences" expandable:"true"`
	// Default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// The account group of type `type_group` that categorizes this customer (for example "Distributors").
	Type *AccountGroup `json:"type" expandable:"true"`
	// Account groups of type `pricing_group` that this customer belongs to, used to apply pricing rules.
	PriceGroups *List[AccountGroup] `json:"price_groups" expandable:"true"`
	// The customer this account belongs to, when it is a child account.
	ParentAccount *Customer `json:"parent_account" expandable:"true"`
	// The customers belonging to this account, when it is a parent account.
	ChildAccounts *List[Customer] `json:"child_accounts" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Customer contact information.
type CustomerContactInfo struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_contact_info"`
	// Email address.
	Email *string `json:"email"`
	// Phone number.
	Phone *string `json:"phone"`
	// Website URL.
	URL *string `json:"url"`
}

// Customer freight and carrier settings.
type CustomerFreightPreferences struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_freight_preferences"`
	// Freight policy applied to this customer's orders.
	//
	// - `free_freight`: the customer is not billed for freight.
	// - `billed_freight`: freight is billed to the customer.
	//
	// Freight is waived when this field, the customer's `type` group, any of its `price_groups`, or any product line the ordered products belong to is `free_freight`, so a shipment can come back freight-exempt even while this field is `billed_freight`.
	Status constants.FreightPolicy `json:"status" validate:"required"`
	// Carrier used on this customer's orders when the order does not specify one.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level used when an order takes its carrier from this customer's default carrier.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Who pays the carrier for shipments.
	//
	// - `sender`: the shipper (you) pays the carrier.
	// - `third_party`: a third party is billed, using `billing_account`.
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// Carrier billing account number charged when `billing_type` is `third_party`.
	BillingAccount *string `json:"billing_account"`
}

// Values used to fill in a new sales order for this customer when the order does not supply its own.
type CustomerDefaults struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_defaults"`
	// Payment term used on this customer's orders when the order does not specify one.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Shipping term used on this customer's orders when the order does not specify one.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Priority used to pre-fill new orders for this customer.
	Priority *Priority `json:"priority" expandable:"true"`
	// Account user credited as the sales rep on this customer's orders.
	//
	// Used when an order is created without a sales rep, unless the customer is commission-exempt. With no default set, the rep is resolved from the sales territory matching the order's ship-to postal code or state.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
	// Calendar days between an order being issued and it being due to ship.
	//
	// Sets each order's `ship_by_date` when it is issued. With none set here the customer inherits its parent account's lead time, then its account group's, then the account default.
	LeadTimeDays *int32 `json:"lead_time_days"`
	// The operating calendar naming the days this customer's dock accepts freight.
	//
	// A promised delivery date is worked back from a day the customer can actually receive on. With none set here the customer inherits its account group's calendar, then the account default, then Monday to Friday.
	ReceiveCalendarID *string `json:"receive_calendar_id"`
}

// The ship-by lead time a new order for this customer would be committed to.
type CustomerLeadTime struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_lead_time"`
	// The customer the lead time was resolved for.
	Customer *Entity `json:"customer" validate:"required"`
	// Calendar days between an order being issued and it being due to ship.
	//
	// `0` means same-day: an order issued today would be due to ship today.
	Days int32 `json:"days"`
	// Which rule in the chain produced this lead time.
	//
	// - `customer`: a lead time set on the customer itself.
	// - `parent_customer`: inherited from the customer's parent account.
	// - `account_group`: inherited from the customer's account group.
	// - `account`: the account-wide fallback.
	//
	// The shared `manual` value cannot appear here: it means a promised date was set on one specific order, which is a fact about that order rather than about the customer.
	Source constants.LeadTimeSource `json:"source" validate:"required"`
	// The account group the lead time was inherited from.
	//
	// Present only when `source` is `account_group`. A customer that belongs to a group but sets its own lead time inherited nothing.
	AccountGroup *AccountGroup `json:"account_group" expandable:"true"`
	// The parent customer the lead time was inherited from.
	//
	// Present only when `source` is `parent_customer`. A customer that has a parent but sets its own lead time inherited nothing.
	ParentCustomer *Customer `json:"parent_customer" expandable:"true"`
}

var SampleCustomerLeadTime = &CustomerLeadTime{
	Object:   constants.ObjectTypeCustomerLeadTime,
	Customer: NewEntity(SampleCustomerID, constants.ObjectTypeCustomer, nil, nil),
	Days:     30,
	Source:   constants.LeadTimeSourceAccount,
}

func (*CustomerLeadTime) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomerLeadTime)
}

// Customer notification settings.
type CustomerNotificationPreferences struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_notification_preferences"`
	// Whether anyone is set up to receive invoice emails for this customer.
	//
	// Derived from the customer's notification recipients: true when at least one of them is configured for invoice notifications.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
}

// --- Sample data ---

var sampleCustomerEmail = "orders@acme.com"
var sampleCustomerPhone = "555-123-4567"
var sampleCustomerURL = "https://acme.com"
var sampleCustomerNote = "Preferred customer since 2020."
var sampleCarrierBillingType = constants.CarrierBillingTypeSender
var sampleCarrierBillingAccount = "123456789"

var SampleCustomer = &Customer{
	ID:               SampleCustomerID,
	Object:           constants.ObjectTypeCustomer,
	Name:             SampleCustomerName,
	Number:           SampleCustomerNumber,
	Status:           constants.AccountStatusCodeNormal,
	EDIStatus:        constants.EDIStatusDisabled,
	RelationshipType: constants.CustomerRelationshipTypeStandalone,
	CommissionPolicy: constants.CommissionPolicyApplied,
	Note:             &sampleCustomerNote,
	CreditLimit:      SampleQuantity,
	ContactInfo: &CustomerContactInfo{
		Object: constants.ObjectTypeCustomerContactInfo,
		Email:  &sampleCustomerEmail,
		Phone:  &sampleCustomerPhone,
		URL:    &sampleCustomerURL,
	},
	FreightPreferences: &CustomerFreightPreferences{
		Object:         constants.ObjectTypeCustomerFreightPreferences,
		Status:         constants.FreightPolicyBilled,
		Carrier:        SampleCarrier,
		ServiceLevel:   SampleServiceLevel,
		BillingType:    &sampleCarrierBillingType,
		BillingAccount: &sampleCarrierBillingAccount,
	},
	Defaults: &CustomerDefaults{
		Object:       constants.ObjectTypeCustomerDefaults,
		PaymentTerm:  SamplePaymentTerm,
		ShippingTerm: SampleShippingTerm,
		Priority:     SamplePriority,
		SalesRep:     SampleAccountUser,
	},
	NotificationPreferences: &CustomerNotificationPreferences{
		Object:               constants.ObjectTypeCustomerNotificationPreferences,
		AcceptsInvoiceEmails: true,
	},
	BillToAddress: SampleAddress,
	ShipToAddress: SampleAddress,
	Type:          SampleAccountGroup,
	PriceGroups:   NewList([]AccountGroup{*SampleAccountGroup}, PageInfo{}),
	ParentAccount: nil,
	ChildAccounts: nil,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
	// The unit of measure this customer orders the item in most often.
	Unit *Unit `json:"unit"`
	// Number of sales order lines on which this customer ordered the item in the unit shown.
	OrderCount int32 `json:"order_count" validate:"required"`
}

var SampleFrequentlyOrderedProduct = &FrequentlyOrderedProduct{
	Object:     constants.ObjectTypeFrequentlyOrderedProduct,
	Item:       SampleItem,
	Unit:       newSampleUnit(SampleUnitName, SampleUnitAbbreviation, constants.UnitTypeMass),
	OrderCount: 42,
}

func (*FrequentlyOrderedProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFrequentlyOrderedProduct)
}
