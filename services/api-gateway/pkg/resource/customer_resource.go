package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCustomerID = "ac_0170df1ac58e4d24c66fc89f5f"
const SampleCustomerName = "Acme Inc."
const SampleCustomerNumber = "100042"
const SampleCustomerRelationID = "acre_0153f41078e241b7487172c749"

// A business you sell to, with its contact details, default fulfillment settings, and order policies.
type Customer struct {
	// Customer ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// The customer's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required"`
	// Human-readable customer number used to identify the account, distinct from the `id`.
	Number string `json:"number" validate:"required"`
	// Account status code, controlling whether the customer can transact.
	//
	// - `normal`: standard active account with no restrictions.
	// - `preferred`: active account flagged as preferred.
	// - `hold_shipment`: orders can be placed, but shipments are held.
	// - `hold_all`: all activity is on hold.
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
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Free-form note about the customer.
	Note *string `json:"note"`
	// Maximum credit extended to this customer.
	CreditLimit *Quantity `json:"credit_limit" expandable:"true"`
	// Contact information.
	ContactInfo *CustomerContactInfo `json:"contact_info" expandable:"true"`
	// Freight and carrier preferences applied to this customer's shipments.
	FreightPreferences *CustomerFreightPreferences `json:"freight_preferences" expandable:"true"`
	// Default settings applied to new orders for this customer.
	Defaults *CustomerDefaults `json:"defaults" expandable:"true"`
	// Notification preferences.
	NotificationPreferences *CustomerNotificationPreferences `json:"notification_preferences" expandable:"true"`
	// Default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// The account group of type `type_group` that categorizes this customer (for example "Distributors").
	Type *AccountGroup `json:"type" expandable:"true"`
	// Account groups of type `pricing_group` that this customer belongs to, used to apply pricing rules.
	PriceGroups *List[AccountGroup] `json:"price_groups" expandable:"true"`
	// Parent account.
	//
	// Present if this is a child account.
	ParentAccount *Customer `json:"parent_account" expandable:"true"`
	// Child accounts.
	//
	// Present if this is a parent account.
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
	// - `billed_freight`: freight is billed to the customer, unless overridden elsewhere.
	Status constants.FreightPolicy `json:"status" validate:"required"`
	// Default carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Default service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Who pays the carrier for shipments.
	//
	// - `sender`: the shipper (you) pays the carrier.
	// - `third_party`: a third party is billed, using `billing_account`.
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// Carrier billing account number charged when `billing_type` is `third_party`.
	BillingAccount *string `json:"billing_account"`
}

// Customer default configuration.
type CustomerDefaults struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_defaults"`
	// Default payment term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// Default shipping term.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// Default priority.
	Priority *Priority `json:"priority" expandable:"true"`
	// Default sales rep.
	SalesRep *AccountUser `json:"sales_rep" expandable:"true"`
}

// Customer notification settings.
type CustomerNotificationPreferences struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_notification_preferences"`
	// Whether invoice emails are accepted.
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

// Product frequently ordered by a customer.
type FrequentlyOrderedProduct struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=frequently_ordered_product"`
	// Associated item.
	Item *Item `json:"item" validate:"required"`
	// Most commonly ordered unit.
	Unit *Unit `json:"unit"`
	// Number of times the customer has ordered this item in the `unit` shown.
	OrderCount int32 `json:"order_count" validate:"required"`
}

var SampleFrequentlyOrderedProduct = &FrequentlyOrderedProduct{
	Object: constants.ObjectTypeFrequentlyOrderedProduct,
	Item: &Item{
		ID:     SampleItemID,
		Object: constants.ObjectTypeItem,
		SKU:    SampleItemSKU,
	},
	Unit:       newSampleUnit(SampleUnitName, SampleUnitAbbreviation, constants.UnitTypeMass),
	OrderCount: 42,
}

func (*FrequentlyOrderedProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFrequentlyOrderedProduct)
}
