// Package integrationhealth keeps an integration's connection_status in step with the provider: every call made
// through a wrapped client reports whether the provider accepted the stored credentials.
package integrationhealth

import (
	"context"
	"errors"
	"log/slog"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
)

// Recorder notes the outcome of one provider call.
type Recorder func(ctx context.Context, callErr *apierror.APIError)

// ForAccount records outcomes on the account's integration for provider. A call that succeeds marks it connected, a
// call refused for its credentials marks it invalid_credentials, and any other failure says nothing about them. A
// change is audited under the caller's identity; recording never fails the call.
func ForAccount(repos domain.RepoFactory, accountID string, provider constants.IntegrationCode) Recorder {
	return func(ctx context.Context, callErr *apierror.APIError) {
		status, ok := StatusOf(callErr)
		if !ok {
			return
		}
		if apiErr := record(ctx, repos, accountID, provider, status); apiErr != nil {
			slog.WarnContext(ctx, "Failed to record integration connection status", "provider", provider, "status", status, "error", apiErr.Error())
		}
	}
}

func record(ctx context.Context, repos domain.RepoFactory, accountID string, provider constants.IntegrationCode, status constants.IntegrationConnectionStatus) *apierror.APIError {
	repo := repos.NewAccountIntegrationRepo()
	before, apiErr := repo.FindByCode(ctx, accountID, provider)
	if apiErr != nil || before.ConnectionStatus == status {
		return apiErr
	}
	changed, apiErr := repo.SetConnectionStatus(ctx, accountID, provider, status)
	if apiErr != nil || !changed {
		return apiErr
	}
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil || !identity.IsTargetAccountSet() || identity.Target.AccountID != accountID {
		return nil
	}
	after := *before
	after.ConnectionStatus = status
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeAccountIntegration,
		ResourceID:   before.ID,
		Changes:      audit.ComputeChanges(before, &after),
	})
}

// StatusOf is the connection status a provider call's outcome implies, if any.
func StatusOf(callErr *apierror.APIError) (constants.IntegrationConnectionStatus, bool) {
	switch {
	case callErr == nil:
		return constants.IntegrationConnectionStatusConnected, true
	case errors.Is(callErr, domain.ErrProviderRejectedCredentials):
		return constants.IntegrationConnectionStatusInvalidCredentials, true
	default:
		return "", false
	}
}

// Shippo wraps a Shippo client so each call is recorded.
func Shippo(c domain.ShippoClient, record Recorder) domain.ShippoClient {
	return &shippoClient{inner: c, record: record}
}

type shippoClient struct {
	inner  domain.ShippoClient
	record Recorder
}

func (c *shippoClient) FindOrRegisterCarrierAccount(ctx context.Context, carrier string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	out, err := c.inner.FindOrRegisterCarrierAccount(ctx, carrier)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) ConnectCarrierAccount(ctx context.Context, carrier, accountID string, params map[string]string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	out, err := c.inner.ConnectCarrierAccount(ctx, carrier, accountID, params)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) GetCarrierAccount(ctx context.Context, objectID string) (*domain.ShippoCarrierAccount, *apierror.APIError) {
	out, err := c.inner.GetCarrierAccount(ctx, objectID)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) DeactivateCarrierAccount(ctx context.Context, objectID string) *apierror.APIError {
	err := c.inner.DeactivateCarrierAccount(ctx, objectID)
	c.record(ctx, err)
	return err
}

