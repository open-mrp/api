package messaging

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/lease"
)

// defaultFailureAlertRecipient mirrors the 5xx error alert recipient so message-processing failures land in the same inbox.
const defaultFailureAlertRecipient = "dev@augno.com"

// maxFailureErrorLen caps how much of a row's last_error is embedded in the alert email so a single verbose stack trace cannot bloat the message.
const maxFailureErrorLen = 500

// InboxFailure describes a message_inbox row the monitor considers failed, stuck, or discarded: the handler recorded an error, the row was inserted and left unleased past the crash-stuck window, or the handler rejected the message as unprocessable.
type InboxFailure struct {
	ID          int64
	MessageID   string
	ServiceName string
	Handler     string
	MessageType string
	Status      InboxStatus
	Attempts    int
	LastError   *string
	ReceivedAt  time.Time
}

// OutboxFailure describes a message_outbox row the enqueuer gave up on: status = 'failed' after exhausting max_attempts publish attempts.
type OutboxFailure struct {
	ID          int64
	MessageID   string
	ServiceName string
	MessageType string
	Destination string
	RoutingKey  string
	Attempts    int
	MaxAttempts int
	LastError   *string
	CreatedAt   time.Time
}

// FailureMonitorRepo defines the persistence interface used by the FailureMonitor to find un-alerted failed/stuck messages and mark them alerted. It is backed by the shared message_inbox and message_outbox tables; a single implementation scans the whole MySQL fleet's messages because those services share one database.
type FailureMonitorRepo interface {
	// ListUnalertedInboxFailures returns un-alerted inbox rows that need a human: 'received' rows whose lease has lapsed and that either carry a last_error or have sat unprocessed longer than crashStuckMinutes, plus every 'discarded' row. Up to limit rows.
	ListUnalertedInboxFailures(ctx context.Context, crashStuckMinutes int, limit int32) ([]InboxFailure, error)

	// ListUnalertedOutboxFailures returns outbox rows in 'failed' status with alerted_at IS NULL, up to limit rows.
	ListUnalertedOutboxFailures(ctx context.Context, limit int32) ([]OutboxFailure, error)

	// MarkInboxAlerted stamps alerted_at on the given inbox rows so subsequent scans skip them.
	MarkInboxAlerted(ctx context.Context, ids []int64) error

	// MarkOutboxAlerted stamps alerted_at on the given outbox rows so subsequent scans skip them.
	MarkOutboxAlerted(ctx context.Context, ids []int64) error
}

// FailureMonitorConfig holds the configuration for the message failure monitor worker.
type FailureMonitorConfig struct {
	// ServiceName (required) identifies which service hosts this monitor. It scopes the distributed lease so the monitor runs on a single pod.
	ServiceName string

	// PlatformMode (optional; default: "") suppresses alert emails in development mode and shortens the default ScanInterval to 1m in test mode.
	PlatformMode constants.PlatformMode

	// Recipient (optional; default: dev@augno.com) is the email address alerts are sent to.
	Recipient string

	// ScanInterval (optional; default: 5m, or 1m in test) controls how frequently the monitor scans for new failures.
	ScanInterval time.Duration

	// CrashStuckMinutes (optional; default: 30) is how long an inbox row may sit unprocessed (no last_error) before the monitor treats it as crash-stuck and alerts on it.
	CrashStuckMinutes int

	// BatchSize (optional; default: 100) caps how many inbox and outbox failures are pulled into a single alert email per scan.
	BatchSize int32

	// LeaseTTL (optional; default: 5m) bounds how long the monitor holds its lease before a crashed holder's claim expires.
	LeaseTTL time.Duration
}

// WithDefaults fills zero-value fields with production defaults and returns the config.
func (c *FailureMonitorConfig) WithDefaults() *FailureMonitorConfig {
	if c == nil {
		c = &FailureMonitorConfig{}
	}

	scanInterval := c.ScanInterval
	if scanInterval == 0 {
		if c.PlatformMode.IsTest() {
			scanInterval = 1 * time.Minute
		} else {
			scanInterval = 5 * time.Minute
		}
	}

	return &FailureMonitorConfig{
		ServiceName:       c.ServiceName,
		PlatformMode:      c.PlatformMode,
		Recipient:         cmp.Or(c.Recipient, defaultFailureAlertRecipient),
		ScanInterval:      scanInterval,
		CrashStuckMinutes: cmp.Or(c.CrashStuckMinutes, 30),
		BatchSize:         int32(cmp.Or(int(c.BatchSize), 100)), // #nosec G115 - small config value
		LeaseTTL:          lease.TTLOr(c.LeaseTTL, 5*time.Minute),
	}
}

