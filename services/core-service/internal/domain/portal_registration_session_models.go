package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// PortalRegistrationSessionData is the scratch form data accumulated across the buyer registration steps, persisted as JSON on the session so a resumed session restores exactly where the buyer left off.
type PortalRegistrationSessionData struct {
	CustomerName      string `json:"customer_name,omitempty"`
	CustomerNumber    string `json:"customer_number,omitempty"`
	CustomerGroupID   string `json:"customer_group_id,omitempty"`
	PaymentTermID     string `json:"payment_term_id,omitempty"`
	ShippingTermID    string `json:"shipping_term_id,omitempty"`
	Phone             string `json:"phone,omitempty"`
	AddressName       string `json:"address_name,omitempty"`
	AddressStreet1    string `json:"address_street_1,omitempty"`
	AddressStreet2    string `json:"address_street_2,omitempty"`
	AddressLocality   string `json:"address_locality,omitempty"`
	AddressState      string `json:"address_state,omitempty"`
	AddressPostalCode string `json:"address_postal_code,omitempty"`
	AddressCountry    string `json:"address_country,omitempty"`
}

// PortalRegistrationSession is a buyer's in-progress (or completed/abandoned) registration into a specific seller's customer portal.
type PortalRegistrationSession struct {
	ID                 string
	UserID             string
	SellerAccountID    string
	SellerSlug         string
	IsExistingCustomer *bool
	Step               constants.PortalRegistrationStep
	CustomerID         *string
	SessionData        PortalRegistrationSessionData
	CompletedAt        *time.Time
	AbandonedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CreatePortalRegistrationSessionParams holds the inputs to start a session.
type CreatePortalRegistrationSessionParams struct {
	UserID             string
	SellerAccountID    string
	SellerSlug         string
	IsExistingCustomer *bool
	Step               constants.PortalRegistrationStep
	SessionData        PortalRegistrationSessionData
}

// UpdatePortalRegistrationSessionParams holds the inputs to advance a session.
type UpdatePortalRegistrationSessionParams struct {
	TypeID             string
	Step               constants.PortalRegistrationStep
	SessionData        PortalRegistrationSessionData
	IsExistingCustomer *bool
}
