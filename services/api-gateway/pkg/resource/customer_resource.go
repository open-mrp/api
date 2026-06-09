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

// Customer account.
type Customer struct {
	// Customer ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Customer number.
	Number string `json:"number" validate:"required"`
	// Account status code.
	Status constants.AccountStatusCode `json:"status" validate:"required"`
	// EDI status.
	EDIStatus constants.EDIStatus `json:"edi_status" validate:"required"`
	// Customer relationship type.
	RelationshipType constants.CustomerRelationshipType `json:"relationship_type" validate:"required"`
	// Commission policy.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Note.
	Note *string `json:"note"`
	// Credit limit.
	CreditLimit *Quantity `json:"credit_limit" expandable:"true"`
	// Contact information.
	ContactInfo *CustomerContactInfo `json:"contact_info" expandable:"true"`
	// Freight preferences.
	FreightPreferences *CustomerFreightPreferences `json:"freight_preferences" expandable:"true"`
	// Default settings.
	Defaults *CustomerDefaults `json:"defaults" expandable:"true"`
	// Notification preferences.
	NotificationPreferences *CustomerNotificationPreferences `json:"notification_preferences" expandable:"true"`
	// Default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Customer type group.
	Type *AccountGroup `json:"type" expandable:"true"`
	// Pricing groups.
	PriceGroups *List[AccountGroup] `json:"price_groups" expandable:"true"`
	// Parent account. Present if this is a child account.
	ParentAccount *Customer `json:"parent_account" expandable:"true"`
	// Child accounts. Present if this is a parent account.
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
	// Freight policy.
	Status constants.FreightPolicy `json:"status" validate:"required"`
	// Default carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Default service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Carrier billing type.
	BillingType *constants.CarrierBillingType `json:"billing_type"`
	// Carrier billing account number.
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
	// Number of times ordered.
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
