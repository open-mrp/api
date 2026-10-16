package service

import (
	"context"

	authtypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

// Automatic subscriptions: the audit trail says who created, edited or was assigned to a followable record, and
// notification-service subscribes each of them whose auto-subscribe setting includes the reason.

// followableAssigneeFields are the fields that name a followable record's assignee (an account user).
var followableAssigneeFields = map[constants.ObjectType]string{
	constants.ObjectTypeSalesOrder:    "sales_rep_id",
	constants.ObjectTypeProductionRun: "responsible_user_id",
	constants.ObjectTypeCustomer:      "default_sales_rep_id",
}

// followableAuditTypes maps the audit resource types of followable records to their record type.
var followableAuditTypes = map[constants.ObjectType]constants.RecordType{
	constants.ObjectTypeSalesOrder:    constants.RecordTypeSalesOrder,
	constants.ObjectTypePurchaseOrder: constants.RecordTypePurchaseOrder,
	constants.ObjectTypeProductionRun: constants.RecordTypeProductionRun,
	constants.ObjectTypeInvoice:       constants.RecordTypeInvoice,
	constants.ObjectTypeCustomer:      constants.RecordTypeCustomer,
	constants.ObjectTypeSupplier:      constants.RecordTypeSupplier,
}

// subscriptionIntentsFor is who an audit event of a followable record subscribes: a person who creates it (creator)
// or edits it (editor), and the account user a change assigns to it (assignee). System actors, API keys and agents
// are never subscribed.
func subscriptionIntentsFor(event *domain.AuditEvent) []messaging.SubscriptionIntent {
	if _, ok := followableAuditTypes[event.ResourceType]; !ok {
		return nil
	}
	var intents []messaging.SubscriptionIntent
	if event.IdentityType == string(authtypes.IdentityActorTypeUser) && !isSystemActorEvent(event) && event.ActorID != "" {
		switch event.Action {
		case constants.AuditActionCreate:
			intents = append(intents, messaging.SubscriptionIntent{UserID: event.ActorID, Reason: string(constants.SubscriptionReasonCreator)})
		case constants.AuditActionUpdate:
			intents = append(intents, messaging.SubscriptionIntent{UserID: event.ActorID, Reason: string(constants.SubscriptionReasonEditor)})
		}
	}
	if field, ok := followableAssigneeFields[event.ResourceType]; ok && event.Action != constants.AuditActionDelete {
		if assignee := decodeString(changeValues(event.Changes, false)[field]); assignee != "" {
			intents = append(intents, messaging.SubscriptionIntent{AccountUserID: assignee, Reason: string(constants.SubscriptionReasonAssignee)})
		}
	}
	return intents
}

// subscribeFromAuditEvent asks notification-service to subscribe the people an audit event of a followable record
// names. Events of other records, or that name nobody, do nothing.
func (s *auditEventSvcImpl) subscribeFromAuditEvent(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	recordType, ok := followableAuditTypes[event.ResourceType]
	if !ok || event.AccountID == "" || event.ResourceID == "" {
		return nil
	}
	intents := subscriptionIntentsFor(event)
	if len(intents) == 0 {
		return nil
	}
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.subscribe_from_audit_event")
	defer span.End()
	return tracing.Trace(span, s.enqueueFanout(ctx, "msg_sub_"+event.ID, messaging.AlertFanoutData{
		AccountID: event.AccountID,
		Kind:      "subscribe",
		Record:    &messaging.RecordRef{Type: string(recordType), ID: event.ResourceID},
		Subscribe: intents,
	}))
}
