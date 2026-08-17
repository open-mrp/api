package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/hubspotsync"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/messaging"
)

// asyncBulkDeps hands the async bulk and export engines the plumbing they run on.
func (s *hubspotSyncSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// bulkResolveReviewInput is one submitted decision. The job id rides on every row because the engine hands Resolve nothing but the rows, and Resolve is where a review is checked to belong to the sync the request named.
type bulkResolveReviewInput struct {
	JobID             string
	ReviewID          string
	Action            string
	ResolvedHubspotID *string
}

// ---------------------------------------------------------------------------
// Bulk resolve
// ---------------------------------------------------------------------------

// rejects unknown actions, links with no company id, and the same review decided twice in one request
func validateBulkResolveReviewRows(rows []bulkResolveReviewInput) *apierror.APIError {
	seen := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, row := range rows {
		if strings.TrimSpace(row.ReviewID) == "" {
			rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].review_id", i), "is required")
		} else if _, dup := seen[row.ReviewID]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].review_id", i), fmt.Sprintf("duplicate review %q in request", row.ReviewID))
		}
		seen[row.ReviewID] = struct{}{}

		switch row.Action {
		case hubspotsync.ReviewResolutionLink:
			if row.ResolvedHubspotID == nil || strings.TrimSpace(*row.ResolvedHubspotID) == "" {
				rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].resolved_hubspot_id", i), "is required when linking")
			}
		case hubspotsync.ReviewResolutionCreateNew, hubspotsync.ReviewActionSkip:
		default:
			rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].action", i), "must be one of: link, create_new, skip")
		}
	}
	return rowErrs.Summary("company reviews")
}

// confirms every named review exists on the sync being resolved, and settles each row's target state
func resolveBulkResolveReviewRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []bulkResolveReviewInput) ([]domain.ResolvedBulkHubspotReviewRow, *apierror.APIError) {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ReviewID)
	}

	existing, apiErr := repos.NewHubspotSyncRepo().GetReviewsByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	byID := make(map[string]*domain.HubspotCompanyReview, len(existing))
	for _, review := range existing {
		byID[review.ID] = review
	}

	var rowErrs apierror.RowErrors
	resolved := make([]domain.ResolvedBulkHubspotReviewRow, len(rows))
	for i, row := range rows {
		review, found := byID[row.ReviewID]
		switch {
		case !found:
			rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].review_id", i), fmt.Sprintf("no company review with id %q", row.ReviewID))
			continue
		// Scoped to the job named in the route, not just the account: the route is what the caller was authorized against, so a review from another sync must not be reachable through it.
		case review.JobID != row.JobID:
			rowErrs.AddValidation(i, fmt.Sprintf("reviews[%d].review_id", i), fmt.Sprintf("review %q belongs to a different sync", row.ReviewID))
			continue
		}

		resolved[i] = domain.ResolvedBulkHubspotReviewRow{ReviewID: row.ReviewID}
		switch row.Action {
		case hubspotsync.ReviewResolutionLink:
			resolution := hubspotsync.ReviewResolutionLink
			resolved[i].Status = hubspotsync.ReviewStatusResolved
			resolved[i].Resolution = &resolution
			resolved[i].ResolvedHubspotID = row.ResolvedHubspotID
		case hubspotsync.ReviewResolutionCreateNew:
			resolution := hubspotsync.ReviewResolutionCreateNew
			resolved[i].Status = hubspotsync.ReviewStatusResolved
			resolved[i].Resolution = &resolution
		case hubspotsync.ReviewActionSkip:
			resolved[i].Status = hubspotsync.ReviewStatusSkipped
		}
	}
	if apiErr := rowErrs.Summary("company reviews"); apiErr != nil {
		return nil, apiErr
	}
	return resolved, nil
}

// applies each decision in its own savepoint, so one stale review cannot cost the rest of the batch
func writeBulkResolveReviews(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedBulkHubspotReviewRow) (BulkWriteResult, *apierror.APIError) {
	repo := txRepos.NewHubspotSyncRepo()

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ReviewID)
	}
	// One read for the whole batch: the pre-images the audit events diff against. Reading each row back after its update would triple the statement count for values this already knows.
	existing, apiErr := repo.GetReviewsByIDs(txCtx, accountID, ids)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	byID := make(map[string]*domain.HubspotCompanyReview, len(existing))
	for _, review := range existing {
		byID[review.ID] = review
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old, found := byID[row.ReviewID]
			if !found {
				return apierror.NewResourceNotFoundError(fmt.Sprintf("Company review %q no longer exists.", row.ReviewID))
			}

			if apiErr := repo.ResolveReview(spCtx, domain.ResolveHubspotCompanyReviewParams{
				ID:                row.ReviewID,
				AccountID:         accountID,
				Status:            row.Status,
				Resolution:        row.Resolution,
				ResolvedHubspotID: row.ResolvedHubspotID,
			}); apiErr != nil {
				return apiErr
			}

			updated := *old
			updated.Status = row.Status
			updated.Resolution = row.Resolution
			updated.ResolvedHubspotID = row.ResolvedHubspotID

			return audit.NewPublisher().Publish(spCtx, txRepos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeHubspotCompanyReview,
				ResourceID:   row.ReviewID,
				Changes:      audit.ComputeChanges(old, &updated),
			})
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}

		results = append(results, newRowResult(i, row.ReviewID, false))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