func (c *FailureMonitorConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("failure monitor: service name is required")
	}
	if c.ScanInterval <= 0 {
		return fmt.Errorf("failure monitor: scan interval must be positive")
	}
	if c.CrashStuckMinutes <= 0 {
		return fmt.Errorf("failure monitor: crash-stuck minutes must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("failure monitor: batch size must be positive")
	}
	return nil
}

// FailureMonitor runs a background goroutine that periodically scans the message_inbox and message_outbox tables for async work that failed to process and emails a digest to the configured recipient. It is the async-message analogue of the api-gateway's 5xx error alert: the 5xx path is per-request and inline, but inbox/outbox failures (handler errors that dead-letter, publish give-ups, and crash-stuck rows) have no single inline moment, so they are surfaced by this scan instead.
//
// Alerts are deduplicated via the alerted_at column: a row is stamped once it has been included in an email so it is never re-alerted. The scan runs under a distributed lease so only one pod alerts per tick, and is suppressed entirely in development mode.
type FailureMonitor struct {
	config    FailureMonitorConfig
	repo      FailureMonitorRepo
	outbox    OutboxRepo
	lease     *lease.Lease
	leaseName string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewFailureMonitor creates a new message failure monitor. A non-nil lease is required so only one pod scans and alerts each tick. The outbox is used to enqueue the alert email through the same durable outbox → notification-service → SES pipeline as every other transactional email.
func NewFailureMonitor(config *FailureMonitorConfig, repo FailureMonitorRepo, outbox OutboxRepo, l *lease.Lease) (*FailureMonitor, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, fmt.Errorf("failure monitor: repo is required")
	}
	if outbox == nil {
		return nil, fmt.Errorf("failure monitor: outbox is required")
	}
	if l == nil {
		return nil, fmt.Errorf("failure monitor: lease is required")
	}
	return &FailureMonitor{
		config:    *config,
		repo:      repo,
		outbox:    outbox,
		lease:     l,
		leaseName: "failure-monitor-" + config.ServiceName,
	}, nil
}

// Start launches the background scan goroutine. The provided context is used as the parent for all scan operations; cancelling it (or calling Stop) shuts down the loop.
func (m *FailureMonitor) Start(ctx context.Context) error {
	ctx = appctx.WithNoTrace(ctx)
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wg.Add(1)
	go m.scanLoop()

	slog.Info("Message failure monitor started",
		"scan_interval", m.config.ScanInterval,
		"crash_stuck_minutes", m.config.CrashStuckMinutes,
		"recipient", m.config.Recipient,
	)

	return nil
}

// Stop cancels the background context and blocks until the scan goroutine has exited.
func (m *FailureMonitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	slog.Info("Message failure monitor stopped")
}

func (m *FailureMonitor) scanLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			_ = m.lease.WithLease(m.ctx, m.leaseName, m.config.LeaseTTL, func(ctx context.Context) error {
				m.scan(ctx)
				return nil
			})
		}
	}
}

func (m *FailureMonitor) scan(ctx context.Context) {
	inboxFailures, err := m.repo.ListUnalertedInboxFailures(ctx, m.config.CrashStuckMinutes, m.config.BatchSize)
	if err != nil {
		slog.Error("Failure monitor: failed to list inbox failures", "error", err)
		return
	}

	outboxFailures, err := m.repo.ListUnalertedOutboxFailures(ctx, m.config.BatchSize)
	if err != nil {
		slog.Error("Failure monitor: failed to list outbox failures", "error", err)
		return
	}

	if len(inboxFailures) == 0 && len(outboxFailures) == 0 {
		return
	}

	// Development runs still stamp alerted_at (below) so the same rows are not rescanned every tick, but do not send email — mirrors the 5xx alert's dev suppression.
	if m.config.PlatformMode != constants.PlatformModeDevelopment {
		if err := m.publishAlert(ctx, inboxFailures, outboxFailures); err != nil {
			// Leave alerted_at unset so the next scan retries the alert rather than silently dropping these failures.
			slog.Error("Failure monitor: failed to enqueue alert email", "error", err)
			return
		}
	}

	m.markAlerted(ctx, inboxFailures, outboxFailures)

	slog.Warn("Message failure monitor detected failed messages",
		"inbox_failures", len(inboxFailures),
		"outbox_failures", len(outboxFailures),
	)
}

