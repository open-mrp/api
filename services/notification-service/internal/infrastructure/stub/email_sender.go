package stub

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// EmailSender is a no-op EmailSender implementation for use in test mode.
type EmailSender struct {
	sent     atomic.Uint64
	once     sync.Once
	instance string
}

// TODO: This needs rethought and old params removed if not needed
// Send returns a message ID no other send has had, the way SES does: the email log deduplicates on it,
// so an ID repeated by this process, or by an earlier one after a restart, would leave that email unlogged.
func (s *EmailSender) Send(_ context.Context, _ domain.EmailData) (*string, *apierror.APIError) {
	s.once.Do(func() { s.instance = uuid.NewString() })
	id := "stub_message_id_" + s.instance + "_" + strconv.FormatUint(s.sent.Add(1), 10)
	return &id, nil
}
