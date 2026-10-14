package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	actOwnerUserID        = "us_owner"
	actOwnerAccountUserID = "acus_owner"
	actCreatorUserID      = "us_creator"
	actActorUserID        = "us_actor"
)

// orderWorld is what the audit trail says about order so_1 in acct_1.
type orderWorld struct {
	// creatorID / creatorType describe the create event; an empty creatorID means none is on record.
	creatorID   string
	creatorType string
	// createRequestID is the request that created the order.
	createRequestID string
	number          string
	// repAccountUserID is the latest recorded sales_rep_id ("" = never set or cleared); repUserID is the user behind it ("" = membership gone).
	repAccountUserID string
	repUserID        string
	// status is the latest recorded sales_order_status_code.
	status string
}

func defaultOrderWorld() *orderWorld {
	return &orderWorld{
		creatorID:        actCreatorUserID,
		creatorType:      "user",
		createRequestID:  "req_create",
		number:           "SO-1042",
		repAccountUserID: actOwnerAccountUserID,
		repUserID:        actOwnerUserID,
		status:           string(constants.SalesOrderStatusCodeIssued),
	}
}

type activityHarness struct {
	repo   *repositorymock.MockAuditEventRepo
	outbox *capturingOutboxRepo
	svc    domain.AuditEventSvc
}

// newActivityHarness wires the audit service over a repo that answers order lookups from world. A nil world
// expects no lookups at all.
func newActivityHarness(t *testing.T, world *orderWorld) *activityHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockAuditEventRepo(ctrl)
	outbox := &capturingOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAuditEventRepo().Return(repo).Times(1)
	factory.EXPECT().NewOutboxRepo().Return(outbox).Times(1)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	if world != nil {
		repo.EXPECT().
			GetResourceCreateEvent(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
			DoAndReturn(func(context.Context, string, constants.ObjectType, string) (*domain.ResourceCreateEvent, *apierror.APIError) {
				if world.creatorID == "" {
					return nil, nil
				}
				requestID := world.createRequestID
				return &domain.ResourceCreateEvent{
					ActorID:      world.creatorID,
					IdentityType: world.creatorType,
					RequestID:    &requestID,
					Changes: []domain.AuditFieldChange{
						{Field: "number", OldValue: json.RawMessage(`null`), NewValue: jsonString(world.number)},
					},
				}, nil
			}).AnyTimes()
		repo.EXPECT().
			GetLatestResourceFieldValue(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ constants.ObjectType, _ string, field string) (json.RawMessage, bool, *apierror.APIError) {
				switch field {
				case "sales_rep_id":
					if world.repAccountUserID == "" {
						return nil, false, nil
					}
					return jsonString(world.repAccountUserID), true, nil
				case "sales_order_status_code":
					return jsonString(world.status), true, nil
				}
				return nil, false, nil
			}).AnyTimes()
		repo.EXPECT().
			GetAccountUserUserID(gomock.Any(), "acct_1", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, accountUserID string) (string, *apierror.APIError) {
				if accountUserID == world.repAccountUserID {
					return world.repUserID, nil
				}
				return "", nil
			}).AnyTimes()
		repo.EXPECT().GetResourceCreateChanges(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	}

	return &activityHarness{repo: repo, outbox: outbox, svc: NewAuditEventSvc(&AuditEventSvcConfig{Repos: factory})}
}

// save persists event as actor and returns the order-activity fan-outs it enqueued.
func (h *activityHarness) save(t *testing.T, ctx context.Context, event *domain.AuditEvent) []messaging.AlertFanoutData {
	t.Helper()
	require.Nil(t, h.svc.SaveAuditEvent(ctx, event))
	fanouts := make([]messaging.AlertFanoutData, 0, len(h.outbox.inputs))
	for _, input := range h.outbox.inputs {
		require.Equal(t, string(contracts.NotificationCmdFanout), input.RoutingKey)
		var data messaging.AlertFanoutData
		require.NoError(t, json.Unmarshal(input.Payload.Data, &data))
		fanouts = append(fanouts, data)
	}
	return fanouts
}

