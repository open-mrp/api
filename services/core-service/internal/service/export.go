package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// Everything that knows a rendered export lives in object storage: the seam that puts it
// there, the worker that drives a job through it, and the read path that links back to it.
// excel_export.go builds the file; nothing in it knows where the bytes go.

// --- Delivery ---

// The link carries its own authorization and is not tied to the caller, so it is kept
// short: reading the job signs a fresh one, and the bucket keeps the file for a day either way.
const exportURLExpiry = 5 * time.Minute

// carries a rendered export to object storage and hands back links to it. Bound to one
// bucket so callers never repeat it.
type ExportDelivery struct {
	store  s3client.ObjectStore
	bucket string
}

// binds the object store to the exports bucket.
func NewExportDelivery(store s3client.ObjectStore, bucket string) ExportDelivery {
	return ExportDelivery{store: store, bucket: bucket}
}

func (d ExportDelivery) Upload(ctx context.Context, key string, body []byte, contentType string) *apierror.APIError {
	return d.store.Upload(ctx, d.bucket, key, bytes.NewReader(body), contentType)
}

// signs a link to an already-uploaded export. Signing is local, so this makes no
// network call and a fresh link costs nothing.
func (d ExportDelivery) PresignedURL(ctx context.Context, key string) (string, *apierror.APIError) {
	return d.store.GetPresignedURL(ctx, d.bucket, key, exportURLExpiry)
}

// --- Read path ---

type exportSvcImpl struct {
	delivery ExportDelivery
}

type ExportSvcConfig struct {
	// Delivery (required) signs the link to a rendered export.
	Delivery ExportDelivery
}

func (c *ExportSvcConfig) validate() error {
	if c.Delivery.store == nil {
		return fmt.Errorf("export service: delivery is required")
	}
	return nil
}

func NewExportSvc(config *ExportSvcConfig) domain.ExportSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &exportSvcImpl{delivery: config.Delivery}
}

// signs a link to the file an export job produced. The key is derived from the job, so
// nothing had to be stored to find it again, and signing is local so this costs no I/O.
func (s *exportSvcImpl) DownloadURL(ctx context.Context, job *domain.Job) (string, *apierror.APIError) {
	if job.Type != constants.JobTypeExport || job.CompletedAt == nil {
		return "", nil
	}
	key, apiErr := exportKeyFor(job)
	if apiErr != nil {
		return "", apiErr
	}
	return s.delivery.PresignedURL(ctx, key)
}

// derives where a completed export job's file was written. The worker derives the same
// string from the same job, which is what lets nothing be stored.
func exportKeyFor(job *domain.Job) (string, *apierror.APIError) {
	if job.AccountID == nil || job.StartedAt == nil {
		return "", apierror.NewInvariantViolationError("A completed export job is missing the fields its object key derives from.")
	}
	var payload exportJobPayload
	if err := json.Unmarshal(job.JobItems, &payload); err != nil {
		return "", apierror.NewInternalError(err, "Job items are not an export payload.")
	}
	return exportObjectKey(*job.AccountID, payload.Slug, job.ID, *job.StartedAt, payload.Ext), nil
}

// --- Worker ---

// runs every accepted export. One instance serves all thirteen: the resource is on the
// job, so the only per-export variance is which builder to call.
type ExportRunner struct {
	repos         domain.RepoFactory
	jobSvcFactory domain.JobSvcFactory
	delivery      ExportDelivery
	builders      map[string]domain.ExportBuilder
}

type ExportRunnerConfig struct {
	// Repos (required) is the repository factory the render reads through.
	Repos domain.RepoFactory
	// JobSvcFactory (required) settles the job the export is tracked by.
	JobSvcFactory domain.JobSvcFactory
	// Delivery (required) uploads the rendered file.
	Delivery ExportDelivery
	// Builders (required) render one resource each, keyed by its slug.
	Builders map[string]domain.ExportBuilder
}

func NewExportRunner(config *ExportRunnerConfig) *ExportRunner {
	return &ExportRunner{
		repos:         config.Repos,
		jobSvcFactory: config.JobSvcFactory,
		delivery:      config.Delivery,
		builders:      config.Builders,
	}
}

// renders an accepted export, uploads it, and settles the job. The inbox de-dupes
// redeliveries; the terminal guard covers a replay that outlives its record.
func (r *ExportRunner) Render(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	ctx, span := exportTracer.Start(ctx, "service.export.render")
	defer span.End()

	if event.JobID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Export job event is missing a job."))
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	accountID := identity.Target.AccountID

	jobs := r.jobSvcFactory.Build(r.repos)
	job, apiErr := jobs.GetJobForExecution(ctx, event.JobID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if job.IsTerminal() {
		return nil
	}

	var payload exportJobPayload
	if err := json.Unmarshal(job.JobItems, &payload); err != nil {
		apiErr := apierror.NewInternalError(err, "Job items are not an export payload.")
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}
	build, ok := r.builders[payload.Slug]
	if !ok {
		apiErr := apierror.NewInvariantViolationError("No export is registered for " + payload.Slug + ".")
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	// The stamp forms half the object key, so the reader derives the same key later
	// without anything having been stored.
	startedAt, apiErr := jobs.StartJob(ctx, domain.StartJobParams{JobID: job.ID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	export, apiErr := build(ctx, accountID, payload.Filters)
	if apiErr != nil {
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	key := exportObjectKey(accountID, payload.Slug, job.ID, startedAt, payload.Ext)
	if apiErr := r.delivery.Upload(ctx, key, export.Body, export.ContentType); apiErr != nil {
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	// Completed only once the object is durable, so a completed job always has a file.
	if apiErr := jobs.CompleteJob(ctx, domain.CompleteJobParams{JobID: job.ID}); apiErr != nil {
		jobs.FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	return nil
}
