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

func TestBurnRateSweepEnqueuesStaleItems(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	outbox := &recordingOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	itemRepo.EXPECT().
		ListStaleBurnRateItems(gomock.Any(), gomock.Any(), int32(2)).
		Return([]domain.StaleBurnRateItem{
			{ItemID: "item-1", AccountID: "acct-1"},
			{ItemID: "item-2", AccountID: "acct-2"},
		}, nil)

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 2}
	s.enqueueStaleRecalcs(context.Background())

	events := recalcEvents(t, outbox)
	require.Len(t, events, 2)
	assert.Equal(t, domain.RecalcItemBurnRateEvent{AccountID: "acct-1", ItemID: "item-1"}, events[0])
	assert.Equal(t, domain.RecalcItemBurnRateEvent{AccountID: "acct-2", ItemID: "item-2"}, events[1])
}

func TestBurnRateSweepPassesStaleThreshold(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()

	var gotStaleBefore time.Time
	itemRepo.EXPECT().
		ListStaleBurnRateItems(gomock.Any(), gomock.Any(), int32(500)).
		DoAndReturn(func(_ context.Context, staleBefore time.Time, _ int32) ([]domain.StaleBurnRateItem, *apierror.APIError) {
			gotStaleBefore = staleBefore
			return nil, nil
		})

	s := &burnRateScheduler{repos: repos, staleThreshold: time.Hour, batchSize: 500}
	before := time.Now().UTC().Add(-time.Hour)
	s.enqueueStaleRecalcs(context.Background())

	// The cutoff is "now minus the threshold"; allow a small window for the elapsed test time.
	assert.WithinDuration(t, before, gotStaleBefore, 5*time.Second)
}

func TestBurnRateSweepListErrorDoesNotEnqueue(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	outbox := &recordingOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	// NewOutboxRepo must never be reached when the listing fails.
	repos.EXPECT().NewOutboxRepo().Return(outbox).Times(0)

	itemRepo.EXPECT().
		ListStaleBurnRateItems(gomock.Any(), gomock.Any(), int32(500)).
		Return(nil, apierror.NewInternalError(errors.New("boom"), "list failed"))

	s := &burnRateScheduler{repos: repos, staleThreshold: 24 * time.Hour, batchSize: 500}
	s.enqueueStaleRecalcs(context.Background())

	assert.Empty(t, outbox.messages)
}
