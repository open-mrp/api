package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
)

func recalcEvents(t *testing.T, outbox *recordingOutboxRepo) []domain.RecalcItemBurnRateEvent {
	t.Helper()
	var events []domain.RecalcItemBurnRateEvent
	for _, msg := range outbox.messages {
		if msg.MessageType != string(contracts.CoreCmdRecalcItemBurnRate) {
			continue
		}
		var evt domain.RecalcItemBurnRateEvent
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &evt))
		events = append(events, evt)
	}
	return events
}

func newSweepTest(t *testing.T) (*repositorymock.MockItemRepo, *recordingOutboxRepo, *factorymock.MockRepoFactory) {
	t.Helper()
	ctrl := gomock.NewController(t)
	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	outbox := &recordingOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()
	return itemRepo, outbox, repos
}

func staleItems(ids ...string) []domain.StaleBurnRateItem {
	out := make([]domain.StaleBurnRateItem, len(ids))
	for i, id := range ids {
		out[i] = domain.StaleBurnRateItem{ItemID: id, AccountID: "acct-" + id}
	}
	return out
}

func enqueuedIDs(t *testing.T, outbox *recordingOutboxRepo) []string {
	t.Helper()
	var ids []string
	for _, e := range recalcEvents(t, outbox) {
		ids = append(ids, e.ItemID)
	}
	return ids
}

func TestBurnRateSweepEnqueuesStaleItems(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().ScanBurnRateItems(gomock.Any(), "", gomock.Any(), int32(250)).Return(staleItems("item-1", "item-2"), "", nil)

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 500, scanPage: 250}
	s.enqueueStaleRecalcs(context.Background())

	events := recalcEvents(t, outbox)
	require.Len(t, events, 2)
	assert.Equal(t, domain.RecalcItemBurnRateEvent{AccountID: "acct-item-1", ItemID: "item-1"}, events[0])
	assert.Equal(t, domain.RecalcItemBurnRateEvent{AccountID: "acct-item-2", ItemID: "item-2"}, events[1])
	assert.Empty(t, s.cursor, "a lap that reached the last item starts the next tick from the first")
}

func TestBurnRateSweepPassesStaleThreshold(t *testing.T) {
	t.Parallel()
	itemRepo, _, repos := newSweepTest(t)

	var gotStaleBefore time.Time
	itemRepo.EXPECT().
		ScanBurnRateItems(gomock.Any(), "", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, staleBefore time.Time, _ int32) ([]domain.StaleBurnRateItem, string, *apierror.APIError) {
			gotStaleBefore = staleBefore
			return nil, "", nil
		})

	s := &burnRateScheduler{repos: repos, staleThreshold: time.Hour, batchSize: 500, scanPage: 250}
	before := time.Now().UTC().Add(-time.Hour)
	s.enqueueStaleRecalcs(context.Background())

	// The cutoff is "now minus the threshold"; allow a small window for the elapsed test time.
	assert.WithinDuration(t, before, gotStaleBefore, 5*time.Second)
}

func TestBurnRateSweepListErrorDoesNotEnqueue(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().
		ScanBurnRateItems(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, "", apierror.NewInternalError(errors.New("boom"), "list failed"))

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 500, scanPage: 250}
	s.enqueueStaleRecalcs(context.Background())

	assert.Empty(t, outbox.messages)
}

func TestBurnRateSweepStopsAtTheBatchAndResumesAfterIt(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().ScanBurnRateItems(gomock.Any(), "", gomock.Any(), int32(3)).Return(staleItems("a", "b", "c"), "c", nil)

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 2, scanPage: 3}
	s.enqueueStaleRecalcs(context.Background())

	assert.Equal(t, []string{"a", "b"}, enqueuedIDs(t, outbox))
	assert.Equal(t, "b", s.cursor)
}

func TestBurnRateSweepWrapsOnceToWhereItStarted(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	gomock.InOrder(
		itemRepo.EXPECT().ScanBurnRateItems(gomock.Any(), "m", gomock.Any(), int32(2)).Return(staleItems("x"), "", nil),
		itemRepo.EXPECT().ScanBurnRateItems(gomock.Any(), "", gomock.Any(), int32(2)).Return(nil, "f", nil),
		itemRepo.EXPECT().ScanBurnRateItems(gomock.Any(), "f", gomock.Any(), int32(2)).Return(staleItems("g"), "n", nil),
	)

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 10, scanPage: 2, cursor: "m"}
	s.enqueueStaleRecalcs(context.Background())

	assert.Equal(t, []string{"x", "g"}, enqueuedIDs(t, outbox))
	assert.Equal(t, "n", s.cursor)
}
