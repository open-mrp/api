package shippo

// CarrierAccount represents a carrier account from the Shippo API.
type CarrierAccount struct {
	ObjectID        string `json:"object_id"`
	Carrier         string `json:"carrier"`
	AccountID       string `json:"account_id"`
	Active          bool   `json:"active"`
	IsShippoAccount bool   `json:"is_shippo_account"`
}

// CarrierAccountListResponse is the list response from Shippo.
type CarrierAccountListResponse struct {
	Results []CarrierAccount `json:"results"`
}

// ServiceLevel represents a shipping service level.
type ServiceLevel struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// ShipmentRate represents a rate from a Shippo shipment response.
type ShipmentRate struct {
	Amount        string       `json:"amount"`
	ServiceLevel  *RateService `json:"servicelevel"`
	EstimatedDays *int32       `json:"estimated_days"`
	Attributes    []string     `json:"attributes"`
}

// RateService is the service level metadata within a rate.
type RateService struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// ShipmentResponse is a Shippo shipment creation response.
type ShipmentResponse struct {
	Rates []ShipmentRate `json:"rates"`
}

// CreateCarrierAccountRequest is the request to create a carrier account.
type CreateCarrierAccountRequest struct {
	Carrier    string            `json:"carrier"`
	AccountID  string            `json:"account_id"`
	Active     bool              `json:"active"`
	Parameters map[string]string `json:"parameters"`
}

// UpdateCarrierAccountRequest is the request to update a carrier account.
type UpdateCarrierAccountRequest struct {
	Carrier    string            `json:"carrier"`
	AccountID  string            `json:"account_id"`
	Active     bool              `json:"active"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// TestAddress is a simple address for test shipments.
type TestAddress struct {
	Name    string `json:"name"`
	Street1 string `json:"street1"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

// TestParcel is a parcel for test shipments.
type TestParcel struct {
	Weight       string `json:"weight"`
	Length       string `json:"length"`
	Width        string `json:"width"`
	Height       string `json:"height"`
	MassUnit     string `json:"mass_unit"`
	DistanceUnit string `json:"distance_unit"`
}

// CreateShipmentRequest is used to create a test shipment to discover service levels.
type CreateShipmentRequest struct {
	AddressFrom     TestAddress  `json:"address_from"`
	AddressTo       TestAddress  `json:"address_to"`
	Parcels         []TestParcel `json:"parcels"`
	CarrierAccounts []string     `json:"carrier_accounts"`
}

// OAuthInitiateResponse represents the Shippo OAuth redirect response.
type OAuthInitiateResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// APIError represents a Shippo API error response body.
type APIError struct {
	Detail         string   `json:"detail"`
	Message        string   `json:"message"`
	Error          string   `json:"error"`
	NonFieldErrors []string `json:"non_field_errors"`
}
