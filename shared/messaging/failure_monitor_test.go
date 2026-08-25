package messaging

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/constants"
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
