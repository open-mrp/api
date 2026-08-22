package event

import (
	"context"
	"log/slog"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// maxCostGraphDepth bounds how far downstream of a change costs are recomputed.
//
// The walk already refuses to revisit an item, so this is not what stops a cycle; it stops a
// pathological routing from turning one price change into an unbounded amount of work. A bill of
// materials deeper than this is not something the costing model represents usefully anyway.
const maxCostGraphDepth = 12

// ItemCostBasisChangedConsumer recomputes the cost of everything downstream of a change.
//
// A cost is only as current as the inputs it was last calculated from, so moving a material's price
// or editing a production step leaves every part and product built on it stale. This walks the
// production graph outwards from the change and recalculates what it reaches.
//
// It replaces a nightly job that recomputed every item with a production, for one hardcoded account,
// whether or not anything had changed — which meant a price change entered on Tuesday was wrong on
// every quote until Wednesday morning, and right by accident rather than by design.
type ItemCostBasisChangedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	itemSvc       domain.ItemSvc
	tracer        trace.Tracer
}

func NewItemCostBasisChangedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	itemSvc domain.ItemSvc,
) *ItemCostBasisChangedConsumer {
	return &ItemCostBasisChangedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		itemSvc:       itemSvc,
		tracer:        tracing.GetTracer("core-service.item_cost_basis_changed_consumer"),
	}
}

func (c *ItemCostBasisChangedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventItemCostBasisChangedQueue,
		c.inboxConsumer.Wrap("core.item_cost_basis_changed", c.handleMessage))
}

func (c *ItemCostBasisChangedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.item_cost_basis_changed",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attribute.String("messaging.message_id", msg.MessageId)),
	)
	defer span.End()

	evt, accountID, err := decodeEvent[domain.ItemCostBasisChangedEvent](msg, func(e domain.ItemCostBasisChangedEvent) string {
		return e.AccountID
	})
	if err != nil {
		span.RecordError(err)
		return err
	}
	if accountID == "" || evt.ItemID == "" {
		slog.ErrorContext(ctx, "item_cost_basis_changed: incomplete event",
			"account_id", accountID, "item_id", evt.ItemID)
		return nil
	}

	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("item.id", evt.ItemID),
		attribute.String("event.reason", evt.Reason),
	)

	affected, apiErr := c.affectedItems(ctx, accountID, evt.ItemID)
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	span.SetAttributes(attribute.Int("items.recomputed", len(affected)))

	// Each item is costed and committed on its own. They are independent calculations, and a routing
	// that breaks one — a step whose flow cannot be resolved — should not withhold the corrected cost
	// from every other item the change touched.
	var failed int
	for _, itemID := range affected {
		if _, apiErr := c.itemSvc.RecomputeItemCosts(ctx, accountID, itemID); apiErr != nil {
			failed++
			slog.WarnContext(ctx, "item_cost_basis_changed: could not recost item",
				"item_id", itemID, "error", apiErr.PublicMessage)
		}
	}

	if failed > 0 {
		slog.WarnContext(ctx, "item_cost_basis_changed: some items could not be recosted",
			"account_id", accountID, "origin_item_id", evt.ItemID,
			"recomputed", len(affected)-failed, "failed", failed)
	}
	return nil
}

// affectedItems is the changed item followed by everything downstream of it, nearest first.
//
// The walk goes a generation at a time: the items produced by steps consuming what has already been
// reached, then the items produced by steps consuming those. `seen` keeps it from revisiting an item
// a second routing also feeds, which is both the dedup and the cycle guard.
//
// Order matters. A part's cost is an input to the product built from it, so recosting nearest-first
// means each item is calculated from inputs this same pass has already corrected, rather than from
// values the change has just invalidated.
func (c *ItemCostBasisChangedConsumer) affectedItems(ctx context.Context, accountID, originItemID string) ([]string, *apierror.APIError) {
	itemRepo := c.repos.NewItemRepo()

	ordered := []string{originItemID}
	seen := map[string]bool{originItemID: true}
	frontier := []string{originItemID}

	for depth := 0; depth < maxCostGraphDepth && len(frontier) > 0; depth++ {
		produced, apiErr := itemRepo.FindItemsProducedFromConsumed(ctx, accountID, frontier)
		if apiErr != nil {
			return nil, apiErr
		}

		var next []string
		for _, itemID := range produced {
			if itemID == "" || seen[itemID] {
				continue
			}
			seen[itemID] = true
			ordered = append(ordered, itemID)
			next = append(next, itemID)
		}
		frontier = next
	}

	return ordered, nil
}