// wires company reviews into the async bulk engine.
func (s *hubspotSyncSvcImpl) bulkResolveSpec() bulkOperationSpec[bulkResolveReviewInput, domain.ResolvedBulkHubspotReviewRow] {
	return bulkOperationSpec[bulkResolveReviewInput, domain.ResolvedBulkHubspotReviewRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeHubspotCompanyReview,
		RoutingKey:       messaging.BulkResolveHubspotCompanyReviews.RoutingKey(),
		PermissionDomain: types.PermissionDomainIntegrations,
		Actions:          []types.Action{types.ActionUpdate},
		EntityName:       "company reviews",
		Validate:         validateBulkResolveReviewRows,
		Resolve:          resolveBulkResolveReviewRows,
		Write:            writeBulkResolveReviews,
	}
}

// BulkResolveReviews validates and resolves synchronously, records the decisions on a job, and returns that job to poll.
func (s *hubspotSyncSvcImpl) BulkResolveReviews(ctx context.Context, params domain.BulkResolveHubspotReviewsParams) (*domain.Job, *apierror.APIError) {
	rows := make([]bulkResolveReviewInput, len(params.Reviews))
	for i, review := range params.Reviews {
		rows[i] = bulkResolveReviewInput{
			JobID:             params.JobID,
			ReviewID:          review.ReviewID,
			Action:            review.Action,
			ResolvedHubspotID: review.ResolvedHubspotID,
		}
	}
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkResolveSpec(), rows)
}

// ExecuteBulkResolveReviews performs the writes for an enqueued bulk resolution.
func (s *hubspotSyncSvcImpl) ExecuteBulkResolveReviews(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkResolveSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns of the review-queue export. The last two are the decision columns a reviewer fills in and the importer reads back, so an exported sheet round-trips.
var hubspotCompanyReviewExportColumns = []excel.ColumnSpec{
	{Header: "Review ID", Key: "review_id", Width: 24},
	{Header: "Customer", Key: "customer", Width: 34},
	{Header: "Customer ID", Key: "customer_id", Width: 24},
	{Header: "Customer Email", Key: "customer_email", Width: 30},
	{Header: "Customer Website", Key: "customer_website", Width: 30},
	{Header: "Candidate Matches", Key: "candidates", Width: 60},
	{Header: "Status", Key: "status", Width: 12},
	{Header: "Decision", Key: "decision", Width: 14},
	{Header: "HubSpot Company ID", Key: "hubspot_company_id", Width: 22},
}

// renders the candidate list as one cell a human can read and compare against HubSpot: "Acme Manufacturing (acme.com) [12345]".
func formatReviewCandidates(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var candidates []hubspotsync.CompanyCandidate
	if err := json.Unmarshal(raw, &candidates); err != nil {
		return ""
	}
	rendered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		entry := candidate.Name
		if candidate.Domain != "" {
			entry += " (" + candidate.Domain + ")"
		}
		rendered = append(rendered, entry+" ["+candidate.HubspotID+"]")
	}
	return strings.Join(rendered, "; ")
}

// wires the review queue into the export engine.
func (s *hubspotSyncSvcImpl) exportSpec() exportSpec[*domain.HubspotCompanyReview, domain.ExportHubspotCompanyReviewsParams] {
	return exportSpec[*domain.HubspotCompanyReview, domain.ExportHubspotCompanyReviewsParams]{
		PermissionDomain: types.PermissionDomainIntegrations,
		Name:             "HubSpot Company Reviews",
		Slug:             "hubspot_company_reviews",
		ResourceType:     constants.ObjectTypeHubspotCompanyReview,
		Columns:          hubspotCompanyReviewExportColumns,

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportHubspotCompanyReviewsParams) ([]*domain.HubspotCompanyReview, *apierror.APIError) {
			// The review list is keyed by job alone, so the job's ownership is what scopes the export to the caller's account.
			if _, apiErr := repos.NewHubspotSyncRepo().GetJob(ctx, accountID, filters.JobID); apiErr != nil {
				return nil, apiErr
			}
			return repos.NewHubspotSyncRepo().ListReviewsForJob(ctx, filters.JobID, filters.Status)
		},

		Project: func(review *domain.HubspotCompanyReview) excel.Row {
			decision := ""
			if review.Resolution != nil {
				decision = *review.Resolution
			} else if review.Status == hubspotsync.ReviewStatusSkipped {
				decision = hubspotsync.ReviewActionSkip
			}
			return excel.Row{
				"review_id":          review.ID,
				"customer":           review.CustomerName,
				"customer_id":        review.AugnoCustomerID,
				"customer_email":     excel.Str(review.CustomerEmail),
				"customer_website":   excel.Str(review.CustomerURL),
				"candidates":         formatReviewCandidates(review.CandidateMatches),
				"status":             review.Status,
				"decision":           decision,
				"hubspot_company_id": excel.Str(review.ResolvedHubspotID),
			}
		},
	}
}

// ExportReviews accepts an export: it records what to build on a job and returns it to poll.
func (s *hubspotSyncSvcImpl) ExportReviews(ctx context.Context, params domain.ExportHubspotCompanyReviewsParams) (*domain.Job, *apierror.APIError) {
	// The export engine authorizes the caller but knows nothing about the job named in the route, and the spec's own ownership check runs in the worker at render time. Without this, exporting another tenant's sync is accepted with a 202 and only fails asynchronously — no rows leak, but the caller is told the request was fine when it never could be.
	identity, apiErr := s.authorize(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, apiErr
	}
	if _, apiErr := s.repos.NewHubspotSyncRepo().GetJob(ctx, identity.Target.AccountID, params.JobID); apiErr != nil {
		return nil, apiErr
	}
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// BuildExportHubspotCompanyReviews renders the file an accepted export recorded.
func (s *hubspotSyncSvcImpl) BuildExportHubspotCompanyReviews(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