func jsonString(s string) json.RawMessage {
	raw, _ := json.Marshal(s)
	return raw
}

func namedActorCtx(accountID, actorID, actorName string) context.Context {
	name := actorName
	acctID := accountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           actorID,
			Name:         &name,
			AccountID:    &acctID,
		},
	})
}

func systemActorCtx(accountID string, actor constants.SystemActor) context.Context {
	return appctx.WithIdentity(context.Background(), types.NewSystemIdentity(actor, accountID))
}

func change(field, oldValue, newValue string) domain.AuditFieldChange {
	return domain.AuditFieldChange{Field: field, OldValue: json.RawMessage(oldValue), NewValue: json.RawMessage(newValue)}
}

func orderEvent(actorID string, action constants.AuditAction, changes ...domain.AuditFieldChange) *domain.AuditEvent {
	requestID := "req_abc123"
	return &domain.AuditEvent{
		ID:               "aevt_so1",
		ActorID:          actorID,
		ActorType:        "internal",
		IdentityType:     "user",
		AccountID:        "acct_1",
		Action:           action,
		ResourceType:     constants.ObjectTypeSalesOrder,
		ResourceID:       "so_1",
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   "so_1",
		Changes:          changes,
		RequestID:        &requestID,
		OccurredAt:       time.Now().UTC(),
	}
}

func childEvent(actorID string, resourceType constants.ObjectType, resourceID string, action constants.AuditAction, changes ...domain.AuditFieldChange) *domain.AuditEvent {
	event := orderEvent(actorID, action, changes...)
	event.ID = "aevt_" + resourceID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	return event
}

func asSystem(event *domain.AuditEvent, actor constants.SystemActor) *domain.AuditEvent {
	event.ActorID = actor.ActorID()
	event.ActorType = "unassigned"
	event.IdentityType = string(types.IdentityActorTypeSystem)
	return event
}

func issueChanges() []domain.AuditFieldChange {
	return []domain.AuditFieldChange{
		change("sales_order_status_code", `"estimate"`, `"issued"`),
		change("issued_at", `null`, `"2026-08-10T12:00:00Z"`),
	}
}

