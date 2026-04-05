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

// BillingPublisher publishes billing-related commands via the outbox pattern.
type BillingPublisher interface {
	// PublishSyncSeats writes a sync-seats command to the outbox for the given account.
	PublishSyncSeats(ctx context.Context, accountID string) *apierror.APIError
	// PublishReportSeatChange writes a report-seat-change command to the outbox for
	// usage metering with the billing provider.
	PublishReportSeatChange(ctx context.Context, accountID string) *apierror.APIError
}
