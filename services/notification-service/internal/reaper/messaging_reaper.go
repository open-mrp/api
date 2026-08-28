// Package reaper holds the background retention worker for the messaging substrate. It periodically deletes read/dismissed notifications, stale notifications past a hard cap, expired announcements (plus their orphaned receipts), and tombstoned messages with their attachments (deleting each attachment's S3 object before its row), mirroring the inbox purger's lease-guarded single-runner design.
package reaper

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	s3 "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
)

// AttachmentPurgeRef identifies an attachment row whose backing object must be removed before the row is deleted.
type AttachmentPurgeRef struct {
	ID    string
	S3Key string
}

// MessagingReaperRepo deletes expired messaging rows in bounded batches.
type MessagingReaperRepo interface {
	// PurgeActionedNotifications deletes read/dismissed notifications older than retentionHours (measured from creation), up to limit rows. Returns the number deleted.
	PurgeActionedNotifications(ctx context.Context, retentionHours int, limit int32) (int64, error)
	// PurgeStaleNotifications deletes any notification older than capHours, up to limit rows.
	PurgeStaleNotifications(ctx context.Context, capHours int, limit int32) (int64, error)
	// PurgeExpiredAnnouncements deletes announcements expired longer than retentionHours ago.
	PurgeExpiredAnnouncements(ctx context.Context, retentionHours int, limit int32) (int64, error)
	// PurgeOrphanedAnnouncementReceipts deletes receipts whose announcement no longer exists.
	PurgeOrphanedAnnouncementReceipts(ctx context.Context, limit int32) (int64, error)
	// ListPurgeableMessageAttachments returns attachments of messages tombstoned longer than retentionHours.
	ListPurgeableMessageAttachments(ctx context.Context, retentionHours int, limit int32) ([]AttachmentPurgeRef, error)
	// DeleteMessageAttachmentByID deletes a single attachment row (after its S3 object is removed).
	DeleteMessageAttachmentByID(ctx context.Context, id string) error
	// PurgeTombstonedMessages hard-deletes messages tombstoned longer than retentionHours, only once their attachments are gone.
	PurgeTombstonedMessages(ctx context.Context, retentionHours int, limit int32) (int64, error)
}

// MessagingReaperConfig configures the retention worker.
type MessagingReaperConfig struct {
	// ServiceName (required) scopes the lease name so each service runs its own reaper.
	ServiceName string
	// PlatformMode (optional) shortens the default ReapInterval to 1m in test mode.
	PlatformMode constants.PlatformMode
	// NotificationRetentionHours (optional; default 2160 = 90d) is how long read/dismissed notifications are kept.
	NotificationRetentionHours int
	// NotificationMaxAgeHours (optional; default 4320 = 180d) is the hard cap after which any notification is deleted regardless of state.
	NotificationMaxAgeHours int
	// AnnouncementRetentionHours (optional; default 2160 = 90d) is how long expired announcements are kept past expiry.
	AnnouncementRetentionHours int
	// MessageRetentionHours (optional; default 720 = 30d) is the grace window after a message is tombstoned before it and its attachments are hard-purged.
	MessageRetentionHours int
	// ReapInterval (optional; default 1h) controls how often the reap loop runs.
	ReapInterval time.Duration
	// BatchSize (optional; default 1000) bounds rows deleted per statement.
	BatchSize int32
	// LeaseTTL (optional; default 5m) bounds how long the reaper holds its lease.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults.
func (c *MessagingReaperConfig) WithDefaults() *MessagingReaperConfig {
	if c == nil {
		c = &MessagingReaperConfig{}
	}

	reapInterval := c.ReapInterval
	if reapInterval == 0 {
		if c.PlatformMode.IsTest() {
			reapInterval = 1 * time.Minute
		} else {
			reapInterval = 1 * time.Hour
		}
	}

	return &MessagingReaperConfig{
		ServiceName:                c.ServiceName,
		PlatformMode:               c.PlatformMode,
		NotificationRetentionHours: cmp.Or(c.NotificationRetentionHours, 24*90), // 90 days
		NotificationMaxAgeHours:    cmp.Or(c.NotificationMaxAgeHours, 24*180),   // 180 days
		AnnouncementRetentionHours: cmp.Or(c.AnnouncementRetentionHours, 24*90), // 90 days
		MessageRetentionHours:      cmp.Or(c.MessageRetentionHours, 24*30),      // 30 days
		ReapInterval:               reapInterval,
		BatchSize:                  int32(cmp.Or(int(c.BatchSize), 1000)), // #nosec G115 - small config value
		LeaseTTL:                   lease.TTLOr(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *MessagingReaperConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("messaging reaper: service name is required")
	}
	if c.NotificationRetentionHours <= 0 || c.NotificationMaxAgeHours <= 0 || c.AnnouncementRetentionHours <= 0 || c.MessageRetentionHours <= 0 {
		return fmt.Errorf("messaging reaper: retention windows must be positive")
	}
	if c.ReapInterval <= 0 {
		return fmt.Errorf("messaging reaper: reap interval must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("messaging reaper: batch size must be positive")
	}
	if c.LeaseTTL <= 0 {
		return fmt.Errorf("messaging reaper: lease ttl must be positive")
	}
	return nil
}

// MessagingReaper runs a lease-guarded background goroutine that prunes expired messaging rows.
type MessagingReaper struct {
	config      MessagingReaperConfig
	repo        MessagingReaperRepo
	objectStore s3.ObjectStore
	chatBucket  string
	lease       *lease.Lease
	leaseName   string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMessagingReaper constructs the reaper. A non-nil lease ensures only one pod reaps per tick.
// objectStore + chatBucket back attachment-object deletion; when objectStore is nil the attachment purge step is skipped (object cleanup deferred to a bucket lifecycle policy).
func NewMessagingReaper(config *MessagingReaperConfig, repo MessagingReaperRepo, l *lease.Lease, objectStore s3.ObjectStore, chatBucket string) (*MessagingReaper, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, fmt.Errorf("messaging reaper: lease is required")
	}
	return &MessagingReaper{
		config:      *config,
		repo:        repo,
		objectStore: objectStore,
		chatBucket:  chatBucket,
		lease:       l,
		leaseName:   "messaging-reaper-" + config.ServiceName,
	}, nil
}

// Start launches the background reap goroutine.
func (r *MessagingReaper) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.wg.Add(1)
	go r.reapLoop()

	slog.Info("Messaging reaper started",
		"notification_retention_hours", r.config.NotificationRetentionHours,
		"notification_max_age_hours", r.config.NotificationMaxAgeHours,
		"announcement_retention_hours", r.config.AnnouncementRetentionHours,
		"message_retention_hours", r.config.MessageRetentionHours,
		"reap_interval", r.config.ReapInterval,
	)
	return nil
}

// Stop cancels the loop and waits for the goroutine to exit.
func (r *MessagingReaper) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	slog.Info("Messaging reaper stopped")
}

func (r *MessagingReaper) reapLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.ReapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			_ = r.lease.WithLease(r.ctx, r.leaseName, r.config.LeaseTTL, func(ctx context.Context) error {
				r.reap(ctx)
				return nil
			})
		}
	}
}