func TestOrderActivity_MilestonesAlertTheOwner(t *testing.T) {
	cases := []struct {
		name  string
		ctx   context.Context
		event *domain.AuditEvent
		world func(*orderWorld)
		title string
		body  string
	}{
		{
			name:  "issued",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...),
			title: "Sales order SO-1042 updated",
			body:  "Blake Doe issued the order.",
		},
		{
			name: "fulfilled",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate,
				change("sales_order_status_code", `"issued"`, `"fulfilled"`),
				change("completed_at", `null`, `"2026-08-10T12:00:00Z"`),
				change("first_ship_at", `null`, `"2026-08-10T12:00:00Z"`)),
			body: "Blake Doe marked the order fulfilled.",
		},
		{
			name:  "shipped",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("first_ship_at", `null`, `"2026-08-10T12:00:00Z"`)),
			body:  "Blake Doe shipped the order.",
		},
		{
			name:  "shipment voided",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("first_ship_at", `"2026-08-10T12:00:00Z"`, `null`)),
			body:  "Blake Doe marked the order unshipped.",
		},
		{
			name: "reverted to estimate",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate,
				change("sales_order_status_code", `"issued"`, `"estimate"`),
				change("issued_at", `"2026-08-10T12:00:00Z"`, `null`)),
			world: func(w *orderWorld) { w.status = string(constants.SalesOrderStatusCodeEstimate) },
			body:  "Blake Doe reverted the order to an estimate.",
		},
		{
			name: "fulfilled back to issued",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate,
				change("sales_order_status_code", `"fulfilled"`, `"issued"`),
				change("completed_at", `"2026-08-10T12:00:00Z"`, `null`)),
			body: "Blake Doe issued the order.",
		},
		{
			name:  "deleted",
			event: orderEvent(actActorUserID, constants.AuditActionDelete, change("number", `"SO-1042"`, `null`)),
			world: func(w *orderWorld) { w.status = string(constants.SalesOrderStatusCodeEstimate) },
			title: "Sales order SO-1042 deleted",
			body:  "Blake Doe deleted the order.",
		},
		{
			name:  "promised date",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("promised_at", `"2026-08-01T00:00:00Z"`, `"2026-09-15T00:00:00Z"`)),
			body:  "Blake Doe changed the promised date to Sep 15, 2026.",
		},
		{
			name:  "ship-by date",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("ship_by_date", `"2026-08-01T00:00:00Z"`, `"2026-09-10T00:00:00Z"`)),
			body:  "Blake Doe changed the ship-by date to Sep 10, 2026.",
		},
		{
			name: "date change narrates the other fields of the same edit",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate,
				change("promised_at", `"2026-08-01T00:00:00Z"`, `"2026-09-15T00:00:00Z"`),
				change("carrier_id", `"car_1"`, `"car_2"`)),
			body: "Blake Doe changed the promised date to Sep 15, 2026 and changed the carrier.",
		},
		{
			name: "created already issued by someone else",
			event: orderEvent(actActorUserID, constants.AuditActionCreate,
				change("number", `null`, `"SO-1042"`),
				change("sales_order_status_code", `null`, `"issued"`),
				change("sales_rep_id", `null`, `"acus_owner"`)),
			world: func(w *orderWorld) { w.creatorID = actActorUserID; w.createRequestID = "req_abc123" },
			body:  "Blake Doe created and issued the order.",
		},
		{
			name: "line added to an issued order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_2", constants.AuditActionCreate,
				change("line_item_number", `null`, `2`),
				change("product_sku", `null`, `"WIDGET-RED"`),
				change("quantity_value", `null`, `"5.000"`),
				change("quantity_unit_name", `null`, `"pair"`)),
			body: "Blake Doe added line 2 (WIDGET-RED) to the order — 5 pairs.",
		},
		{
			name: "line removed from an issued order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_2", constants.AuditActionDelete,
				change("line_item_number", `2`, `null`),
				change("product_sku", `"WIDGET-RED"`, `null`)),
			body: "Blake Doe removed line 2 (WIDGET-RED) from the order.",
		},
		{
			name:  "line quantity on an issued order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`)),
			body:  "Blake Doe updated a line to 2.",
		},
		{
			name: "line product on an issued order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate,
				change("product_id", `"prod_1"`, `"prod_2"`),
				change("product_sku", `"WIDGET-BLUE"`, `"WIDGET-RED"`)),
			body: "Blake Doe updated the WIDGET-RED line: product to WIDGET-RED.",
		},
		{
			name:  "line change on a fulfilled order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`)),
			world: func(w *orderWorld) { w.status = string(constants.SalesOrderStatusCodeFulfilled) },
			body:  "Blake Doe updated a line to 2.",
		},
		{
			name: "a system actor fulfils the order",
			ctx:  systemActorCtx("acct_1", constants.SystemActorStripe),
			event: asSystem(orderEvent("", constants.AuditActionUpdate,
				change("sales_order_status_code", `"issued"`, `"fulfilled"`)), constants.SystemActorStripe),
			body: "Stripe marked the order fulfilled.",
		},
		{
			name:  "no sales rep: the creator owns the order",
			event: orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...),
			world: func(w *orderWorld) { w.repAccountUserID = ""; w.repUserID = "" },
			body:  "Blake Doe issued the order.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := defaultOrderWorld()
			if tc.world != nil {
				tc.world(world)
			}
			ctx := tc.ctx
			if ctx == nil {
				ctx = namedActorCtx("acct_1", actActorUserID, "Blake Doe")
			}

			fanouts := newActivityHarness(t, world).save(t, ctx, tc.event)

			require.Len(t, fanouts, 1)
			data := fanouts[0]
			wantOwner := world.repUserID
			if world.repAccountUserID == "" {
				wantOwner = world.creatorID
			}
			assert.Equal(t, []string{wantOwner}, data.RecipientUserIDs)
			assert.Empty(t, data.RecipientAccountUserIDs)
			assert.Equal(t, tc.body, data.Body)
			if tc.title != "" {
				assert.Equal(t, tc.title, data.Title)
			}
		})
	}
}

