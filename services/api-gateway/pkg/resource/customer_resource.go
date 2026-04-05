package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCustomerID = "ac_01gf7a8200er3ar3pkfrb6kk29"
const SampleCustomerName = "Acme Inc."
const SampleCustomerNumber = "100042"
const SampleCustomerRelationID = "acre_01gf7a8200er3ar3pkfrb6kk30"

// Customer represents a customer account with full detail.
type Customer struct {
	// The unique identifier for the customer (account ID).
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// The display name of the customer.
	Name string `json:"name" validate:"required"`
	// The external customer number.
	Number string `json:"number" validate:"required"`
	// The customer's account status code.
	Status constants.AccountStatusCode `json:"status" validate:"required"`
	// Whether EDI is enabled for this customer.
	IsEdiEnabled bool `json:"is_edi_enabled"`
	// Whether this customer is a parent account.
	IsParentAccount bool `json:"is_parent_account"`
	// The commission status for this customer.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Notes about the customer.
	Note *string `json:"note"`
	// The customer's contact information.
	ContactInfo *CustomerContactInfo `json:"contact_info" expandable:"true"`
	// The customer's freight preferences.
	FreightPreferences *CustomerFreightPreferences `json:"freight_preferences" expandable:"true"`
	// The customer's default settings.
	Defaults *CustomerDefaults `json:"defaults" expandable:"true"`
	// The customer's notification preferences.
	NotificationPreferences *CustomerNotificationPreferences `json:"notification_preferences" expandable:"true"`
	// The customer's default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The customer's default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// The customer type group.
	Type *AccountGroup `json:"type" expandable:"true"`
	// The customer's pricing groups.
	PriceGroups *List[AccountGroup] `json:"price_groups" expandable:"true"`
	// The parent customer account, if this is a child account.
	ParentAccount *Customer `json:"parent_account" expandable:"true"`
	// The child customer accounts, if this is a parent account.
	ChildAccounts *List[Customer] `json:"child_accounts" expandable:"true"`
	// When this customer was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this customer was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// CustomerContactInfo groups the customer's contact information.
type CustomerContactInfo struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_contact_info"`
	// The customer's email address.
	Email *string `json:"email"`
	// The customer's phone number.
	Phone *string `json:"phone"`
	// The customer's website URL.
	URL *string `json:"url"`
}

// CustomerFreightPreferences groups the customer's freight and carrier settings.
type CustomerFreightPreferences struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_freight_preferences"`
	// The freight status for this customer.
	Status constants.FreightPolicy `json:"status" validate:"required"`
	// The default carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The default service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// The carrier billing type.
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// The carrier billing account number.
	BillingAccount *string `json:"billing_account"`
}

// CustomerDefaults groups the customer's default configuration.
type CustomerDefaults struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_defaults"`
	// The default payment term.
	PaymentTerm *PaymentTerm `json:"payment_term" expandable:"true"`
	// The default shipping term.
	ShippingTerm *ShippingTerm `json:"shipping_term" expandable:"true"`
	// The default priority.
	Priority *Priority `json:"priority" expandable:"true"`
	// The default sales representative.
	SalesRep *User `json:"sales_rep" expandable:"true"`
}

// CustomerNotificationPreferences groups the customer's notification settings.
type CustomerNotificationPreferences struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_notification_preferences"`
	// Whether the customer accepts invoice emails.
	AcceptsInvoiceEmails bool `json:"accepts_invoice_emails"`
}

// CustomerSummary is a lightweight customer representation for list responses.
type CustomerSummary struct {
	// The unique identifier for the customer (account ID).
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_summary"`
	// The display name of the customer.
	Name string `json:"name" validate:"required"`
	// The external customer number.
	Number string `json:"number" validate:"required"`
	// The customer's email address.
	Email *string `json:"email"`
	// The customer type group name.
	CustomerTypeGroup *string `json:"customer_type_group"`
	// The customer's account status code.
	Status constants.AccountStatusCode `json:"status" validate:"required"`
	// When this customer was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
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
	IsEdiEnabled:     false,
	IsParentAccount:  false,
	CommissionPolicy: constants.CommissionPolicyApplied,
	Note:             &sampleCustomerNote,
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
		SalesRep:     SampleUser,
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

var sampleCustomerSummaryEmail = "orders@acme.com"
var sampleCustomerTypeGroupName = "Wholesale Customers"

var SampleCustomerSummary = &CustomerSummary{
	ID:                SampleCustomerID,
	Object:            constants.ObjectTypeCustomerSummary,
	Name:              SampleCustomerName,
	Number:            SampleCustomerNumber,
	Email:             &sampleCustomerSummaryEmail,
	CustomerTypeGroup: &sampleCustomerTypeGroupName,
	Status:            constants.AccountStatusCodeNormal,
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*CustomerSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomerSummary)
}

// FrequentlyOrderedProduct represents a product frequently ordered by a customer.
type FrequentlyOrderedProduct struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=frequently_ordered_product"`
	// The item associated with this product.
	Item *Item `json:"item" validate:"required"`
	// The most commonly ordered unit.
	Unit *Unit `json:"unit"`
	// The number of times this product has been ordered.
	OrderCount int32 `json:"order_count" validate:"required"`
}

var SampleFrequentlyOrderedProduct = &FrequentlyOrderedProduct{
	Object: constants.ObjectTypeFrequentlyOrderedProduct,
	Item: &Item{
		ID:     "it_01jm4r6700e3kxb9w2nqh7g5fp",
		Object: constants.ObjectTypeItem,
		SKU:    "HB-M10X30-ZN",
	},
	Unit: &Unit{
		ID:           "un_01jm4r6700e3kxb9w2nqh7g5fp",
		Object:       constants.ObjectTypeUnit,
		Name:         "Case",
		Abbreviation: "cs",
	},
	OrderCount: 42,
}

func (*FrequentlyOrderedProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFrequentlyOrderedProduct)
}
