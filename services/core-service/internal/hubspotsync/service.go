// Package hubspotsync orchestrates pushing OpenMRP sales orders to HubSpot as Closed-Won deals, upserting the associated company and contact along the way.
//
// It is the single source of truth for the HubSpot mapping and is designed to be shared by both the incremental order-created consumer (this step) and the account backfill worker (a later step). Each operation is idempotent on replay: deals are keyed on the augno_sales_order_id property, contacts dedupe on email, and companies are matched by domain/name before creation.
package hubspotsync

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	// defaultPipelineID / defaultClosedWonStage target HubSpot's standard sales pipeline.
	defaultPipelineID     = "default"
	defaultClosedWonStage = "closedwon"

	// lifecycleCustomer is the HubSpot lifecycle stage applied to companies/contacts with a won deal.
	lifecycleCustomer = "customer"

	// HubSpot CRM object type names used for associations.
	objectTypeDeals     = "deals"
	objectTypeCompanies = "companies"
	objectTypeContacts  = "contacts"
)

var tracer = tracing.GetTracer("core-service.hubspot_sync")

// Config selects where won deals land. Empty fields fall back to HubSpot's default pipeline / closedwon stage.
type Config struct {
	// PipelineID (optional; default: "default") is the HubSpot pipeline won deals are created in.
	PipelineID string

	// ClosedWonStageID (optional; default: "closedwon") is the HubSpot deal stage applied to won deals.
	ClosedWonStageID string
}

// WithDefaults fills zero-value optional fields with production defaults and returns the config. It is safe to call with a nil receiver.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}

	if c.PipelineID == "" {
		c.PipelineID = defaultPipelineID
	}
	if c.ClosedWonStageID == "" {
		c.ClosedWonStageID = defaultClosedWonStage
	}
	return c
}

// validate ensures the effective config targets a concrete HubSpot pipeline and stage. WithDefaults populates both from production defaults, so an empty value here means a default constant was cleared or overridden with "".
func (c *Config) validate() error {
	if c.PipelineID == "" {
		return errors.New("hubspotsync: pipeline id is required")
	}
	if c.ClosedWonStageID == "" {
		return errors.New("hubspotsync: closed-won stage id is required")
	}
	return nil
}

// Service syncs OpenMRP sales orders to a connected HubSpot account.
type Service interface {
	// SyncOrder upserts the order's company + contact and creates/moves its deal to Closed-Won. It is a no-op (returns nil) when the account has no active HubSpot integration.
	SyncOrder(ctx context.Context, accountID, salesOrderID string) *apierror.APIError

	// RunPreview executes the read-only matching pass for a backfill job: matches customers to HubSpot companies, queues ambiguous ones for review, tallies the dry-run report, and moves the job to review_pending. Writes nothing to HubSpot.
	RunPreview(ctx context.Context, accountID, jobID string) *apierror.APIError

	// RunExecute applies a reviewed backfill job to HubSpot (companies + contacts, then Closed-Won deals for orders on/after the cutoff). Gated on zero pending reviews; resumable via per-page cursor checkpoints.
	RunExecute(ctx context.Context, accountID, jobID string) *apierror.APIError
}

type service struct {
	repos         domain.RepoFactory
	clientFactory domain.HubspotClientFactory
	encryptionKey []byte
	cfg           Config

	// propertiesEnsured tracks accounts whose custom deal properties have been verified/created this process lifetime, so we don't re-check on every order sync. Keyed by account id.
	propertiesEnsured sync.Map
}

func NewService(repos domain.RepoFactory, clientFactory domain.HubspotClientFactory, encryptionKey []byte, cfg Config) Service {
	effective := cfg.WithDefaults()
	if err := effective.validate(); err != nil {
		panic(err)
	}

	return &service{
		repos:         repos,
		clientFactory: clientFactory,
		encryptionKey: encryptionKey,
		cfg:           *effective,
	}
}