func TestOrderActivity_NonMilestonesStayQuiet(t *testing.T) {
	estimate := func(w *orderWorld) { w.status = string(constants.SalesOrderStatusCodeEstimate) }

	cases := []struct {
		name  string
		ctx   context.Context
		event *domain.AuditEvent
		world func(*orderWorld)
	}{
		{name: "note", event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("note", `null`, `"call first"`))},
		{name: "customer PO", event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("customer_po_number", `null`, `"PO-7"`))},
		{name: "carrier", event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("carrier_id", `"car_1"`, `"car_2"`))},
		{name: "acknowledgement sent", event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("is_acknowledgment_sent", `false`, `true`))},
		{name: "sales rep reassigned", event: orderEvent(actActorUserID, constants.AuditActionUpdate, change("sales_rep_id", `"acus_old"`, `"acus_owner"`))},
		{
			name:  "created as an estimate",
			event: orderEvent(actActorUserID, constants.AuditActionCreate, change("sales_order_status_code", `null`, `"estimate"`)),
			world: estimate,
		},
		{
			name:  "line added to an estimate",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_2", constants.AuditActionCreate, change("quantity_value", `null`, `"5"`)),
			world: estimate,
		},
		{
			name:  "line quantity on an estimate",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`)),
			world: estimate,
		},
		{name: "line price", event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("unit_price_value", `"1"`, `"2"`))},
		{name: "line cost", event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("unit_cost_value", `"1"`, `"2"`))},
		{name: "line picked quantity", event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_picked_value", `"0"`, `"2"`))},
		{
			name:  "lines written by the request that created the order",
			event: childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionCreate, change("quantity_value", `null`, `"5"`)),
			world: func(w *orderWorld) { w.createRequestID = "req_abc123" },
		},
		{name: "pick", event: childEvent(actActorUserID, constants.ObjectTypePick, "pk_1", constants.AuditActionCreate)},
		{name: "pick line", event: childEvent(actActorUserID, constants.ObjectTypePickLine, "pkl_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`))},
		{name: "shipping case", event: childEvent(actActorUserID, constants.ObjectTypeShippingCase, "sc_1", constants.AuditActionCreate)},
		{name: "shipment", event: childEvent(actActorUserID, constants.ObjectTypeShipment, "shp_1", constants.AuditActionUpdate, change("tracking_number", `null`, `"1Z"`))},
		{name: "invoice", event: childEvent(actActorUserID, constants.ObjectTypeInvoice, "inv_1", constants.AuditActionCreate)},
		{
			name:  "system payment",
			ctx:   systemActorCtx("acct_1", constants.SystemActorStripe),
			event: asSystem(childEvent("", constants.ObjectTypeInvoice, "inv_1", constants.AuditActionUpdate, change("amount_paid_value", `"0"`, `"10"`)), constants.SystemActorStripe),
		},
		{name: "shipment line", event: childEvent(actActorUserID, constants.ObjectTypeShipmentLine, "shpl_1", constants.AuditActionCreate)},
		{name: "freight cost write-back", event: childEvent(actActorUserID, constants.ObjectTypeFreight, "frt_1", constants.AuditActionUpdate, change("cost_value", `"0"`, `"12"`))},
		{
			name:  "system ship-by recompute",
			ctx:   systemActorCtx("acct_1", constants.SystemActorScheduler),
			event: asSystem(orderEvent("", constants.AuditActionUpdate, change("ship_by_date", `"2026-08-01T00:00:00Z"`, `"2026-08-03T00:00:00Z"`)), constants.SystemActorScheduler),
		},
		{
			name:  "system promised-date change",
			ctx:   systemActorCtx("acct_1", constants.SystemActorScheduler),
			event: asSystem(orderEvent("", constants.AuditActionUpdate, change("promised_at", `"2026-08-01T00:00:00Z"`, `"2026-08-03T00:00:00Z"`)), constants.SystemActorScheduler),
		},
		{
			name:  "system line change",
			ctx:   systemActorCtx("acct_1", constants.SystemActorStripe),
			event: asSystem(childEvent("", constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`)), constants.SystemActorStripe),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := defaultOrderWorld()
			if tc.world != nil {
				tc.world(world)
			}
			ctx := tc.ctx
			if ctx == nil {
				ctx = namedActorCtx("acct_1", actActorUserID, "Blake Doe")
			}
			assert.Empty(t, newActivityHarness(t, world).save(t, ctx, tc.event))
		})
	}
}

func TestOrderActivity_NoOwnerToTell(t *testing.T) {
	cases := []struct {
		name  string
		actor string
		world func(*orderWorld)
	}{
		{name: "the rep issues their own order", actor: actOwnerUserID},
		{
			name:  "the creator issues their own rep-less order",
			actor: actCreatorUserID,
			world: func(w *orderWorld) { w.repAccountUserID = ""; w.repUserID = "" },
		},
		{
			name:  "the rep's membership is gone",
			actor: actActorUserID,
			world: func(w *orderWorld) { w.repUserID = "" },
		},
		{
			name:  "an API key created the rep-less order",
			actor: actActorUserID,
			world: func(w *orderWorld) { w.repAccountUserID = ""; w.creatorID = "key_1"; w.creatorType = "api_key" },
		},
		{
			name:  "an agent created the rep-less order",
			actor: actActorUserID,
			world: func(w *orderWorld) { w.repAccountUserID = ""; w.creatorID = "agt_1"; w.creatorType = "agent" },
		},
		{
			name:  "a system actor created the rep-less order",
			actor: actActorUserID,
			world: func(w *orderWorld) {
				w.repAccountUserID = ""
				w.creatorID = constants.SystemActorStripe.ActorID()
				w.creatorType = string(types.IdentityActorTypeSystem)
			},
		},
		{
			name:  "no create event and no rep on record",
			actor: actActorUserID,
			world: func(w *orderWorld) { w.repAccountUserID = ""; w.creatorID = "" },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := defaultOrderWorld()
			if tc.world != nil {
				tc.world(world)
			}
			ctx := namedActorCtx("acct_1", tc.actor, "Avery Doe")
			assert.Empty(t, newActivityHarness(t, world).save(t, ctx, orderEvent(tc.actor, constants.AuditActionUpdate, issueChanges()...)))
		})
	}
}

func TestOrderActivity_NonMilestonesSkipTheLookups(t *testing.T) {
	// A nil world fails on any repo read beyond the event's own insert.
	h := newActivityHarness(t, nil)
	ctx := namedActorCtx("acct_1", actActorUserID, "Blake Doe")
	assert.Empty(t, h.save(t, ctx, orderEvent(actActorUserID, constants.AuditActionUpdate, change("note", `null`, `"x"`))))
	assert.Empty(t, h.save(t, ctx, childEvent(actActorUserID, constants.ObjectTypePickLine, "pkl_1", constants.AuditActionCreate)))
	assert.Empty(t, h.save(t, ctx, &domain.AuditEvent{
		ID: "aevt_unit", ActorID: actActorUserID, IdentityType: "user", AccountID: "acct_1",
		Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeUnit, ResourceID: "unit_1",
		Changes: []domain.AuditFieldChange{change("name", `"g"`, `"kg"`)},
	}))
}

func TestOrderActivity_FanoutCoalescesQuietlyPerOrderPerDay(t *testing.T) {
	h := newActivityHarness(t, defaultOrderWorld())
	ctx := namedActorCtx("acct_1", actActorUserID, "Blake Doe")

	fanouts := h.save(t, ctx, orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...))

	require.Len(t, fanouts, 1)
	data := fanouts[0]
	assert.Equal(t, "msg_ordact_req_abc123_so_1", h.outbox.inputs[0].MessageID)
	assert.Equal(t, "acct_1", data.AccountID)
	assert.Equal(t, string(constants.NotificationCategoryOrderUpdated), data.Category)
	assert.Equal(t, "alert", data.Kind)
	assert.Equal(t, string(constants.ObjectTypeSalesOrder), data.LinkResourceType)
	assert.Equal(t, "so_1", data.LinkResourceID)
	assert.Equal(t, string(constants.NotificationSenderTypeSystem), data.SenderType)
	// One bell row per order per UTC day; later milestones that day refresh it without re-marking it unread.
	assert.Equal(t, "ordact_so_1_"+time.Now().UTC().Format("20060102"), data.DedupeKey)
	assert.True(t, data.CoalesceQuietly)
}

func TestOrderActivity_NoRequestIDFallsBackToEventID(t *testing.T) {
	h := newActivityHarness(t, defaultOrderWorld())
	event := orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...)
	event.RequestID = nil

	require.Len(t, h.save(t, namedActorCtx("acct_1", actActorUserID, "Blake Doe"), event), 1)
	assert.Equal(t, "msg_ordact_aevt_so1", h.outbox.inputs[0].MessageID)
}

func TestOrderActivity_DuplicateFanoutIsIdempotent(t *testing.T) {
	h := newActivityHarness(t, defaultOrderWorld())
	// Another audit event from the same request already claimed this (order, request) fan-out.
	h.outbox.err = &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}

	assert.Nil(t, h.svc.SaveAuditEvent(namedActorCtx("acct_1", actActorUserID, "Blake Doe"),
		orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...)))
}

func TestOrderActivity_LookupErrorPropagates(t *testing.T) {
	h := newActivityHarness(t, nil)
	expected := apierror.NewInternalError(nil, "db failure")
	h.repo.EXPECT().
		GetResourceCreateEvent(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, expected)

	apiErr := h.svc.SaveAuditEvent(namedActorCtx("acct_1", actActorUserID, "Blake Doe"),
		orderEvent(actActorUserID, constants.AuditActionUpdate, issueChanges()...))
	require.NotNil(t, apiErr)
	assert.Equal(t, expected.Code, apiErr.Code)
	assert.Empty(t, h.outbox.inputs)
}

func TestOrderActivity_LineBodyUsesTheLineSnapshot(t *testing.T) {
	world := defaultOrderWorld()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockAuditEventRepo(ctrl)
	outbox := &capturingOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAuditEventRepo().Return(repo)
	factory.EXPECT().NewOutboxRepo().Return(outbox)
	h := &activityHarness{repo: repo, outbox: outbox, svc: NewAuditEventSvc(&AuditEventSvcConfig{Repos: factory})}

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().GetResourceCreateEvent(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(&domain.ResourceCreateEvent{ActorID: actCreatorUserID, IdentityType: "user"}, nil)
	repo.EXPECT().GetLatestResourceFieldValue(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1", "sales_order_status_code").
		Return(jsonString(world.status), true, nil)
	repo.EXPECT().GetLatestResourceFieldValue(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1", "sales_rep_id").
		Return(jsonString(actOwnerAccountUserID), true, nil)
	repo.EXPECT().GetAccountUserUserID(gomock.Any(), "acct_1", actOwnerAccountUserID).Return(actOwnerUserID, nil)
	// The line's own create snapshot supplies its display identity (number, SKU, unit).
	repo.EXPECT().GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrderLine, "sol_1").
		Return([]domain.AuditFieldChange{
			change("line_item_number", `null`, `1`),
			change("product_sku", `null`, `"WIDGET-BLUE"`),
			change("quantity_value", `null`, `"1"`),
			change("quantity_unit_name", `null`, `"pair"`),
		}, nil)

	fanouts := h.save(t, namedActorCtx("acct_1", actActorUserID, "Casey Doe"),
		childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionUpdate, change("quantity_value", `"1"`, `"2"`)))

	require.Len(t, fanouts, 1)
	assert.Equal(t, []string{actOwnerUserID}, fanouts[0].RecipientUserIDs)
	assert.Equal(t, "Sales order updated", fanouts[0].Title)
	assert.Equal(t, "Casey Doe updated line 1 (WIDGET-BLUE) to 2 pairs.", fanouts[0].Body)
}
