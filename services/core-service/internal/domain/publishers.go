package domain

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// NotificationPublisher publishes email notification messages via the outbox pattern.
type NotificationPublisher interface {
	// PublishSendEmail writes a send-email command to the outbox for reliable delivery.
	PublishSendEmail(ctx context.Context, data messaging.EmailSendData) *apierror.APIError
}

// SalesOrderEventPublisher publishes sales-order domain events via the outbox pattern.
type SalesOrderEventPublisher interface {
	// PublishSalesOrderCreated writes a sales-order-created event to the outbox so out-of-band consumers (e.g. CRM sync) can react without blocking the create.
	PublishSalesOrderCreated(ctx context.Context, data messaging.SalesOrderCreatedData) *apierror.APIError
	// PublishSalesOrderShippingUpdated writes a sales-order shipping-changed event to the outbox so the shipment records are re-synced out-of-band from the update.
	PublishSalesOrderShippingUpdated(ctx context.Context, data messaging.SalesOrderShippingUpdatedData) *apierror.APIError
}

// HubspotSyncPublisher publishes HubSpot backfill commands via the outbox pattern, so the command commits atomically with the job row.
type HubspotSyncPublisher interface {
	// PublishPreview writes a preview command to the outbox for the given backfill job.
	PublishPreview(ctx context.Context, data messaging.HubspotSyncCommandData) *apierror.APIError
	// PublishExecute writes an execute command to the outbox for the given backfill job.
	PublishExecute(ctx context.Context, data messaging.HubspotSyncCommandData) *apierror.APIError
}

// BillingPublisher publishes billing-related commands via the outbox pattern.
type BillingPublisher interface {
	// PublishSyncSeats writes a sync-seats command to the outbox for the given account.
	PublishSyncSeats(ctx context.Context, accountID string) *apierror.APIError
	// PublishReportSeatChange writes a report-seat-change command to the outbox for usage metering with the billing provider.
	PublishReportSeatChange(ctx context.Context, accountID string) *apierror.APIError
	// Writes a report-invoice-created command to the outbox for usage metering with the billing provider.
	PublishReportInvoiceCreated(ctx context.Context, accountID, invoiceID string) *apierror.APIError
}

// ProductionScheduleEnqueuer publishes a generate command. The cadence tick uses it so
// the solve happens out of band rather than inside the scheduler lease.
type ProductionScheduleEnqueuer interface {
	// EnqueueGeneration writes a generate-production-schedule command to the outbox for the given placeholder schedule.
	EnqueueGeneration(ctx context.Context, params EnqueueGenerationParams) *apierror.APIError
}
