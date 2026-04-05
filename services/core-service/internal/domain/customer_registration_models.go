package domain

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
