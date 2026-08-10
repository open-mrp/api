package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/mock/gomock"
)

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

func salesOrderUpdateEvent(actorID string) *domain.AuditEvent {
	requestID := "req_abc123"
	return &domain.AuditEvent{
		ID:           "aevt_so1",
		ActorID:      actorID,
		ActorType:    "internal",
		IdentityType: "user",
		AccountID:    "acct_1",
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeSalesOrder,
		ResourceID:   "so_1",
		Changes: []domain.AuditFieldChange{
			{Field: "sales_order_status_code", OldValue: json.RawMessage(`"estimate"`), NewValue: json.RawMessage(`"issued"`)},
			{Field: "issued_at", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"2026-08-10T12:00:00Z"`)},
		},
		RequestID:  &requestID,
		OccurredAt: time.Now().UTC(),
	}
}

func (s *AuditEventServiceTestSuite) decodeFanout(input messaging.OutboxMessageInput) messaging.AlertFanoutData {
	var data messaging.AlertFanoutData
	s.Require().NoError(json.Unmarshal(input.Payload.Data, &data))
	return data
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_OrderUpdateNotifiesFollowers() {
	event := salesOrderUpdateEvent("usr_b")
	ctx := namedActorCtx("acct_1", "usr_b", "Blake Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_b"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]domain.AuditFieldChange{
			{Field: "number", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"SO-1042"`)},
		}, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)

	s.Require().Len(s.outbox.inputs, 1)
	input := s.outbox.inputs[0]
	s.Equal(string(contracts.NotificationCmdFanout), input.RoutingKey)
	s.Equal("msg_ordact_req_abc123_so_1", input.MessageID)

	data := s.decodeFanout(input)
	s.Equal("acct_1", data.AccountID)
	s.Equal(string(constants.NotificationCategoryOrderUpdated), data.Category)
	s.Equal([]string{"usr_a"}, data.RecipientUserIDs)
	s.Equal("Sales order SO-1042 updated", data.Title)
	s.Equal("Blake Doe issued the order.", data.Body)
	s.Equal(string(constants.ObjectTypeSalesOrder), data.LinkResourceType)
	s.Equal("so_1", data.LinkResourceID)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_OrderLineEventNotifiesViaRoot() {
	requestID := "req_line1"
	event := &domain.AuditEvent{
		ID:               "aevt_line1",
		ActorID:          "usr_c",
		IdentityType:     "user",
		AccountID:        "acct_1",
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypeSalesOrderLine,
		ResourceID:       "sol_1",
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   "so_1",
		Changes: []domain.AuditFieldChange{
			{Field: "quantity_value", OldValue: json.RawMessage(`"1"`), NewValue: json.RawMessage(`"2"`)},
		},
		RequestID:  &requestID,
		OccurredAt: time.Now().UTC(),
	}
	ctx := namedActorCtx("acct_1", "usr_c", "Casey Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_b", "usr_c"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, nil).Times(1)
	// The line's own create snapshot supplies its display identity (number, SKU, unit).
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrderLine, "sol_1").
		Return([]domain.AuditFieldChange{
			{Field: "line_item_number", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`1`)},
			{Field: "product_sku", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"WIDGET-BLUE"`)},
			{Field: "quantity_value", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"1"`)},
			{Field: "quantity_unit_name", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"pair"`)},
		}, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)

	s.Require().Len(s.outbox.inputs, 1)
	data := s.decodeFanout(s.outbox.inputs[0])
	s.ElementsMatch([]string{"usr_a", "usr_b"}, data.RecipientUserIDs)
	s.Equal("Sales order updated", data.Title)
	s.Equal("Casey Doe updated line 1 (WIDGET-BLUE) to 2 pairs.", data.Body)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_LineAddedBody() {
	requestID := "req_line2"
	event := &domain.AuditEvent{
		ID:               "aevt_line2",
		ActorID:          "usr_c",
		IdentityType:     "user",
		AccountID:        "acct_1",
		Action:           constants.AuditActionCreate,
		ResourceType:     constants.ObjectTypeSalesOrderLine,
		ResourceID:       "sol_2",
		RootResourceType: constants.ObjectTypeSalesOrder,
		RootResourceID:   "so_1",
		Changes: []domain.AuditFieldChange{
			{Field: "line_item_number", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`2`)},
			{Field: "product_sku", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"WIDGET-RED"`)},
			{Field: "quantity_value", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"5.000"`)},
			{Field: "quantity_unit_name", OldValue: json.RawMessage(`null`), NewValue: json.RawMessage(`"pair"`)},
		},
		RequestID:  &requestID,
		OccurredAt: time.Now().UTC(),
	}
	ctx := namedActorCtx("acct_1", "usr_c", "Casey Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_c"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)

	s.Require().Len(s.outbox.inputs, 1)
	data := s.decodeFanout(s.outbox.inputs[0])
	s.Equal("Casey Doe added line 2 (WIDGET-RED) to the order — 5 pairs.", data.Body)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_OrderFieldChangesBody() {
	event := salesOrderUpdateEvent("usr_b")
	event.Changes = []domain.AuditFieldChange{
		{Field: "promised_at", OldValue: json.RawMessage(`"2026-08-01T00:00:00Z"`), NewValue: json.RawMessage(`"2026-09-15T00:00:00Z"`)},
		{Field: "carrier_id", OldValue: json.RawMessage(`"car_1"`), NewValue: json.RawMessage(`"car_2"`)},
	}
	ctx := namedActorCtx("acct_1", "usr_b", "Blake Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_b"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)

	s.Require().Len(s.outbox.inputs, 1)
	data := s.decodeFanout(s.outbox.inputs[0])
	s.Equal("Blake Doe changed the promised date to Sep 15, 2026 and changed the carrier.", data.Body)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_NoFollowersNoFanout() {
	event := salesOrderUpdateEvent("usr_a")
	ctx := namedActorCtx("acct_1", "usr_a", "Avery Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	// The only prior actor is the current actor — nobody else to notify.
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a"}, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)
	s.Empty(s.outbox.inputs)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_NonOrderEventNoFanout() {
	event := &domain.AuditEvent{
		ID:           "aevt_unit",
		ActorID:      "usr_a",
		IdentityType: "user",
		AccountID:    "acct_1",
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeUnit,
		ResourceID:   "unit_1",
		Changes: []domain.AuditFieldChange{
			{Field: "name", OldValue: json.RawMessage(`"g"`), NewValue: json.RawMessage(`"kg"`)},
		},
		OccurredAt: time.Now().UTC(),
	}

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(context.Background(), event)
	s.Nil(apiErr)
	s.Empty(s.outbox.inputs)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_DuplicateFanoutIsIdempotent() {
	event := salesOrderUpdateEvent("usr_b")
	ctx := namedActorCtx("acct_1", "usr_b", "Blake Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_b"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, nil).Times(1)

	// Another audit event from the same request already claimed this (order, request) fan-out.
	s.outbox.err = &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_FollowerQueryErrorPropagates() {
	event := salesOrderUpdateEvent("usr_b")
	ctx := namedActorCtx("acct_1", "usr_b", "Blake Doe")
	expected := apierror.NewInternalError(nil, "db failure")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, expected).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Require().NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
	s.Empty(s.outbox.inputs)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_NoRequestIDFallsBackToEventID() {
	event := salesOrderUpdateEvent("usr_b")
	event.RequestID = nil
	ctx := namedActorCtx("acct_1", "usr_b", "Blake Doe")

	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)
	s.auditEventRepo.EXPECT().
		ListResourceUserActorIDs(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return([]string{"usr_a", "usr_b"}, nil).Times(1)
	s.auditEventRepo.EXPECT().
		GetResourceCreateChanges(gomock.Any(), "acct_1", constants.ObjectTypeSalesOrder, "so_1").
		Return(nil, nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(ctx, event)
	s.Nil(apiErr)
	s.Require().Len(s.outbox.inputs, 1)
	s.Equal("msg_ordact_aevt_so1", s.outbox.inputs[0].MessageID)
}
