package messaging

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
)

type fakeFailureRepo struct {
	inbox  []InboxFailure
	outbox []OutboxFailure

	markedInbox  []int64
	markedOutbox []int64
}

func (f *fakeFailureRepo) ListUnalertedInboxFailures(context.Context, int, int32) ([]InboxFailure, error) {
	return f.inbox, nil
}

func (f *fakeFailureRepo) ListUnalertedOutboxFailures(context.Context, int32) ([]OutboxFailure, error) {
	return f.outbox, nil
}

func (f *fakeFailureRepo) MarkInboxAlerted(_ context.Context, ids []int64) error {
	f.markedInbox = append(f.markedInbox, ids...)
	return nil
}

func (f *fakeFailureRepo) MarkOutboxAlerted(_ context.Context, ids []int64) error {
	f.markedOutbox = append(f.markedOutbox, ids...)
	return nil
}

type fakeOutboxRepo struct {
	mu      sync.Mutex
	creates []OutboxMessageInput
}

func (f *fakeOutboxRepo) Create(_ context.Context, input OutboxMessageInput) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates = append(f.creates, input)
	return int64(len(f.creates)), nil
}

func newTestMonitor(t *testing.T, mode constants.PlatformMode, repo FailureMonitorRepo, outbox OutboxRepo) *FailureMonitor {
	t.Helper()
	cfg := (&FailureMonitorConfig{ServiceName: "platform-service", PlatformMode: mode}).WithDefaults()
	return &FailureMonitor{config: *cfg, repo: repo, outbox: outbox}
}

func TestFailureMonitorScan_NoFailures_NoEmail(t *testing.T) {
	repo := &fakeFailureRepo{}
	outbox := &fakeOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	if len(outbox.creates) != 0 {
		t.Fatalf("expected no email when there are no failures, got %d", len(outbox.creates))
	}
	if len(repo.markedInbox) != 0 || len(repo.markedOutbox) != 0 {
		t.Fatalf("expected nothing marked alerted, got inbox=%v outbox=%v", repo.markedInbox, repo.markedOutbox)
	}
}

func TestFailureMonitorScan_Failures_EmailsAndMarksAlerted(t *testing.T) {
	errMsg := "handler blew up"
	repo := &fakeFailureRepo{
		inbox:  []InboxFailure{{ID: 11, Handler: "notification.send_email", MessageType: "notification.cmd.send_email", LastError: &errMsg, ReceivedAt: time.Unix(0, 0)}},
		outbox: []OutboxFailure{{ID: 22, MessageType: "sales_order.created", RoutingKey: "sales_order.created", Attempts: 25, MaxAttempts: 25, CreatedAt: time.Unix(0, 0)}},
	}
	outbox := &fakeOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	if len(outbox.creates) != 1 {
		t.Fatalf("expected exactly one alert email, got %d", len(outbox.creates))
	}
	got := outbox.creates[0]
	if got.MessageType != "notification.cmd.send_email" {
		t.Fatalf("alert should route as a send-email command, got %q", got.MessageType)
	}
	if len(repo.markedInbox) != 1 || repo.markedInbox[0] != 11 {
		t.Fatalf("expected inbox row 11 marked alerted, got %v", repo.markedInbox)
	}
	if len(repo.markedOutbox) != 1 || repo.markedOutbox[0] != 22 {
		t.Fatalf("expected outbox row 22 marked alerted, got %v", repo.markedOutbox)
	}
}

func TestFailureMonitorScan_DevMode_MarksButDoesNotEmail(t *testing.T) {
	repo := &fakeFailureRepo{
		outbox: []OutboxFailure{{ID: 22, MessageType: "sales_order.created", Attempts: 25, MaxAttempts: 25, CreatedAt: time.Unix(0, 0)}},
	}
	outbox := &fakeOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeDevelopment, repo, outbox).scan(context.Background())

	if len(outbox.creates) != 0 {
		t.Fatalf("development mode must not send alert email, got %d", len(outbox.creates))
	}
	if len(repo.markedOutbox) != 1 {
		t.Fatalf("rows should still be marked alerted in dev mode so they are not rescanned, got %v", repo.markedOutbox)
	}
}

// stubFailureRepo lets each scan test choose which query fails and which mark fails.
type stubFailureRepo struct {
	mu sync.Mutex

	inbox     []InboxFailure
	inboxErr  error
	outbox    []OutboxFailure
	outboxErr error

	markInboxErr  error
	markOutboxErr error

	inboxCalls   int
	outboxCalls  int
	markedInbox  []int64
	markedOutbox []int64
}

func (s *stubFailureRepo) ListUnalertedInboxFailures(context.Context, int, int32) ([]InboxFailure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inboxCalls++
	return s.inbox, s.inboxErr
}

func (s *stubFailureRepo) ListUnalertedOutboxFailures(context.Context, int32) ([]OutboxFailure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outboxCalls++
	return s.outbox, s.outboxErr
}

