package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var exportTracer = tracing.GetTracer("core-service.export_operation")

// renders a boolean the way a spreadsheet reader expects, and the importer parses
func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

// strips the storage padding off a stored DECIMAL, which MySQL hands back at full
// scale — "12" reads as "12.000000000000000000000000000000".
func decimalCell(stored string) string {
	return db.TrimDecimal(stored)
}

// renders an optional stored DECIMAL, blank where it was never set
func decimalCellPtr(stored *string) string {
	if stored == nil {
		return ""
	}
	return db.TrimDecimal(*stored)
}

// names the read-access check an external caller must clear
type externalAccess int

const (
	// internalOnly admits internal actors and nobody else.
	internalOnly externalAccess = iota
	// externalDirect admits an assigned external actor with read access to the target.
	externalDirect
	// externalCounterparty admits an assigned external actor standing in a
	// counterparty relationship to the target.
	externalCounterparty
)

// describes how one resource renders as a spreadsheet; the engine owns the rest.
// TRow is the domain row, TFilters the request's filters minus pagination.
type exportSpec[TRow, TFilters any] struct {
	// PermissionDomain is the domain read permission is checked against.
	PermissionDomain types.PermissionDomain
	// CheckPermission replaces that check, for a resource whose required
	// permission depends on whose account is being read.
	CheckPermission func(identity *types.Identity) *apierror.APIError
	// ExternalAccess is the check an external caller must clear; the zero value
	// keeps the resource internal-only.
	ExternalAccess externalAccess
	// Name titles the file and, unless SheetName overrides it, the worksheet.
	Name string
	// SheetName overrides the worksheet's name where it differs from Name.
	SheetName string
	// Slug names the resource in the span, snake_case so it groups with the
	// entity's other spans rather than reading as prose.
	Slug string
	// ResourceType is the object type the file lists. It rides on the job, which is
	// otherwise silent about what an export was of: its results are empty, because an
	// export produces a file rather than resources.
	ResourceType constants.ObjectType
	// Ext is the file's extension, for an export whose builder does not render a
	// spreadsheet. Empty means xlsx, which is what every column-driven export is.
	Ext string

	Columns []excel.ColumnSpec
	// ColumnsFor derives the columns from the fetched rows, for a resource whose
	// sheet gains a column per property present in the data. Takes precedence
	// over Columns.
	ColumnsFor func(rows []TRow) []excel.ColumnSpec

	// NarrowFilters restricts the filters to what this caller may see, for a
	// resource whose visibility depends on who is asking.
	NarrowFilters func(identity *types.Identity, filters TFilters) TFilters

	// Fetch returns every matching row, unpaginated, up to one past ExportRowLimit so an
	// oversized export is rejected rather than truncated.
	Fetch func(ctx context.Context, repos domain.RepoFactory, accountID string, filters TFilters) ([]TRow, *apierror.APIError)
	// Project turns one row into its cells, keyed by ColumnSpec.Key.
	Project func(row TRow) excel.Row
	// Expand turns one row into several, for a resource that lists a parent's
	// children one per sheet row (see excel.Group). Takes precedence over Project.
	Expand func(row TRow) []excel.Row
}

// names the object an export job's file is stored under. Derived rather than recorded:
// every part is already on the job, so the worker and the reader agree without storage.
func exportObjectKey(accountID, slug, jobID string, startedAt time.Time, ext string) string {
	return "exports/" + accountID + "/" + slug + "/" + jobID + "/" + excel.FilenameWithExt(slug, startedAt, ext)
}

// records on the job what to build. The slug rides along because the download endpoint
// is given a job id and nothing else.
type exportJobPayload struct {
	Slug    string          `json:"slug"`
	Filters json.RawMessage `json:"filters"`
	// Ext is the file's extension, which forms the end of the object key and therefore
	// the name the browser saves. Absent on payloads written before non-spreadsheet
	// exports existed, and those files were uploaded as .xlsx — so empty means xlsx.
	Ext string `json:"ext,omitempty"`
}

// adapts one resource's spec into the builder the download endpoint calls, decoding the
// filters the accept stored
func exportBuilder[TRow, TFilters any](repos domain.RepoFactory, spec exportSpec[TRow, TFilters]) domain.ExportBuilder {
	return func(ctx context.Context, accountID string, raw json.RawMessage) (*domain.Export, *apierror.APIError) {
		var filters TFilters
		if err := json.Unmarshal(raw, &filters); err != nil {
			return nil, apierror.NewInternalError(err, "Job items are not an export payload.")
		}
		return buildExport(ctx, repos, spec, accountID, filters)
	}
}

// turns the fetched rows into sheet rows, one-to-one or one-to-many
func (s exportSpec[TRow, TFilters]) project(rows []TRow) []excel.Row {
	if s.Expand == nil {
		sheetRows := make([]excel.Row, len(rows))
		for i, row := range rows {
			sheetRows[i] = s.Project(row)
		}
		return sheetRows
	}

	sheetRows := make([]excel.Row, 0, len(rows))
	for _, row := range rows {
		sheetRows = append(sheetRows, s.Expand(row)...)
	}
	return sheetRows
}