func (r *MessagingReaper) reap(ctx context.Context) {
	if n, err := r.repo.PurgeActionedNotifications(ctx, r.config.NotificationRetentionHours, r.config.BatchSize); err != nil {
		slog.Error("Failed to purge actioned notifications", "error", err)
	} else if n > 0 {
		slog.Info("Purged actioned notifications", "count", n)
	}

	if n, err := r.repo.PurgeStaleNotifications(ctx, r.config.NotificationMaxAgeHours, r.config.BatchSize); err != nil {
		slog.Error("Failed to purge stale notifications", "error", err)
	} else if n > 0 {
		slog.Info("Purged stale notifications", "count", n)
	}

	if n, err := r.repo.PurgeExpiredAnnouncements(ctx, r.config.AnnouncementRetentionHours, r.config.BatchSize); err != nil {
		slog.Error("Failed to purge expired announcements", "error", err)
	} else if n > 0 {
		slog.Info("Purged expired announcements", "count", n)
	}

	if n, err := r.repo.PurgeOrphanedAnnouncementReceipts(ctx, r.config.BatchSize); err != nil {
		slog.Error("Failed to purge orphaned announcement receipts", "error", err)
	} else if n > 0 {
		slog.Info("Purged orphaned announcement receipts", "count", n)
	}

	// Attachments of tombstoned messages must be purged before the message rows so their S3 objects are deleted first; PurgeTombstonedMessages only removes messages once their attachments are gone.
	r.reapTombstonedAttachments(ctx)

	if n, err := r.repo.PurgeTombstonedMessages(ctx, r.config.MessageRetentionHours, r.config.BatchSize); err != nil {
		slog.Error("Failed to purge tombstoned messages", "error", err)
	} else if n > 0 {
		slog.Info("Purged tombstoned messages", "count", n)
	}
}

// reapTombstonedAttachments deletes the S3 object for each purgeable attachment before deleting its row. Object deletion is idempotent, so a crash between the two steps re-attempts the object delete on the next tick rather than orphaning it. When no object store is configured, only rows are removed (object cleanup is then expected from a bucket lifecycle policy).
func (r *MessagingReaper) reapTombstonedAttachments(ctx context.Context) {
	refs, err := r.repo.ListPurgeableMessageAttachments(ctx, r.config.MessageRetentionHours, r.config.BatchSize)
	if err != nil {
		slog.Error("Failed to list purgeable message attachments", "error", err)
		return
	}
	var purged int
	for _, ref := range refs {
		if ref.S3Key != "" && r.objectStore != nil {
			if apiErr := r.objectStore.Delete(ctx, r.chatBucket, ref.S3Key); apiErr != nil {
				slog.Error("Failed to delete attachment object", "attachment_id", ref.ID, "error", apiErr.PublicMessage)
				continue // leave the row so the object delete is retried next tick
			}
		}
		if err := r.repo.DeleteMessageAttachmentByID(ctx, ref.ID); err != nil {
			slog.Error("Failed to delete attachment row", "attachment_id", ref.ID, "error", err)
			continue
		}
		purged++
	}
	if purged > 0 {
		slog.Info("Purged tombstoned message attachments", "count", purged)
	}
}
