package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

// EmailBridgeSvc owns the self-serve email-domain verification flow and inbox CRUD that back the chat↔email bridge. All methods are account-scoped via the caller identity.
type EmailBridgeSvc interface {
	CreateDomain(ctx context.Context, domain string) (*EmailDomain, *apierror.APIError)
	ListDomains(ctx context.Context) ([]*EmailDomain, *apierror.APIError)
	GetDomain(ctx context.Context, id string) (*EmailDomain, *apierror.APIError)
	// VerifyDomain re-polls SES and flips the domain to verified once DKIM is confirmed.
	VerifyDomain(ctx context.Context, id string) (*EmailDomain, *apierror.APIError)
	// DeleteDomain deregisters a domain (deleting its SES identity) once it has no inboxes bound to it.
	DeleteDomain(ctx context.Context, id string) *apierror.APIError

	CreateInbox(ctx context.Context, input CreateEmailInboxInput) (*EmailInbox, *apierror.APIError)
	ListInboxes(ctx context.Context) ([]*EmailInbox, *apierror.APIError)
	GetInbox(ctx context.Context, id string) (*EmailInbox, *apierror.APIError)
	UpdateInbox(ctx context.Context, id string, input UpdateEmailInboxInput) (*EmailInbox, *apierror.APIError)
	DeleteInbox(ctx context.Context, id string) *apierror.APIError
}

// EmailIdentityProvider abstracts the SES domain-identity operations behind the self-serve verification flow: registering a customer domain (enabling DKIM, returning the CNAME tokens the customer must publish) and polling whether SES has confirmed it.
type EmailIdentityProvider interface {
	// RegisterDomain idempotently registers the domain as a sending/receiving identity and enables
	// Easy DKIM, returning the DKIM CNAME tokens the customer publishes to verify the domain.
	RegisterDomain(ctx context.Context, domain string) (dkimTokens []string, apiErr *apierror.APIError)
	// DomainVerified reports whether SES has confirmed the domain's DKIM verification.
	DomainVerified(ctx context.Context, domain string) (bool, *apierror.APIError)
	// DeleteDomain deletes the domain's SES identity. It is idempotent: deleting an already-removed
	// identity is a no-op, so a failed domain deletion can be safely retried.
	DeleteDomain(ctx context.Context, domain string) *apierror.APIError
}
