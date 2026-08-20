package shippo

import "encoding/json"

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

// Addresses a shipment for rating. Buying a label needs more of the recipient, which LabelAddress carries.
type ShipmentAddress struct {
	Name    string `json:"name"`
	Street1 string `json:"street1"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

// Sizes a parcel the way Shippo's API takes it: decimal strings, with the units named alongside.
type Parcel struct {
	Weight       string `json:"weight"`
	Length       string `json:"length"`
	Width        string `json:"width"`
	Height       string `json:"height"`
	MassUnit     string `json:"mass_unit"`
	DistanceUnit string `json:"distance_unit"`
}

// Asks Shippo to rate a shipment. Used both to quote a real customer's parcels and, with throwaway
// values, to discover which service levels a carrier offers.
type CreateShipmentRequest struct {
	AddressFrom     ShipmentAddress `json:"address_from"`
	AddressTo       ShipmentAddress `json:"address_to"`
	Parcels         []Parcel        `json:"parcels"`
	CarrierAccounts []string        `json:"carrier_accounts"`
	Extra           *ShipmentExtra  `json:"extra,omitempty"`
}

// ShipmentExtra carries optional Shippo shipment options; only third-party billing is used here.
type ShipmentExtra struct {
	Billing *ShipmentBilling `json:"billing,omitempty"`
}

// ShipmentBilling is the Shippo third-party freight-billing block.
type ShipmentBilling struct {
	Type    string `json:"type"`
	Account string `json:"account,omitempty"`
	Country string `json:"country,omitempty"`
	Zip     string `json:"zip,omitempty"`
}

// Carries a full address for a label purchase; unlike rating, the carrier prints these fields.
type LabelAddress struct {
	Name    string `json:"name"`
	Company string `json:"company,omitempty"`
	Street1 string `json:"street1"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
	// Sent explicitly so the carrier does not guess residential and surcharge.
	IsResidential bool `json:"is_residential"`
}

// Holds the shipment inlined into an instant-label transaction.
type LabelShipment struct {
	AddressFrom LabelAddress   `json:"address_from"`
	AddressTo   LabelAddress   `json:"address_to"`
	Parcels     []Parcel       `json:"parcels"`
	Extra       *ShipmentExtra `json:"extra,omitempty"`
}

// Buys labels in one call by inlining the shipment rather than rating first.
type CreateTransactionRequest struct {
	CarrierAccount    string        `json:"carrier_account"`
	ServiceLevelToken string        `json:"servicelevel_token"`
	Shipment          LabelShipment `json:"shipment"`
	LabelFileType     string        `json:"label_file_type"`
	Async             bool          `json:"async"`
}

// Represents a Shippo transaction — one purchased label, or the master transaction.
type TransactionResponse struct {
	ObjectID       string `json:"object_id"`
	Status         string `json:"status"`
	TrackingNumber string `json:"tracking_number"`
	LabelURL       string `json:"label_url"`
	// Arrives as either the rate's object id string or the embedded rate object, depending on the call.
	Rate     json.RawMessage      `json:"rate"`
	Messages []TransactionMessage `json:"messages"`
}

// Carries a carrier or Shippo explanation for a non-SUCCESS transaction.
type TransactionMessage struct {
	Text string `json:"text"`
}

// Holds the rate object embedded in a transaction.
type TransactionRate struct {
	ObjectID string `json:"object_id"`
	Amount   string `json:"amount"`
}

// Wraps the transactions listed for one rate.
type TransactionListResponse struct {
	Results []TransactionResponse `json:"results"`
}

// Requests a refund of a purchased label transaction.
type CreateRefundRequest struct {
	Transaction string `json:"transaction"`
	Async       bool   `json:"async"`
}

// Reports the outcome of a refund request.
type RefundResponse struct {
	Status string `json:"status"`
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
