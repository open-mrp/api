package stub

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/augno/api/services/notification-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// EmailSender is a no-op EmailSender implementation for use in test mode.
type EmailSender struct {
	sent atomic.Uint64
}

// Send returns a distinct message ID per call, the way SES does.
//
// It used to return one constant. The email log deduplicates on the SES message ID, so
// every email after the first in a process silently deduplicated away and no row was
// written — which made "was this email logged?" depend on whether the email under test
// happened to be the first one the stack ever sent.
func (s *EmailSender) Send(_ context.Context, _ domain.EmailData) (*string, *apierror.APIError) {
	id := "stub_message_id_" + strconv.FormatUint(s.sent.Add(1), 10)
	return &id, nil
}