func (c *shippoClient) GetCarrierServiceLevels(ctx context.Context, objectID string) ([]domain.ShippoServiceLevel, *apierror.APIError) {
	out, err := c.inner.GetCarrierServiceLevels(ctx, objectID)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) InitiateOAuth(ctx context.Context, objectID, redirectURI string, state *string) (string, *apierror.APIError) {
	out, err := c.inner.InitiateOAuth(ctx, objectID, redirectURI, state)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) FetchShippingRate(ctx context.Context, params domain.FetchShippingRateParams) (float64, *apierror.APIError) {
	out, err := c.inner.FetchShippingRate(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) FetchAllShippingRates(ctx context.Context, params domain.FetchAllShippingRatesParams) ([]domain.ShippoRateOption, *apierror.APIError) {
	out, err := c.inner.FetchAllShippingRates(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) CreateTransactionInstantLabel(ctx context.Context, params domain.CreateLabelParams) (*domain.LabelResult, *apierror.APIError) {
	out, err := c.inner.CreateTransactionInstantLabel(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *shippoClient) RefundTransaction(ctx context.Context, transactionID string) *apierror.APIError {
	err := c.inner.RefundTransaction(ctx, transactionID)
	c.record(ctx, err)
	return err
}

// Stripe wraps an account's Stripe client so each call to Stripe is recorded. Verifying a webhook signature makes no
// call, so it records nothing.
func Stripe(c domain.StripeCheckoutClient, record Recorder) domain.StripeCheckoutClient {
	return &stripeClient{inner: c, record: record}
}

type stripeClient struct {
	inner  domain.StripeCheckoutClient
	record Recorder
}

func (c *stripeClient) CreateOneTimeCheckoutSession(ctx context.Context, params domain.CreateCheckoutSessionParams) (*domain.StripeCheckoutSession, *apierror.APIError) {
	out, err := c.inner.CreateOneTimeCheckoutSession(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *stripeClient) CreateEmbeddedCheckoutSession(ctx context.Context, params domain.CreateEmbeddedCheckoutSessionParams) (*domain.StripeEmbeddedCheckoutSession, *apierror.APIError) {
	out, err := c.inner.CreateEmbeddedCheckoutSession(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *stripeClient) CreateStripeCustomer(ctx context.Context, params domain.CreateStripeCustomerParams) (*domain.StripeCustomer, *apierror.APIError) {
	out, err := c.inner.CreateStripeCustomer(ctx, params)
	c.record(ctx, err)
	return out, err
}

func (c *stripeClient) UpdateStripeCustomer(ctx context.Context, params domain.UpdateStripeCustomerParams) *apierror.APIError {
	err := c.inner.UpdateStripeCustomer(ctx, params)
	c.record(ctx, err)
	return err
}

func (c *stripeClient) ConstructWebhookEvent(payload []byte, signature, webhookSecret string) (*domain.StripeWebhookEvent, *domain.StripePaymentIntent, *apierror.APIError) {
	return c.inner.ConstructWebhookEvent(payload, signature, webhookSecret)
}

func (c *stripeClient) ListPayoutPaymentIntentIDs(ctx context.Context, payoutID string) ([]string, *apierror.APIError) {
	out, err := c.inner.ListPayoutPaymentIntentIDs(ctx, payoutID)
	c.record(ctx, err)
	return out, err
}

// Hubspot wraps an account's HubSpot client so each call is recorded.
func Hubspot(c domain.HubspotClient, record Recorder) domain.HubspotClient {
	return &hubspotClient{inner: c, record: record}
}

type hubspotClient struct {
	inner  domain.HubspotClient
	record Recorder
}

func (c *hubspotClient) EnsureDealProperties(ctx context.Context) *apierror.APIError {
	err := c.inner.EnsureDealProperties(ctx)
	c.record(ctx, err)
	return err
}

func (c *hubspotClient) SearchCompaniesByDomain(ctx context.Context, d string) ([]domain.HubspotCompany, *apierror.APIError) {
	out, err := c.inner.SearchCompaniesByDomain(ctx, d)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) SearchCompaniesByName(ctx context.Context, name string) ([]domain.HubspotCompany, *apierror.APIError) {
	out, err := c.inner.SearchCompaniesByName(ctx, name)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) ListCompanies(ctx context.Context, cursor string) ([]domain.HubspotCompany, string, *apierror.APIError) {
	out, next, err := c.inner.ListCompanies(ctx, cursor)
	c.record(ctx, err)
	return out, next, err
}

func (c *hubspotClient) CreateCompany(ctx context.Context, company domain.HubspotCompany) (*domain.HubspotCompany, *apierror.APIError) {
	out, err := c.inner.CreateCompany(ctx, company)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) UpdateCompany(ctx context.Context, id string, company domain.HubspotCompany) *apierror.APIError {
	err := c.inner.UpdateCompany(ctx, id, company)
	c.record(ctx, err)
	return err
}

func (c *hubspotClient) UpsertContactByEmail(ctx context.Context, contact domain.HubspotContact) (*domain.HubspotContact, *apierror.APIError) {
	out, err := c.inner.UpsertContactByEmail(ctx, contact)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) SearchContactByEmail(ctx context.Context, email string) (*domain.HubspotContact, *apierror.APIError) {
	out, err := c.inner.SearchContactByEmail(ctx, email)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) UpdateContact(ctx context.Context, id string, contact domain.HubspotContact) *apierror.APIError {
	err := c.inner.UpdateContact(ctx, id, contact)
	c.record(ctx, err)
	return err
}

func (c *hubspotClient) SearchDealBySalesOrderID(ctx context.Context, salesOrderID string) (*domain.HubspotDeal, *apierror.APIError) {
	out, err := c.inner.SearchDealBySalesOrderID(ctx, salesOrderID)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) CreateDeal(ctx context.Context, deal domain.HubspotDeal) (*domain.HubspotDeal, *apierror.APIError) {
	out, err := c.inner.CreateDeal(ctx, deal)
	c.record(ctx, err)
	return out, err
}

func (c *hubspotClient) UpdateDeal(ctx context.Context, id string, deal domain.HubspotDeal) *apierror.APIError {
	err := c.inner.UpdateDeal(ctx, id, deal)
	c.record(ctx, err)
	return err
}

func (c *hubspotClient) Associate(ctx context.Context, fromType, fromID, toType, toID string) *apierror.APIError {
	err := c.inner.Associate(ctx, fromType, fromID, toType, toID)
	c.record(ctx, err)
	return err
}