// accepts an export: authorizes the caller and records what to build on a job. Nothing
// is rendered here — the file is built when it is downloaded.
func enqueueExport[TRow, TFilters any](
	ctx context.Context,
	deps asyncBulkDeps,
	spec exportSpec[TRow, TFilters],
	filters TFilters,
) (*domain.Job, *apierror.APIError) {
	ctx, span := exportTracer.Start(ctx, "service.export.enqueue."+spec.Slug)
	defer span.End()

	identity, apiErr := authorizeExport(ctx, deps, spec.checkPermission(), spec.ExternalAccess)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	if spec.NarrowFilters != nil {
		filters = spec.NarrowFilters(identity, filters)
	}

	// Narrowed before storing, so the download re-reads what this caller was allowed to
	// see rather than re-deriving it from an identity it no longer has. The slug rides
	// along because the job is the only thing the download endpoint is given.
	rawFilters, err := json.Marshal(filters)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal export filters."))
	}
	jobItems, err := json.Marshal(exportJobPayload{Slug: spec.Slug, Filters: rawFilters, Ext: spec.Ext})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal export filters."))
	}

	op, ok := messaging.ExportOperationFor(spec.Slug)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("No export command is registered for "+spec.Slug+"."))
	}

	meds := deps.mediatorFactory.Build(deps.repos)
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Job](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		createdByID := jobCreatedByID(ctx, deps.repos, accountID, identity)

		var raisedJob *domain.Job
		apiErr = deps.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			job, apiErr := deps.jobs(txRepos).CreateJob(txCtx, domain.CreateJobServiceParams{
				JobItems:     jobItems,
				Type:         constants.JobTypeExport,
				ResourceType: spec.ResourceType,
				CreatedByID:  createdByID,
			})
			if apiErr != nil {
				return apiErr
			}
			raisedJob = job

			// Enqueued via the outbox inside the transaction so the command is published
			// if and only if the job commits.
			payloadJSON, err := json.Marshal(domain.BulkOperationJobEvent{JobID: job.ID})
			if err != nil {
				return apierror.NewInternalError(err, "Failed to marshal export job event.")
			}
			msg := contracts.AmqpMessage{Data: payloadJSON, Identity: identity}
			if requestID, ok := appctx.GetRequestID(txCtx); ok {
				msg.RequestID = requestID
			}
			if _, err := txRepos.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
				ServiceName: domain.ServiceName,
				MessageType: string(op.RoutingKey()),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(op.RoutingKey()),
				Payload:     msg,
			}); err != nil {
				return apierror.NewInternalError(err, "Failed to create export outbox message.")
			}

			return deps.mediatorFactory.Build(txRepos).Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, raisedJob)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return raisedJob, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// fetches the rows and renders them, the part of an export that is pure work
func buildExport[TRow, TFilters any](
	ctx context.Context,
	repos domain.RepoFactory,
	spec exportSpec[TRow, TFilters],
	accountID string,
	filters TFilters,
) (*domain.Export, *apierror.APIError) {
	rows, apiErr := spec.Fetch(ctx, repos, accountID, filters)
	if apiErr != nil {
		return nil, apiErr
	}
	// Fetch reads one row past the cap, so an overflow means the account has more than a
	// worker can hold in memory: fail deterministically rather than render a silent truncation.
	if len(rows) > domain.ExportRowLimit {
		return nil, apierror.NewValidationError(fmt.Sprintf(
			"This export matches more than %d rows. Narrow the filters and try again.", domain.ExportRowLimit))
	}

	columns := spec.Columns
	if spec.ColumnsFor != nil {
		columns = spec.ColumnsFor(rows)
	}

	sheetName := spec.SheetName
	if sheetName == "" {
		sheetName = spec.Name
	}

	body, err := excel.Build(excel.Spec{
		Sheets: []excel.Sheet{{
			Name:    sheetName,
			Columns: columns,
			Rows:    spec.project(rows),
		}},
	})
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build the export file.")
	}

	return &domain.Export{
		ContentType: excel.ContentType,
		Body:        body,
		// Resource rows, not sheet rows: a grouped export writes one row per child.
		RowCount: int32(len(rows)), // #nosec G115 - a sheet cannot hold more rows than an int32
	}, nil
}

// settles on the read-permission check this resource authorizes against
func (s exportSpec[TRow, TFilters]) checkPermission() func(*types.Identity) *apierror.APIError {
	if s.CheckPermission != nil {
		return s.CheckPermission
	}
	domainName := s.PermissionDomain
	return func(identity *types.Identity) *apierror.APIError {
		return identity.CheckHasPermission(domainName, types.ActionRead)
	}
}

// resolves the caller to the identity whose account's rows may be exported
func authorizeExport(ctx context.Context, deps asyncBulkDeps, checkPermission func(*types.Identity) *apierror.APIError, access externalAccess) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}

	if access == internalOnly {
		if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
			return nil, apiErr
		}
	} else if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, apiErr
	}

	if apiErr := checkPermission(identity); apiErr != nil {
		return nil, apiErr
	}
	if !identity.IsTargetAccountSet() {
		return nil, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
	}

	if identity.IsExternalTarget() {
		readAccess := deps.mediatorFactory.Build(deps.repos).ReadAccess
		var apiErr *apierror.APIError
		switch access {
		case externalDirect:
			apiErr = readAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID)
		case externalCounterparty:
			apiErr = readAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID)
		case internalOnly:
		}
		if apiErr != nil {
			return nil, apiErr
		}
	}

	return identity, nil
}
