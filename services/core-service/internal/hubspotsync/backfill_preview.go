package hubspotsync

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// Job status values (mirror the hubspot_sync_job.status column).
const (
	StatusPreviewing    = "previewing"
	StatusReviewPending = "review_pending"
	StatusExecuting     = "executing"
	StatusCompleted     = "completed"
	StatusFailed        = "failed"
)

// Company-review queue states and the user actions that resolve them.
const (
	ReviewStatusPending  = "pending"
	ReviewStatusResolved = "resolved"
	ReviewStatusSkipped  = "skipped"

	// ReviewResolutionLink / ReviewResolutionCreateNew are persisted resolutions; ReviewActionSkip is a user action that maps to ReviewStatusSkipped.
	ReviewResolutionLink      = "link"
	ReviewResolutionCreateNew = "create_new"
	ReviewActionSkip          = "skip"
)

// hubspot_sync_record.augno_type values. A customer maps to both its HubSpot company and its primary contact.
const (
	augnoTypeCustomer = "customer"
	augnoTypeContact  = "contact"
)

// customerPageSize bounds how many customers we page through at a time during preview.
const customerPageSize = 200

// CompanyCandidate is one possible HubSpot company match surfaced to a human for an ambiguous customer. It is the JSON shape stored in hubspot_company_review.candidate_matches.
type CompanyCandidate struct {
	HubspotID string `json:"hubspot_id"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
}

// PreviewCounts is the dry-run report stored in hubspot_sync_job.counts. It tallies what the execute phase would do without writing anything to HubSpot.
type PreviewCounts struct {
	CustomersTotal     int `json:"customers_total"`
	CompaniesConfident int `json:"companies_confident"` // unique domain match → auto-linked
	CompaniesAmbiguous int `json:"companies_ambiguous"` // queued for human review
	CompaniesToCreate  int `json:"companies_to_create"` // no match → created on execute
	ContactsWithEmail  int `json:"contacts_with_email"` // customers with an email → contact upsert candidates
}

// RunPreview performs the read-only matching pass for a backfill job: it pulls the account's HubSpot company inventory, matches every customer (confident / ambiguous / none), stores confident company mappings, queues ambiguous ones for review, tallies the dry-run report, and moves the job to review_pending. It writes nothing to HubSpot. On failure it marks the job failed and returns the error.
func (s *service) RunPreview(ctx context.Context, accountID, jobID string) *apierror.APIError {
	ctx, span := tracer.Start(ctx, "hubspotsync.run_preview")
	defer span.End()

	client, connected, apiErr := s.clientForAccount(ctx, accountID)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	if !connected {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apierror.NewValidationError("HubSpot integration is not connected or is inactive.")))
	}

	byDomain, byName, apiErr := s.loadCompanyIndex(ctx, client)
	if apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}

	syncRepo := s.repos.NewHubspotSyncRepo()
	customerRepo := s.repos.NewCustomerRepo()
	var counts PreviewCounts
	var cursor *string
	for {
		page, apiErr := customerRepo.List(ctx, domain.ListCustomersParams{AccountID: accountID, Cursor: cursor, Limit: customerPageSize})
		if apiErr != nil {
			return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
		}

		for _, customer := range page.Items {
			counts.CustomersTotal++
			if customer.Email != nil && *customer.Email != "" {
				counts.ContactsWithEmail++
			}

			tier, candidates := classifyCustomer(customer, byDomain, byName)
			switch tier {
			case matchConfident:
				counts.CompaniesConfident++
				if apiErr := syncRepo.UpsertRecord(ctx, domain.UpsertHubspotSyncRecordParams{
					AccountID:   accountID,
					AugnoType:   augnoTypeCustomer,
					AugnoID:     customer.ID,
					HubspotType: objectTypeCompanies,
					HubspotID:   candidates[0].HubspotID,
				}); apiErr != nil {
					return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
				}
			case matchAmbiguous:
				counts.CompaniesAmbiguous++
				candidatesJSON, err := json.Marshal(candidates)
				if err != nil {
					return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apierror.NewInternalError(err, "Failed to encode HubSpot company candidates.")))
				}
				if _, apiErr := syncRepo.CreateReview(ctx, domain.CreateHubspotCompanyReviewParams{
					JobID:            jobID,
					AccountID:        accountID,
					AugnoCustomerID:  customer.ID,
					CustomerName:     customer.Name,
					CandidateMatches: candidatesJSON,
					Status:           ReviewStatusPending,
				}); apiErr != nil {
					return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
				}
			case matchNone:
				counts.CompaniesToCreate++
			}
		}

		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.NextCursor
	}

	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apierror.NewInternalError(err, "Failed to encode preview counts.")))
	}
	status := StatusReviewPending
	if apiErr := syncRepo.UpdateJob(ctx, domain.UpdateHubspotSyncJobParams{
		ID:        jobID,
		AccountID: accountID,
		Status:    &status,
		Counts:    countsJSON,
	}); apiErr != nil {
		return tracing.Trace(span, s.failJob(ctx, accountID, jobID, apiErr))
	}
	return nil
}

type matchTier int

const (
	matchConfident matchTier = iota // exactly one HubSpot company shares the customer's domain
	matchAmbiguous                  // multiple domain matches, or any name-only match — needs a human
	matchNone                       // no plausible match — execute creates a new company
)

// classifyCustomer buckets a customer against the HubSpot company indexes. Only a unique domain match is confident; a name match (or multiple domain matches) is ambiguous, and nothing matching is none.
func classifyCustomer(customer *domain.Customer, byDomain, byName map[string][]CompanyCandidate) (matchTier, []CompanyCandidate) {
	if domainName := deriveDomain(customer.URL); domainName != "" {
		if matches := byDomain[domainName]; len(matches) == 1 {
			return matchConfident, matches
		} else if len(matches) > 1 {
			return matchAmbiguous, matches
		}
	}

	if nameKey := strings.ToLower(strings.TrimSpace(customer.Name)); nameKey != "" {
		if matches := byName[nameKey]; len(matches) > 0 {
			return matchAmbiguous, matches
		}
	}
	return matchNone, nil
}

// loadCompanyIndex pages the account's entire HubSpot company inventory once and indexes it by domain and by lowercased name for in-memory matching (far fewer API calls than per-customer search).
func (s *service) loadCompanyIndex(ctx context.Context, client domain.HubspotClient) (byDomain, byName map[string][]CompanyCandidate, _ *apierror.APIError) {
	byDomain = map[string][]CompanyCandidate{}
	byName = map[string][]CompanyCandidate{}

	var cursor string
	for {
		page, next, apiErr := client.ListCompanies(ctx, cursor)
		if apiErr != nil {
			return nil, nil, apiErr
		}
		for _, company := range page {
			candidate := CompanyCandidate{HubspotID: company.ID, Name: company.Name, Domain: company.Domain}
			if d := strings.ToLower(strings.TrimSpace(company.Domain)); d != "" {
				byDomain[d] = append(byDomain[d], candidate)
			}
			if n := strings.ToLower(strings.TrimSpace(company.Name)); n != "" {
				byName[n] = append(byName[n], candidate)
			}
		}
		if next == "" {
			break
		}
		cursor = next
	}
	return byDomain, byName, nil
}

// failJob records a terminal failure on the job and returns the original error so the caller can propagate it.
func (s *service) failJob(ctx context.Context, accountID, jobID string, cause *apierror.APIError) *apierror.APIError {
	status := StatusFailed
	msg := cause.Error()
	_ = s.repos.NewHubspotSyncRepo().UpdateJob(ctx, domain.UpdateHubspotSyncJobParams{
		ID:        jobID,
		AccountID: accountID,
		Status:    &status,
		LastError: &msg,
	})
	return cause
}
