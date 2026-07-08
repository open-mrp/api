package domain

import "time"

// Account represents the full account with branding and portal sub-resources.
type Account struct {
	ID                       string
	Name                     string           `audit:"name"`
	DefaultBillingAddressID  *string          `audit:"default_billing_address_id"`
	DefaultShippingAddressID *string          `audit:"default_shipping_address_id"`
	Branding                 *AccountBranding `audit:"branding"`
	Portal                   *AccountPortal   `audit:"portal"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// AccountBranding holds the branding metadata for an account.
type AccountBranding struct {
	ID              string
	SupportEmail    *string `audit:"support_email"`
	PhoneNumber     *string `audit:"phone_number"`
	LogoURL         *string `audit:"logo_url"`
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
	// PortalDomain is the account's verified custom portal domain (e.g. shop.acme.com), when one exists.
	PortalDomain *string
}

// PortalProfile is the authenticated seller portal profile: identity plus the seller's public letterhead address. Served to logged-in customer-portal pages, unlike the minimal, unauthenticated PublicAccountBySlug.
type PortalProfile struct {
	ID           string
	Name         string
	Slug         string
	LogoURL      *string
	SupportEmail *string
	Address      *Address
}

// UpdateAccountParams holds the optional fields for updating an account.
type UpdateAccountParams struct {
	AccountID       string
	Name            *string
	SupportEmail    *string
	PhoneNumber     *string
	Slug            *string
	WebsiteURL      *string
	FacebookHandle  *string
	InstagramHandle *string
	LinkedInHandle  *string
	TwitterHandle   *string
}

// HasBrandingUpdates returns true if any branding fields are set.
func (p *UpdateAccountParams) HasBrandingUpdates() bool {
	return p.SupportEmail != nil || p.PhoneNumber != nil || p.WebsiteURL != nil ||
		p.FacebookHandle != nil || p.InstagramHandle != nil ||
		p.LinkedInHandle != nil || p.TwitterHandle != nil
}