func (s *stubFailureRepo) MarkInboxAlerted(_ context.Context, ids []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markedInbox = append(s.markedInbox, ids...)
	return s.markInboxErr
}

func (s *stubFailureRepo) MarkOutboxAlerted(_ context.Context, ids []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markedOutbox = append(s.markedOutbox, ids...)
	return s.markOutboxErr
}

// stubOutboxRepo fails every enqueue attempt with a non-lock error so WithOutboxDBLockRetry returns immediately.
type stubOutboxRepo struct {
	mu      sync.Mutex
	err     error
	creates int
}

func (s *stubOutboxRepo) Create(context.Context, OutboxMessageInput) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creates++
	if s.err != nil {
		return 0, s.err
	}
	return int64(s.creates), nil
}

func testInboxFailure() InboxFailure {
	return InboxFailure{ID: 11, MessageType: "sales_order.created", ReceivedAt: time.Unix(0, 0)}
}

func testOutboxFailure() OutboxFailure {
	return OutboxFailure{ID: 22, MessageType: "sales_order.created", Attempts: 25, MaxAttempts: 25, CreatedAt: time.Unix(0, 0)}
}

func TestFailureMonitorScan_InboxQueryError_SuppressesEntireScan(t *testing.T) {
	t.Parallel()

	repo := &stubFailureRepo{
		inboxErr: errors.New("query failed"),
		outbox:   []OutboxFailure{testOutboxFailure()},
	}
	outbox := &stubOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	// The inbox query is first and returns early, so a failure there also stops outbox failures being alerted on.
	if repo.outboxCalls != 0 {
		t.Fatalf("expected outbox query to be skipped, got %d calls", repo.outboxCalls)
	}
	if outbox.creates != 0 {
		t.Fatalf("expected no alert email, got %d", outbox.creates)
	}
	if len(repo.markedOutbox) != 0 {
		t.Fatalf("expected nothing marked alerted, got %v", repo.markedOutbox)
	}
}

func TestFailureMonitorScan_OutboxQueryError_NoAlert(t *testing.T) {
	t.Parallel()

	repo := &stubFailureRepo{
		inbox:     []InboxFailure{testInboxFailure()},
		outboxErr: errors.New("query failed"),
	}
	outbox := &stubOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	if outbox.creates != 0 {
		t.Fatalf("expected no alert email, got %d", outbox.creates)
	}
	if len(repo.markedInbox) != 0 {
		t.Fatalf("expected nothing marked alerted, got %v", repo.markedInbox)
	}
}

func TestFailureMonitorScan_AlertEnqueueFails_LeavesRowsUnalerted(t *testing.T) {
	t.Parallel()

	repo := &stubFailureRepo{
		inbox:  []InboxFailure{testInboxFailure()},
		outbox: []OutboxFailure{testOutboxFailure()},
	}
	outbox := &stubOutboxRepo{err: errors.New("insert failed")}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	// alerted_at must stay unset so the next scan re-attempts the alert instead of dropping these failures.
	if len(repo.markedInbox) != 0 || len(repo.markedOutbox) != 0 {
		t.Fatalf("expected nothing marked alerted after a failed enqueue, got inbox=%v outbox=%v", repo.markedInbox, repo.markedOutbox)
	}
}

func TestFailureMonitorScan_MarkAlertedErrors_AreNotFatal(t *testing.T) {
	t.Parallel()

	repo := &stubFailureRepo{
		inbox:         []InboxFailure{testInboxFailure()},
		outbox:        []OutboxFailure{testOutboxFailure()},
		markInboxErr:  errors.New("mark failed"),
		markOutboxErr: errors.New("mark failed"),
	}
	outbox := &stubOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeProduction, repo, outbox).scan(context.Background())

	if outbox.creates != 1 {
		t.Fatalf("expected the alert email to be sent, got %d", outbox.creates)
	}
	// A failed inbox mark must not abort the outbox mark.
	if len(repo.markedInbox) != 1 || len(repo.markedOutbox) != 1 {
		t.Fatalf("expected both marks attempted, got inbox=%v outbox=%v", repo.markedInbox, repo.markedOutbox)
	}
}

func TestFailureMonitorScan_TestMode_SendsEmail(t *testing.T) {
	t.Parallel()

	repo := &stubFailureRepo{outbox: []OutboxFailure{testOutboxFailure()}}
	outbox := &stubOutboxRepo{}

	newTestMonitor(t, constants.PlatformModeTest, repo, outbox).scan(context.Background())

	// Only development mode suppresses the email; e2e suites assert on the alert.
	if outbox.creates != 1 {
		t.Fatalf("expected test mode to send the alert email, got %d", outbox.creates)
	}
	if len(repo.markedOutbox) != 1 {
		t.Fatalf("expected the row to be marked alerted, got %v", repo.markedOutbox)
	}
}

