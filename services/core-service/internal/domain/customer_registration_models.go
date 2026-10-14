package domain

import "github.com/open-mrp/api/shared/constants"

// CustomerRegistrationAddressParams holds the address data needed for customer registration.
type CustomerRegistrationAddressParams struct {
	Name        *string
	StreetLine1 string
	StreetLine2 *string
	Locality    string
	State       string
	PostalCode  string
	Country     string
}

type CreateNewCustomerAccountParams struct {
	AccountID       string
	CustomerName    string
	CustomerNumber  string
	CustomerGroupID string
	PaymentTermID   string
	ShippingTermID  string
	Email           string
	Phone           *string
	Address         CustomerRegistrationAddressParams
}

type RegisterCustomerParams struct {
	AccountSlug        string
	IsExistingCustomer bool
	CustomerData       CustomerRegistrationData
	Lookup             ExistingCustomerLookup
}

// ExistingCustomerLookup finds the customer an existing-customer registration joins: a number of the given type, confirmed by the ZIP on the customer's default billing or shipping address.
type ExistingCustomerLookup struct {
	Type       constants.PortalRegistrationLookupType
	Number     string
	PostalCode string
}

type CustomerRegistrationData struct {
	Number          *string
	Name            *string
	CustomerGroupID *string
	Phone           *string
	Address         *CustomerRegistrationAddress
	ShippingTermID  *string
	PaymentTermID   *string
}

type CustomerRegistrationAddress struct {
	StreetLine1 string
	StreetLine2 *string
	Locality    string
	State       string
	PostalCode  string
	Country     string
	Name        *string
}

// CustomerMatch names a customer of the seller.
type CustomerMatch struct {
	AccountID string
	Name      string
	Number    string
}

// CustomerDefaultAddresses is a customer with the street and postal code of its default billing and shipping addresses.
type CustomerDefaultAddresses struct {
	CustomerMatch
	BillingStreetLine1  *string
	BillingPostalCode   *string
	ShippingStreetLine1 *string
	ShippingPostalCode  *string
}
