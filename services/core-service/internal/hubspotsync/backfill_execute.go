package hubspotsync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

// Execute phases (stored in hubspot_sync_job.cursors.phase) so an interrupted run resumes where it left off.
const (
	phaseCompanies = "companies"
	phaseDeals     = "deals"
	phaseDone      = "done"
)

const orderPageSize = 200

// executeCursors is the resume state persisted in hubspot_sync_job.cursors.
type executeCursors struct {
	Phase          string  `json:"phase"`
	CustomerCursor *string `json:"customer_cursor,omitempty"`
	OrderCursor    *string `json:"order_cursor,omitempty"`
}

// RunExecute applies a reviewed backfill job to HubSpot. It is gated on zero pending company reviews, then runs two passes: (1) resolve/create each customer's company and primary contact (no lifecycle yet) and (2) stream the account's orders on/after the go-live cutoff, syncing each as a Closed-Won deal (which promotes the now-won company/contact to the customer lifecycle). Progress is checkpointed per page in job.cursors so a failed run resumes; every operation is idempotent, so a re-run converges. On failure the job is marked failed with the error.
func (s *service) RunExecute(ctx context.Context, accountID, jobID string) *apierror.APIError {
	ctx, span := tracer.Start(ctx, "hubspotsync.run_execute")
	defer span.End()

	syncRepo := s.repos.NewHubspotSyncRepo()

	// StartExecute has already claimed the job into the executing state, so every failure from here on has to land on the job — returning bare would strand it as permanently in-flight with nothing to explain why.
	job, apiErr := syncRepo.GetJob(ctx, accountID, jobID)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}

	pending, apiErr := syncRepo.CountPendingReviews(ctx, jobID)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	if pending > 0 {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apierror.NewValidationError(fmt.Sprintf("Resolve all %d pending company reviews before executing the sync.", pending))))
	}

	client, connected, apiErr := s.clientForAccount(ctx, accountID)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	if !connected {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apierror.NewValidationError("HubSpot integration is not connected or is inactive.")))
	}
	if apiErr := s.ensureDealProperties(ctx, client, accountID); apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}

	cur := parseExecuteCursors(job.Cursors)

	executing := StatusExecuting
	startParams := domain.UpdateHubspotSyncJobParams{ID: jobID, AccountID: accountID, Status: &executing}
	if job.StartedAt == nil {
		now := time.Now().UTC()
		startParams.StartedAt = &now
	}
	if apiErr := syncRepo.UpdateJob(ctx, startParams); apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}

	reviews, apiErr := syncRepo.ListReviewsForJob(ctx, jobID, nil)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	reviewByCustomer := make(map[string]*domain.HubspotCompanyReview, len(reviews))
	for _, review := range reviews {
		reviewByCustomer[review.AugnoCustomerID] = review
	}

	// Pass 1: companies + contacts (no lifecycle).
	if cur.Phase == "" || cur.Phase == phaseCompanies {
		if apiErr := s.executeCompaniesPass(ctx, client, accountID, jobID, reviewByCustomer, &cur); apiErr != nil {
			return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
		}
		cur.Phase = phaseDeals
		cur.CustomerCursor = nil
		if apiErr := s.persistCursors(ctx, accountID, jobID, cur); apiErr != nil {
			return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
		}
	}

	// Pass 2: deals for orders on/after the cutoff (skipped entirely when no cutoff = no historical deals).
	if cur.Phase == phaseDeals {
		if job.GoLiveCutoffAt != nil {
			if apiErr := s.executeDealsPass(ctx, client, accountID, jobID, *job.GoLiveCutoffAt, &cur); apiErr != nil {
				return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
			}
		}
		cur.Phase = phaseDone
		if apiErr := s.persistCursors(ctx, accountID, jobID, cur); apiErr != nil {
			return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
		}
	}

	completed := StatusCompleted
	completedAt := time.Now().UTC()
	if apiErr := syncRepo.UpdateJob(ctx, domain.UpdateHubspotSyncJobParams{ID: jobID, AccountID: accountID, Status: &completed, CompletedAt: &completedAt}); apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	return nil
}

// executeCompaniesPass resolves and persists each customer's company + primary contact without setting lifecycle (that happens in the deals pass for customers with a won order). It checkpoints the customer cursor per page.
func (s *service) executeCompaniesPass(ctx context.Context, client domain.HubspotClient, accountID, jobID string, reviewByCustomer map[string]*domain.HubspotCompanyReview, cur *executeCursors) *apierror.APIError {
	customerRepo := s.repos.NewCustomerRepo()
	cursor := cur.CustomerCursor
	for {
		page, apiErr := customerRepo.List(ctx, domain.ListCustomersParams{AccountID: accountID, Cursor: cursor, Limit: customerPageSize})
		if apiErr != nil {
			return apiErr
		}
		for _, customer := range page.Items {
			if apiErr := s.executeCustomerCompany(ctx, client, accountID, customer, reviewByCustomer[customer.ID]); apiErr != nil {
				return apiErr
			}
		}
		if !page.PageInfo.HasNextPage {
			cur.CustomerCursor = nil
			return nil
		}
		cursor = page.PageInfo.NextCursor
		cur.CustomerCursor = cursor
		if apiErr := s.persistCursors(ctx, accountID, jobID, *cur); apiErr != nil {
			return apiErr
		}
	}
}

