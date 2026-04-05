package stub

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// EmailSender is a no-op EmailSender implementation for use in test mode.
type EmailSender struct{}

func (s *EmailSender) Send(_ context.Context, _ domain.EmailData) (*string, *apierror.APIError) {
	id := "stub_message_id"
	return &id, nil
}
