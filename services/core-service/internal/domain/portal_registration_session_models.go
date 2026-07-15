package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
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

// DeriveStatus computes the session's lifecycle status as of now: completed/abandoned take precedence over the derived in-progress-vs-expired split (an incomplete session past its resume TTL reads as expired). now is passed in so callers control the clock (testability).
func (s *PortalRegistrationSession) DeriveStatus(now time.Time) constants.PortalRegistrationStatus {
	switch {
	case s.CompletedAt != nil:
		return constants.PortalRegistrationStatusCompleted
	case s.AbandonedAt != nil:
		return constants.PortalRegistrationStatusAbandoned
	case now.Sub(s.CreatedAt) >= constants.PortalRegistrationSessionTTL:
		return constants.PortalRegistrationStatusExpired
	default:
		return constants.PortalRegistrationStatusInProgress
	}
}

// ListPortalRegistrationSessionsParams lists a seller's buyer-registration sessions for the customer-service follow-up view.
type ListPortalRegistrationSessionsParams struct {
	// SellerAccountID scopes the list to one seller account.
	SellerAccountID string
	// Cursor / Limit drive keyset pagination.
	Cursor *string
	Limit  int32
	// StatusFilter, when set, restricts to one derived status (in_progress | completed | abandoned | expired).
	StatusFilter *string
	// SearchTerm, when set, matches the registrant's captured customer name/number or the session id.
	SearchTerm *string
	// ExpiryThreshold is the created-at boundary (now - TTL) below which an incomplete session counts as expired. Supplied by the service so the TTL is single-sourced.
	ExpiryThreshold time.Time
}

// ListPortalRegistrationSessionsResult is a page of registration sessions.
type ListPortalRegistrationSessionsResult struct {
	Sessions []*PortalRegistrationSession
	PageInfo pagination.PageInfo
}