func (m *FailureMonitor) publishAlert(ctx context.Context, inboxFailures []InboxFailure, outboxFailures []OutboxFailure) error {
	params := map[string]any{
		"InboxCount":     len(inboxFailures),
		"OutboxCount":    len(outboxFailures),
		"InboxFailures":  inboxFailureParams(inboxFailures),
		"OutboxFailures": outboxFailureParams(outboxFailures),
		"GeneratedAt":    time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}

	emailData := EmailSendData{
		To:         []string{m.config.Recipient},
		Subject:    fmt.Sprintf("[Message Failure Alert] %d inbox, %d outbox failures", len(inboxFailures), len(outboxFailures)),
		TemplateID: constants.EmailTemplateMessageFailureAlert,
		Params:     params,
	}

	emailJSON, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("marshal alert email: %w", err)
	}

	input := OutboxMessageInput{
		ServiceName: m.config.ServiceName,
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     contracts.AmqpMessage{Data: emailJSON},
	}

	return WithOutboxDBLockRetry(ctx, OutboxDBRetryConfig(m.config.PlatformMode), "failure_monitor.alert.create", func() error {
		_, err := m.outbox.Create(ctx, input)
		return err
	})
}

// markAlerted stamps alerted_at on every row that was included in the alert. Failures here are logged but not fatal: a re-alert on the next scan is preferable to losing the dedup marker.
func (m *FailureMonitor) markAlerted(ctx context.Context, inboxFailures []InboxFailure, outboxFailures []OutboxFailure) {
	if len(inboxFailures) > 0 {
		ids := make([]int64, len(inboxFailures))
		for i, f := range inboxFailures {
			ids[i] = f.ID
		}
		if err := m.repo.MarkInboxAlerted(ctx, ids); err != nil {
			slog.Error("Failure monitor: failed to mark inbox rows alerted", "error", err)
		}
	}

	if len(outboxFailures) > 0 {
		ids := make([]int64, len(outboxFailures))
		for i, f := range outboxFailures {
			ids[i] = f.ID
		}
		if err := m.repo.MarkOutboxAlerted(ctx, ids); err != nil {
			slog.Error("Failure monitor: failed to mark outbox rows alerted", "error", err)
		}
	}
}

func inboxFailureParams(failures []InboxFailure) []map[string]any {
	out := make([]map[string]any, 0, len(failures))
	for _, f := range failures {
		out = append(out, map[string]any{
			"MessageID":   f.MessageID,
			"ServiceName": f.ServiceName,
			"Handler":     f.Handler,
			"MessageType": f.MessageType,
			"Status":      string(f.Status),
			"Attempts":    f.Attempts,
			"Error":       truncateError(f.LastError),
			"ReceivedAt":  f.ReceivedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		})
	}
	return out
}

func outboxFailureParams(failures []OutboxFailure) []map[string]any {
	out := make([]map[string]any, 0, len(failures))
	for _, f := range failures {
		out = append(out, map[string]any{
			"MessageID":   f.MessageID,
			"ServiceName": f.ServiceName,
			"MessageType": f.MessageType,
			"RoutingKey":  f.RoutingKey,
			"Attempts":    f.Attempts,
			"MaxAttempts": f.MaxAttempts,
			"Error":       truncateError(f.LastError),
			"CreatedAt":   f.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		})
	}
	return out
}

// truncateError renders a nullable last_error for the email, standing in a placeholder for the crash-stuck case (no recorded error) and bounding length so one verbose stack trace cannot bloat the digest.
func truncateError(errMsg *string) string {
	if errMsg == nil || *errMsg == "" {
		return "(no error recorded — likely crash-stuck)"
	}
	msg := strings.TrimSpace(*errMsg)
	if len(msg) > maxFailureErrorLen {
		return msg[:maxFailureErrorLen] + "…"
	}
	return msg
}
