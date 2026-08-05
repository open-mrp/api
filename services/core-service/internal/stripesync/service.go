// Package stripesync reconciles Augno customers with their counterparts in a merchant's connected Stripe account.
//
// Every operation is idempotent on replay: the link lives on the account relation, so a redelivered message updates the existing Stripe customer rather than creating a second one.
//
// Checkout deliberately does not route through here. It resolves-or-creates the same link inline because it cannot wait for the queue and, unlike this path, always has a recipient email in hand — so it creates a Stripe customer for a customer record with no email of its own, where this path correctly no-ops. Both write the link the same way, so whichever runs first wins and the other finds it already there.
package stripesync

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var tracer = tracing.GetTracer("core-service.stripe_sync")

// Service syncs Augno customers to a connected Stripe account.
type Service interface {
	// SyncCustomer creates the customer's Stripe counterpart on first sync, or pushes a changed email/name/number onto the existing one. It is a no-op (returns nil) when the account has no active Stripe integration, and when an unlinked customer has no email to create one with.
	SyncCustomer(ctx context.Context, ownerAccountID, customerAccountID string) *apierror.APIError
}

type service struct {
	repos         domain.RepoFactory
	clientFactory domain.StripeCheckoutClientFactory
	encryptionKey []byte
}

func NewService(repos domain.RepoFactory, clientFactory domain.StripeCheckoutClientFactory, encryptionKey []byte) Service {
	return &service{
		repos:         repos,
		clientFactory: clientFactory,
		encryptionKey: encryptionKey,
	}
}

func (s *service) SyncCustomer(ctx context.Context, ownerAccountID, customerAccountID string) *apierror.APIError {
	ctx, span := tracer.Start(ctx, "stripesync.sync_customer")
	defer span.End()

	client, connected, apiErr := s.clientForAccount(ctx, ownerAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !connected {
		return nil
	}

	customerRepo := s.repos.NewCustomerRepo()

	// Re-read at handling time rather than trusting a snapshot from publish time: several
	// edits in quick succession each enqueue a command, and they can be handled out of
	// order. Reading now means whichever message is handled last writes the current row.
	customer, apiErr := customerRepo.Get(ctx, ownerAccountID, customerAccountID, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	stripeCustomerID, syncedEmail, apiErr := customerRepo.GetStripeCustomerID(ctx, ownerAccountID, customerAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	email := ""
	if customer.Email != nil {
		email = *customer.Email
	}

	if stripeCustomerID == nil || *stripeCustomerID == "" {
		// Stripe requires an email to be worth creating a customer for — receipts and
		// the hosted checkout page are addressed to it. A customer with no email yet is
		// not an error: the next update that adds one enqueues another sync.
		if email == "" {
			return nil
		}

		created, apiErr := client.CreateStripeCustomer(ctx, domain.CreateStripeCustomerParams{
			Email:      email,
			Name:       customer.Name,
			Number:     customer.Number,
			CustomerID: customerAccountID,
		})
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if apiErr := customerRepo.SetStripeCustomerID(ctx, ownerAccountID, customerAccountID, created.ID, email); apiErr != nil {
			// The Stripe customer exists but is unlinked. Returning the error redelivers the
			// message, and the retry creates a second Stripe customer — the alternative,
			// swallowing it, leaves the link permanently broken and every later sync creating
			// yet another. Duplicates are visible and mergeable in Stripe; a lost link is not.
			return tracing.Trace(span, apiErr)
		}

		return nil
	}

	// Name and number are not tracked on the relation, so there is nothing to compare
	// them against — only the email is. Skipping on an unchanged email would therefore
	// silently drop renames, so the update goes out whenever anything could have moved.
	// Stripe treats a no-op update as a success, which keeps replays cheap and harmless.
	updateParams := domain.UpdateStripeCustomerParams{
		StripeCustomerID: *stripeCustomerID,
		Name:             &customer.Name,
		Number:           &customer.Number,
	}
	// An email is never cleared on Stripe: the customer may have live subscriptions or
	// receipts pointing at it, and Augno clearing the branding email is not a request to
	// strip Stripe's copy.
	if email != "" {
		updateParams.Email = &email
	}

	if apiErr := client.UpdateStripeCustomer(ctx, updateParams); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Keep the mirrored email in step with what Stripe now holds, so checkout and any
	// later diffing read the synced value rather than the one from first link.
	if email != "" && (syncedEmail == nil || *syncedEmail != email) {
		if apiErr := customerRepo.SetStripeCustomerID(ctx, ownerAccountID, customerAccountID, *stripeCustomerID, email); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

// clientForAccount returns a Stripe client for the account, reporting connected=false when no integration exists or it is inactive (so callers can cleanly no-op).
func (s *service) clientForAccount(ctx context.Context, accountID string) (domain.StripeCheckoutClient, bool, *apierror.APIError) {
	repo := s.repos.NewAccountIntegrationRepo()
	has, apiErr := repo.HasIntegration(ctx, accountID, constants.IntegrationCodeStripe)
	if apiErr != nil {
		return nil, false, apiErr
	}
	if !has {
		return nil, false, nil
	}

	encrypted, isActive, apiErr := repo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeStripe)
	if apiErr != nil {
		return nil, false, apiErr
	}
	if !isActive {
		return nil, false, nil
	}

	// Unreadable credentials are a permanent condition: a blob that won't decrypt or parse will fail identically on every retry, so classifying these as internal errors (which the platform treats as transient) would redeliver every customer edit for this account forever. Authentication errors are non-transient and name the fix.
	//
	// The blob is sealed with the account ID as additional authenticated data, matching how core-service and the legacy dashboard API both encrypt integration credentials.
	plaintext, err := crypto.DecryptAESGCM(encrypted, s.encryptionKey, []byte(accountID))
	if err != nil {
		return nil, false, apierror.NewAuthenticationError("The stored Stripe credentials could not be decrypted. Reconnect the Stripe integration.")
	}
	var creds domain.StripeCredentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, false, apierror.NewAuthenticationError("The stored Stripe credentials could not be read. Reconnect the Stripe integration.")
	}
	if creds.PrivateKey == "" {
		return nil, false, apierror.NewAuthenticationError("The Stripe integration has no API key. Reconnect the Stripe integration.")
	}

	return s.clientFactory.Build(creds.PrivateKey), true, nil
}