func TestTruncateError(t *testing.T) {
	t.Parallel()

	placeholder := "(no error recorded — likely crash-stuck)"
	empty := ""
	short := "  boom  "
	exact := strings.Repeat("a", maxFailureErrorLen)
	long := strings.Repeat("a", maxFailureErrorLen+100)

	tests := []struct {
		name string
		in   *string
		want string
	}{
		{name: "nil", in: nil, want: placeholder},
		{name: "empty", in: &empty, want: placeholder},
		{name: "trimmed", in: &short, want: "boom"},
		{name: "exactly the limit", in: &exact, want: exact},
		{name: "over the limit", in: &long, want: strings.Repeat("a", maxFailureErrorLen) + "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := truncateError(tt.in); got != tt.want {
				t.Errorf("truncateError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFailureMonitorConfigWithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           *FailureMonitorConfig
		wantScanInterval time.Duration
	}{
		{name: "nil config", config: nil, wantScanInterval: 5 * time.Minute},
		{name: "production", config: &FailureMonitorConfig{PlatformMode: constants.PlatformModeProduction}, wantScanInterval: 5 * time.Minute},
		{name: "test mode scans faster", config: &FailureMonitorConfig{PlatformMode: constants.PlatformModeTest}, wantScanInterval: time.Minute},
		{name: "explicit interval wins", config: &FailureMonitorConfig{PlatformMode: constants.PlatformModeTest, ScanInterval: 42 * time.Second}, wantScanInterval: 42 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.config.WithDefaults()
			if got.ScanInterval != tt.wantScanInterval {
				t.Errorf("ScanInterval = %v, want %v", got.ScanInterval, tt.wantScanInterval)
			}
			if got.Recipient != defaultFailureAlertRecipient {
				t.Errorf("Recipient = %q, want %q", got.Recipient, defaultFailureAlertRecipient)
			}
			if got.CrashStuckMinutes != 30 {
				t.Errorf("CrashStuckMinutes = %d, want 30", got.CrashStuckMinutes)
			}
			if got.BatchSize != 100 {
				t.Errorf("BatchSize = %d, want 100", got.BatchSize)
			}
			if got.LeaseTTL != 5*time.Minute {
				t.Errorf("LeaseTTL = %v, want 5m", got.LeaseTTL)
			}
		})
	}
}

func TestFailureMonitorConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  FailureMonitorConfig
		wantErr string
	}{
		{name: "valid", config: FailureMonitorConfig{ServiceName: "platform-service", ScanInterval: time.Minute, CrashStuckMinutes: 30, BatchSize: 100}},
		{name: "missing service name", config: FailureMonitorConfig{ScanInterval: time.Minute, CrashStuckMinutes: 30, BatchSize: 100}, wantErr: "service name is required"},
		{name: "non-positive scan interval", config: FailureMonitorConfig{ServiceName: "s", ScanInterval: -time.Minute, CrashStuckMinutes: 30, BatchSize: 100}, wantErr: "scan interval must be positive"},
		{name: "non-positive crash-stuck minutes", config: FailureMonitorConfig{ServiceName: "s", ScanInterval: time.Minute, CrashStuckMinutes: -1, BatchSize: 100}, wantErr: "crash-stuck minutes must be positive"},
		{name: "non-positive batch size", config: FailureMonitorConfig{ServiceName: "s", ScanInterval: time.Minute, CrashStuckMinutes: 30, BatchSize: -1}, wantErr: "batch size must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewFailureMonitorRequiresDependencies(t *testing.T) {
	t.Parallel()

	validConfig := func() *FailureMonitorConfig {
		return &FailureMonitorConfig{ServiceName: "platform-service"}
	}

	tests := []struct {
		name    string
		config  *FailureMonitorConfig
		repo    FailureMonitorRepo
		outbox  OutboxRepo
		lease   *lease.Lease
		wantErr string
	}{
		{name: "missing service name", config: &FailureMonitorConfig{}, repo: &stubFailureRepo{}, outbox: &stubOutboxRepo{}, lease: testLease(), wantErr: "service name is required"},
		{name: "nil repo", config: validConfig(), outbox: &stubOutboxRepo{}, lease: testLease(), wantErr: "repo is required"},
		{name: "nil outbox", config: validConfig(), repo: &stubFailureRepo{}, lease: testLease(), wantErr: "outbox is required"},
		{name: "nil lease", config: validConfig(), repo: &stubFailureRepo{}, outbox: &stubOutboxRepo{}, wantErr: "lease is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			monitor, err := NewFailureMonitor(tt.config, tt.repo, tt.outbox, tt.lease)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
			if monitor != nil {
				t.Error("expected no monitor when construction fails")
			}
		})
	}
}

func TestNewFailureMonitorScopesLeaseToService(t *testing.T) {
	t.Parallel()

	monitor, err := NewFailureMonitor(&FailureMonitorConfig{ServiceName: "platform-service"}, &stubFailureRepo{}, &stubOutboxRepo{}, testLease())
	if err != nil {
		t.Fatalf("expected construction to succeed, got %v", err)
	}
	if monitor.leaseName != "failure-monitor-platform-service" {
		t.Errorf("lease name = %q, want failure-monitor-platform-service", monitor.leaseName)
	}
}
