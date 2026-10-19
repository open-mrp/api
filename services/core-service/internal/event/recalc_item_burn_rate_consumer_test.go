package event

import (
	"context"
	"encoding/json"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

func newRecalcConsumerTest(t *testing.T) (*RecalcItemBurnRateConsumer, *repositorymock.MockItemRepo, *stubTxManager) {
	t.Helper()
	ctrl := gomock.NewController(t)
	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	tx := &stubTxManager{factory: repos}
	return &RecalcItemBurnRateConsumer{
		repos:     repos,
		txManager: tx,
		tracer:    tracing.GetTracer("test.recalc_item_burn_rate_consumer"),
	}, itemRepo, tx
}

func recalcDelivery(t *testing.T, evt domain.RecalcItemBurnRateEvent) amqp.Delivery {
	t.Helper()
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	body, err := json.Marshal(contracts.AmqpMessage{Data: data})
	require.NoError(t, err)
	return amqp.Delivery{Body: body}
}

func expectIdleItem(itemRepo *repositorymock.MockItemRepo, itemID string) {
	itemRepo.EXPECT().Get(gomock.Any(), domain.GetItemParams{AccountID: "ac_burn", ItemID: itemID}).
		Return(&domain.Item{ID: itemID, ItemCategoryID: "cat_1", BurnRateID: "rate_" + itemID}, nil)
	itemRepo.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "cat_1").Return("unit_base", "", nil)
	itemRepo.EXPECT().ListConsumptionChangeLogsForBurnRate(gomock.Any(), "ac_burn", itemID).Return(nil, nil)
	itemRepo.EXPECT().MarkBurnRatesComputed(gomock.Any(), []string{itemID}).Return(nil)
}

func TestRecalcConsumerRecomputesEachBatchedItemInItsOwnTransaction(t *testing.T) {
	t.Parallel()
	c, itemRepo, tx := newRecalcConsumerTest(t)
	expectIdleItem(itemRepo, "itm_a")
	expectIdleItem(itemRepo, "itm_b")

	require.NoError(t, c.handleMessage(context.Background(), recalcDelivery(t, domain.RecalcItemBurnRateEvent{AccountID: "ac_burn", ItemIDs: []string{"itm_a", "itm_b"}})))
	require.Equal(t, 2, tx.calls)
}

func TestRecalcConsumerSkipsAnItemDeletedSinceTheSweep(t *testing.T) {
	t.Parallel()
	c, itemRepo, _ := newRecalcConsumerTest(t)
	itemRepo.EXPECT().Get(gomock.Any(), domain.GetItemParams{AccountID: "ac_burn", ItemID: "itm_gone"}).
		Return(nil, apierror.NewResourceNotFoundError("No such item."))
	expectIdleItem(itemRepo, "itm_b")

	require.NoError(t, c.handleMessage(context.Background(), recalcDelivery(t, domain.RecalcItemBurnRateEvent{AccountID: "ac_burn", ItemIDs: []string{"itm_gone", "itm_b"}})))
}

func TestRecalcConsumerReturnsOtherFailuresForARetry(t *testing.T) {
	t.Parallel()
	c, itemRepo, _ := newRecalcConsumerTest(t)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, apierror.NewInternalError(nil, "down"))

	require.Error(t, c.handleMessage(context.Background(), recalcDelivery(t, domain.RecalcItemBurnRateEvent{AccountID: "ac_burn", ItemIDs: []string{"itm_a", "itm_b"}})))
}

func TestRecalcConsumerStillHandlesASingleItemCommand(t *testing.T) {
	t.Parallel()
	c, itemRepo, _ := newRecalcConsumerTest(t)
	expectIdleItem(itemRepo, "itm_a")

	require.NoError(t, c.handleMessage(context.Background(), recalcDelivery(t, domain.RecalcItemBurnRateEvent{AccountID: "ac_burn", ItemID: "itm_a"})))
}
