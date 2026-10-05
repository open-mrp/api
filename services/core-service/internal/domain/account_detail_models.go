package domain

import (
	"time"

	"github.com/open-mrp/api/shared/field"
)

// Account represents the full account with branding and portal sub-resources.
//
// Branding and Portal carry no audit tag of their own: a tagged struct field is recorded as one change
// holding the whole struct, so the account update diffs their tagged fields one by one instead.
type Account struct {
	ID                       string
	Name                     string  `audit:"name"`
	DefaultBillingAddressID  *string `audit:"default_billing_address_id"`
	DefaultShippingAddressID *string `audit:"default_shipping_address_id"`
	Branding                 *AccountBranding
	Portal                   *AccountPortal
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// AccountBranding holds the branding metadata for an account.
type AccountBranding struct {
	ID              string
	SupportEmail    *string `audit:"support_email"`
	PhoneNumber     *string `audit:"phone_number"`
	LogoURL         *string `audit:"logo_url"`
	FaviconURL      *string `audit:"favicon_url"`
	FacebookHandle  *string `audit:"facebook_handle"`
	InstagramHandle *string `audit:"instagram_handle"`
	LinkedInHandle  *string `audit:"linkedin_handle"`
	TwitterHandle   *string `audit:"twitter_handle"`
	WebsiteURL      *string `audit:"website_url"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AccountPortal holds the portal metadata for an account.
type AccountPortal struct {
	ID        string
	Slug      string `audit:"slug"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PublicAccountBySlug is a minimal account representation returned for unauthenticated slug lookups.
type PublicAccountBySlug struct {
	ID                      string
	Name                    string
	Slug                    string
	DefaultBillingAddressID *string
	SupportEmail            *string
	LogoURL                 *string
	FaviconURL              *string
	// PortalDomain is the account's verified custom portal domain (e.g. shop.acme.com), when one exists.
	PortalDomain *string
}

// PortalProfile is the authenticated seller portal profile: identity plus the seller's public letterhead address. Served to logged-in customer-portal pages, unlike the minimal, unauthenticated PublicAccountBySlug.
type PortalProfile struct {
	ID           string
	Name         string
	Slug         string
	LogoURL      *string
	FaviconURL   *string
	SupportEmail *string
	Address      *Address
}

// UpdateAccountParams holds the optional fields for updating an account. A cleared branding field is
// removed.
type UpdateAccountParams struct {
	AccountID       string
	Name            *string
	SupportEmail    field.Clearable[string]
	PhoneNumber     field.Clearable[string]
	Slug            *string
	WebsiteURL      field.Clearable[string]
	FacebookHandle  field.Clearable[string]
	InstagramHandle field.Clearable[string]
	LinkedInHandle  field.Clearable[string]
	TwitterHandle   field.Clearable[string]
	// DefaultBillingAddressID and DefaultShippingAddressID must name addresses linked to the account.
	DefaultBillingAddressID  *string
	DefaultShippingAddressID *string
}

// HasBrandingUpdates returns true if any branding field is set or cleared.
func (p *UpdateAccountParams) HasBrandingUpdates() bool {
	return p.SupportEmail.WasProvided() || p.PhoneNumber.WasProvided() || p.WebsiteURL.WasProvided() ||
		p.FacebookHandle.WasProvided() || p.InstagramHandle.WasProvided() ||
		p.LinkedInHandle.WasProvided() || p.TwitterHandle.WasProvided()
}

// BrandingAfter is the account's branding with this update's fields applied to before.
func (p *UpdateAccountParams) BrandingAfter(before AccountBranding) AccountBranding {
	after := before
	after.SupportEmail = p.SupportEmail.StringPtrAfterBackfill(before.SupportEmail)
	after.PhoneNumber = p.PhoneNumber.StringPtrAfterBackfill(before.PhoneNumber)
	after.WebsiteURL = p.WebsiteURL.StringPtrAfterBackfill(before.WebsiteURL)
	after.FacebookHandle = p.FacebookHandle.StringPtrAfterBackfill(before.FacebookHandle)
	after.InstagramHandle = p.InstagramHandle.StringPtrAfterBackfill(before.InstagramHandle)
	after.LinkedInHandle = p.LinkedInHandle.StringPtrAfterBackfill(before.LinkedInHandle)
	after.TwitterHandle = p.TwitterHandle.StringPtrAfterBackfill(before.TwitterHandle)
	return after
}