// executeCustomerCompany resolves one customer's HubSpot company (confident mapping / resolved review / create-new), persists the mapping, and upserts the primary contact — all without lifecycle. Customers whose review was skipped are left untouched.
func (s *service) executeCustomerCompany(ctx context.Context, client domain.HubspotClient, accountID string, customer *domain.Customer, review *domain.HubspotCompanyReview) *apierror.APIError {
	mapping, apiErr := s.repos.NewHubspotSyncRepo().GetRecord(ctx, accountID, openMRPTypeCustomer, customer.ID)
	if apiErr != nil {
		return apiErr
	}

	var companyID string
	switch {
	case mapping != nil:
		companyID = mapping.HubspotID
	case review != nil && review.Status == ReviewStatusSkipped:
		return nil
	case review != nil && review.Status == ReviewStatusResolved && ptrutil.Deref(review.Resolution) == ReviewResolutionLink:
		if review.ResolvedHubspotID == nil {
			return apierror.NewInternalError(nil, "Resolved company review is missing the linked HubSpot company id.")
		}
		companyID = *review.ResolvedHubspotID
		if apiErr := s.storeMapping(ctx, accountID, openMRPTypeCustomer, customer.ID, objectTypeCompanies, companyID); apiErr != nil {
			return apiErr
		}
	default:
		// create_new resolution, or a "none" tier with no review: create a fresh company.
		companyID, apiErr = s.createCompanyNoLifecycle(ctx, client, accountID, customer)
		if apiErr != nil {
			return apiErr
		}
	}

	if _, apiErr := s.upsertContact(ctx, client, accountID, customer.ID, ptrutil.Deref(customer.Email), customer.Name, ptrutil.Deref(customer.Phone), companyID, false); apiErr != nil {
		return apiErr
	}
	return nil
}

// createCompanyNoLifecycle creates a HubSpot company for the customer (no lifecycle) and persists the mapping. Used for create-new resolutions and unmatched customers; it does not re-search for matches (preview already classified them).
func (s *service) createCompanyNoLifecycle(ctx context.Context, client domain.HubspotClient, accountID string, customer *domain.Customer) (string, *apierror.APIError) {
	created, apiErr := client.CreateCompany(ctx, domain.HubspotCompany{Name: customer.Name, Domain: deriveDomain(customer.URL)})
	if apiErr != nil {
		return "", apiErr
	}
	if apiErr := s.storeMapping(ctx, accountID, openMRPTypeCustomer, customer.ID, objectTypeCompanies, created.ID); apiErr != nil {
		return "", apiErr
	}
	return created.ID, nil
}

// executeDealsPass streams orders on/after the cutoff and syncs each whose customer has a company mapping (skipping customers the user excluded). It reuses syncOrderWithClient, so each order produces the same Closed-Won deal as the live path and promotes the won customer's lifecycle. Checkpoints the order cursor per page.
func (s *service) executeDealsPass(ctx context.Context, client domain.HubspotClient, accountID, jobID string, cutoff time.Time, cur *executeCursors) *apierror.APIError {
	soRepo := s.repos.NewSalesOrderRepo()
	syncRepo := s.repos.NewHubspotSyncRepo()
	startDate := cutoff.UTC().Format("2006-01-02")
	cursor := cur.OrderCursor
	for {
		page, apiErr := soRepo.List(ctx, domain.ListSalesOrdersParams{AccountID: accountID, StartDate: &startDate, Cursor: cursor, Limit: orderPageSize})
		if apiErr != nil {
			return apiErr
		}
		for _, order := range page.SalesOrders {
			mapping, apiErr := syncRepo.GetRecord(ctx, accountID, openMRPTypeCustomer, order.BuyerAccountID)
			if apiErr != nil {
				return apiErr
			}
			if mapping == nil {
				continue // customer was skipped during review; don't create a deal for them
			}
			if apiErr := s.syncOrderWithClient(ctx, client, accountID, order.ID); apiErr != nil {
				return apiErr
			}
		}
		if !page.PageInfo.HasNextPage {
			cur.OrderCursor = nil
			return nil
		}
		cursor = page.PageInfo.NextCursor
		cur.OrderCursor = cursor
		if apiErr := s.persistCursors(ctx, accountID, jobID, *cur); apiErr != nil {
			return apiErr
		}
	}
}

// persistCursors checkpoints the execute resume state on the job.
func (s *service) persistCursors(ctx context.Context, accountID, jobID string, cur executeCursors) *apierror.APIError {
	encoded, err := json.Marshal(cur)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to encode sync cursors.")
	}
	return s.repos.NewHubspotSyncRepo().UpdateJob(ctx, domain.UpdateHubspotSyncJobParams{ID: jobID, AccountID: accountID, Cursors: encoded})
}

// parseExecuteCursors decodes the resume state, defaulting to the companies phase for a fresh or unparseable value.
func parseExecuteCursors(raw json.RawMessage) executeCursors {
	var cur executeCursors
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &cur)
	}
	return cur
}
