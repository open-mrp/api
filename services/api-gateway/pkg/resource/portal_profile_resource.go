package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// PortalProfile is the authenticated seller portal profile served to logged-in customer-portal pages: the seller's identity plus its public letterhead address. Unlike PublicAccount (public, minimal, for pre-login pages), this requires authentication and includes the address inline as a plain field.
type PortalProfile struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_profile"`
	// The seller's display name.
	Name string `json:"name" validate:"required"`
	// Portal slug.
	Slug string `json:"slug" validate:"required"`
	// Logo URL.
	LogoURL *string `json:"logo_url"`
	// Customer-portal favicon URL.
	FaviconURL *string `json:"favicon_url"`
	// Support email address.
	SupportEmail *string `json:"support_email"`
	// The seller's letterhead address (its default billing address), or null when the account has none.
	Address *Address `json:"address"`
}

var samplePortalProfile = &PortalProfile{
	ID:           SampleAccountID,
	Object:       constants.ObjectTypePortalProfile,
	Name:         "Acme Inc",
	Slug:         SampleAccountPortalSlug,
	LogoURL:      nil,
	SupportEmail: nil,
	Address:      SampleAddress,
}

func (*PortalProfile) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePortalProfile)
}