func (s *service) SyncOrder(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := tracer.Start(ctx, "hubspotsync.sync_order")
	defer span.End()

	client, connected, apiErr := s.clientForAccount(ctx, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !connected {
		return nil
	}
	if apiErr := s.ensureDealProperties(ctx, client, accountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := s.syncOrderWithClient(ctx, client, accountID, salesOrderID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// syncOrderWithClient performs the per-order sync (company + contact + Closed-Won deal) against an already-resolved client. Lifecycle is promoted to customer because an order represents won business. The backfill execute phase reuses this so a backfilled order and a live-event order produce identical results.
func (s *service) syncOrderWithClient(ctx context.Context, client domain.HubspotClient, accountID, salesOrderID string) *apierror.APIError {
	soRepo := s.repos.NewSalesOrderRepo()
	order, apiErr := soRepo.Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	lines, apiErr := soRepo.GetLines(ctx, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	amount, apiErr := orderTotal(ctx, lines, s.repos.NewUnitConversionRepo().ConvertValue)
	if apiErr != nil {
		return apiErr
	}

	customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, order.OwnerAccountID, order.BuyerAccountID, nil)
	if apiErr != nil {
		return apiErr
	}

	companyID, apiErr := s.syncCompany(ctx, client, accountID, customer, true)
	if apiErr != nil {
		return apiErr
	}

	contactID, apiErr := s.syncContact(ctx, client, accountID, order, customer, companyID, true)
	if apiErr != nil {
		return apiErr
	}

	return s.syncDeal(ctx, client, order, amount, companyID, contactID)
}

// ensureDealProperties verifies/creates the custom deal properties the sync needs, at most once per account per process lifetime (properties are per-portal, i.e. per account).
func (s *service) ensureDealProperties(ctx context.Context, client domain.HubspotClient, accountID string) *apierror.APIError {
	if _, ok := s.propertiesEnsured.Load(accountID); ok {
		return nil
	}
	if apiErr := client.EnsureDealProperties(ctx); apiErr != nil {
		return apiErr
	}
	s.propertiesEnsured.Store(accountID, struct{}{})
	return nil
}

// clientForAccount returns a HubSpot client for the account, reporting connected=false when no integration exists or it is inactive (so callers can cleanly no-op).
func (s *service) clientForAccount(ctx context.Context, accountID string) (domain.HubspotClient, bool, *apierror.APIError) {
	repo := s.repos.NewAccountIntegrationRepo()
	has, apiErr := repo.HasIntegration(ctx, accountID, constants.IntegrationCodeHubspot)
	if apiErr != nil {
		return nil, false, apiErr
	}
	if !has {
		return nil, false, nil
	}

	encrypted, isActive, apiErr := repo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeHubspot)
	if apiErr != nil {
		return nil, false, apiErr
	}
	if !isActive {
		return nil, false, nil
	}

	// Unreadable credentials are a permanent condition: a blob that won't decrypt or parse will fail identically on every retry, so classifying these as internal errors (which the platform treats as transient) would redeliver every order for this account forever. Authentication errors are non-transient and name the fix.
	plaintext, err := crypto.DecryptAESGCM(encrypted, s.encryptionKey, []byte(accountID))
	if err != nil {
		return nil, false, apierror.NewAuthenticationError("The stored HubSpot credentials could not be decrypted. Reconnect the HubSpot integration.")
	}
	var creds domain.HubspotCredentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, false, apierror.NewAuthenticationError("The stored HubSpot credentials could not be read. Reconnect the HubSpot integration.")
	}
	if creds.AccessToken == "" {
		return nil, false, apierror.NewAuthenticationError("The HubSpot integration has no access token. Reconnect the HubSpot integration.")
	}
	return s.clientFactory.Build(creds.AccessToken), true, nil
}

// syncCompany resolves the customer's HubSpot company id, matching or creating it on first sync and reusing the stored mapping thereafter. When promoteLifecycle is set (an order represents won business) it moves the company to the customer lifecycle stage. The resolved mapping is persisted so later syncs and the backfill share it.
func (s *service) syncCompany(ctx context.Context, client domain.HubspotClient, accountID string, customer *domain.Customer, promoteLifecycle bool) (string, *apierror.APIError) {
	mapping, apiErr := s.repos.NewHubspotSyncRepo().GetRecord(ctx, accountID, openMRPTypeCustomer, customer.ID)
	if apiErr != nil {
		return "", apiErr
	}
	if mapping != nil {
		if promoteLifecycle {
			if apiErr := client.UpdateCompany(ctx, mapping.HubspotID, domain.HubspotCompany{Lifecycle: lifecycleCustomer}); apiErr != nil {
				return "", apiErr
			}
		}
		return mapping.HubspotID, nil
	}

	lifecycle := ""
	if promoteLifecycle {
		lifecycle = lifecycleCustomer
	}

	domainName := deriveDomain(customer.URL)
	var matches []domain.HubspotCompany
	if domainName != "" {
		matches, apiErr = client.SearchCompaniesByDomain(ctx, domainName)
	} else {
		matches, apiErr = client.SearchCompaniesByName(ctx, customer.Name)
	}
	if apiErr != nil {
		return "", apiErr
	}

	var companyID string
	if len(matches) > 0 {
		companyID = matches[0].ID
		// Promote lifecycle on the existing company; leave its name/domain untouched.
		if promoteLifecycle {
			if apiErr := client.UpdateCompany(ctx, companyID, domain.HubspotCompany{Lifecycle: lifecycleCustomer}); apiErr != nil {
				return "", apiErr
			}
		}
	} else {
		created, apiErr := client.CreateCompany(ctx, domain.HubspotCompany{Name: customer.Name, Domain: domainName, Lifecycle: lifecycle})
		if apiErr != nil {
			return "", apiErr
		}
		companyID = created.ID
	}

	if apiErr := s.storeMapping(ctx, accountID, openMRPTypeCustomer, customer.ID, objectTypeCompanies, companyID); apiErr != nil {
		return "", apiErr
	}
	return companyID, nil
}

// syncContact upserts the order's primary contact (bill-to email, falling back to the customer email) and associates it to the company.
func (s *service) syncContact(ctx context.Context, client domain.HubspotClient, accountID string, order *domain.SalesOrder, customer *domain.Customer, companyID string, promoteLifecycle bool) (string, *apierror.APIError) {
	email := firstNonEmpty(order.BillToEmail, customer.Email)
	fullName := firstNonEmptyStr(ptrutil.Deref(order.BillToName), customer.Name)
	phone := firstNonEmpty(order.BillToPhone, customer.Phone)
	return s.upsertContact(ctx, client, accountID, customer.ID, email, fullName, phone, companyID, promoteLifecycle)
}

// upsertContact upserts a contact by email (HubSpot's native dedupe key), associates it to the company, persists the customer→contact mapping, and optionally promotes its lifecycle. Returns "" when email is empty.
func (s *service) upsertContact(ctx context.Context, client domain.HubspotClient, accountID, customerID, email, fullName, phone, companyID string, promoteLifecycle bool) (string, *apierror.APIError) {
	if email == "" {
		return "", nil
	}
	lifecycle := ""
	if promoteLifecycle {
		lifecycle = lifecycleCustomer
	}
	first, last := splitName(fullName)
	contact, apiErr := client.UpsertContactByEmail(ctx, domain.HubspotContact{
		Email:     email,
		FirstName: first,
		LastName:  last,
		Phone:     phone,
		Lifecycle: lifecycle,
	})
	if apiErr != nil {
		return "", apiErr
	}
	if companyID != "" {
		if apiErr := client.Associate(ctx, objectTypeContacts, contact.ID, objectTypeCompanies, companyID); apiErr != nil {
			return "", apiErr
		}
	}
	if apiErr := s.storeMapping(ctx, accountID, openMRPTypeContact, customerID, objectTypeContacts, contact.ID); apiErr != nil {
		return "", apiErr
	}
	return contact.ID, nil
}

// storeMapping persists an OpenMRP→HubSpot id mapping (idempotent upsert).
func (s *service) storeMapping(ctx context.Context, accountID, augnoType, augnoID, hubspotType, hubspotID string) *apierror.APIError {
	return s.repos.NewHubspotSyncRepo().UpsertRecord(ctx, domain.UpsertHubspotSyncRecordParams{
		AccountID:   accountID,
		AugnoType:   augnoType,
		AugnoID:     augnoID,
		HubspotType: hubspotType,
		HubspotID:   hubspotID,
	})
}

// syncDeal creates or updates the Closed-Won deal for the order and associates it to the company and contact.
func (s *service) syncDeal(ctx context.Context, client domain.HubspotClient, order *domain.SalesOrder, amount, companyID, contactID string) *apierror.APIError {
	deal := domain.HubspotDeal{
		Name:         order.Number,
		Amount:       amount,
		CloseDate:    closeDate(order),
		PipelineID:   s.cfg.PipelineID,
		StageID:      s.cfg.ClosedWonStageID,
		SalesOrderID: order.ID,
	}

	existing, apiErr := client.SearchDealBySalesOrderID(ctx, order.ID)
	if apiErr != nil {
		return apiErr
	}

	var dealID string
	if existing != nil {
		if apiErr := client.UpdateDeal(ctx, existing.ID, deal); apiErr != nil {
			return apiErr
		}
		dealID = existing.ID
	} else {
		created, apiErr := client.CreateDeal(ctx, deal)
		if apiErr != nil {
			return apiErr
		}
		dealID = created.ID
	}

	if companyID != "" {
		if apiErr := client.Associate(ctx, objectTypeDeals, dealID, objectTypeCompanies, companyID); apiErr != nil {
			return apiErr
		}
	}
	if contactID != "" {
		if apiErr := client.Associate(ctx, objectTypeDeals, dealID, objectTypeContacts, contactID); apiErr != nil {
			return apiErr
		}
	}
	return nil
}
