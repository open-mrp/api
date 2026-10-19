package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

var sweepNow = time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

func computedAt(t time.Time) *time.Time { return &t }

func newSweeper(repos domain.RepoFactory) *burnRateScheduler {
	return &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 500, now: func() time.Time { return sweepNow }}
}

func TestBurnRateSweepEnqueuesNeverComputedItemsInOneMessagePerAccount(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().ListStaleBurnRateItems(gomock.Any(), sweepNow.Add(-24*time.Hour), int32(500)).Return([]domain.StaleBurnRateItem{
		{ItemID: "item-1", AccountID: "ac_one"},
		{ItemID: "item-2", AccountID: "ac_two"},
		{ItemID: "item-3", AccountID: "ac_one"},
	}, nil)

	newSweeper(repos).enqueueStaleRecalcs(context.Background())

	assert.Equal(t, []domain.RecalcItemBurnRateEvent{
		{AccountID: "ac_one", ItemIDs: []string{"item-1", "item-3"}},
		{AccountID: "ac_two", ItemIDs: []string{"item-2"}},
	}, recalcEvents(t, outbox))
}

func TestBurnRateSweepRecomputesOnlyItemsWithLogsLeavingTheWindow(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	lastWeek := sweepNow.AddDate(0, 0, -7)
	twoDaysAgo := sweepNow.AddDate(0, 0, -2)
	itemRepo.EXPECT().ListStaleBurnRateItems(gomock.Any(), gomock.Any(), gomock.Any()).Return([]domain.StaleBurnRateItem{
		{ItemID: "itm_idle", AccountID: "ac_one", ComputedAt: computedAt(lastWeek)},
		{ItemID: "itm_sliding", AccountID: "ac_one", ComputedAt: computedAt(twoDaysAgo)},
		{ItemID: "itm_recent", AccountID: "ac_one", ComputedAt: computedAt(twoDaysAgo)},
	}, nil)
	itemRepo.EXPECT().OldestChangeLogsSince(gomock.Any(), "ac_one", []string{"itm_idle", "itm_sliding", "itm_recent"}, lastWeek.Add(-burnRateWindow)).
		Return(map[string]time.Time{
			"itm_sliding": sweepNow.AddDate(0, 0, -31),
			"itm_recent":  sweepNow.AddDate(0, 0, -3),
		}, nil)
	itemRepo.EXPECT().MarkBurnRatesComputed(gomock.Any(), []string{"itm_idle", "itm_recent"}).Return(nil)

	newSweeper(repos).enqueueStaleRecalcs(context.Background())

	assert.Equal(t, []domain.RecalcItemBurnRateEvent{{AccountID: "ac_one", ItemIDs: []string{"itm_sliding"}}}, recalcEvents(t, outbox))
}

func TestBurnRateSweepSplitsAnAccountIntoBoundedMessages(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	var items []domain.StaleBurnRateItem
	for i := range burnRateSweepMessageItems + 1 {
		items = append(items, domain.StaleBurnRateItem{ItemID: fmt.Sprintf("itm_%03d", i), AccountID: "ac_one"})
	}
	itemRepo.EXPECT().ListStaleBurnRateItems(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

	newSweeper(repos).enqueueStaleRecalcs(context.Background())

	events := recalcEvents(t, outbox)
	require.Len(t, events, 2)
	assert.Len(t, events[0].ItemIDs, burnRateSweepMessageItems)
	assert.Equal(t, []string{"itm_100"}, events[1].ItemIDs)
}

func TestBurnRateSweepChangeLogErrorSkipsOnlyThatAccount(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().ListStaleBurnRateItems(gomock.Any(), gomock.Any(), gomock.Any()).Return([]domain.StaleBurnRateItem{
		{ItemID: "itm_a", AccountID: "ac_one", ComputedAt: computedAt(sweepNow.AddDate(0, 0, -2))},
		{ItemID: "itm_b", AccountID: "ac_two"},
	}, nil)
	itemRepo.EXPECT().OldestChangeLogsSince(gomock.Any(), "ac_one", gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("boom"), "read failed"))

	newSweeper(repos).enqueueStaleRecalcs(context.Background())

	assert.Equal(t, []domain.RecalcItemBurnRateEvent{{AccountID: "ac_two", ItemIDs: []string{"itm_b"}}}, recalcEvents(t, outbox))
}

func TestBurnRateSweepListErrorDoesNotEnqueue(t *testing.T) {
	t.Parallel()
	itemRepo, outbox, repos := newSweepTest(t)
	itemRepo.EXPECT().ListStaleBurnRateItems(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("boom"), "list failed"))

	newSweeper(repos).enqueueStaleRecalcs(context.Background())

	assert.Empty(t, outbox.messages)
}
