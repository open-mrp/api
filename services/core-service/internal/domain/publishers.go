package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
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
}
