package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
)

type Publisher struct{}

func NewPublisher() *Publisher { return &Publisher{} }

type auditEventOutboxPayload struct {
	TypeID           string                `json:"type_id"`
	Action           constants.AuditAction `json:"action"`
	ResourceType     constants.ObjectType  `json:"resource_type"`
	ResourceID       string                `json:"resource_id"`
	RootResourceType constants.ObjectType  `json:"root_resource_type,omitempty"`
	RootResourceID   string                `json:"root_resource_id,omitempty"`
	Changes          []FieldChange         `json:"changes"`
	Metadata         map[string]any        `json:"metadata"`
	ServiceName      string                `json:"service_name"`
	IdempotencyKeyID *string               `json:"idempotency_key_id,omitempty"`
	SourceIP         *string               `json:"source_ip,omitempty"`
	OccurredAt       time.Time             `json:"occurred_at"`
}

// Publish enqueues an audit event via the platform outbox pipeline.
//
// It is safe to call inside a service transaction: the outboxRepo write is performed atomically with the business mutation.
func (p *Publisher) Publish(
	ctx context.Context,
	outboxRepo messaging.OutboxRepo,
	data EventData,
) *apierror.APIError {
	// Skip no-op updates: an update event with no recorded changes and no metadata carries no information (e.g. a PATCH that sets fields to their current values). Create and delete events always publish.
	if data.Action == constants.AuditActionUpdate && len(data.Changes) == 0 && len(data.Metadata) == 0 {
		return nil
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if !identity.IsTargetAccountSet() {
		return apierror.NewAuthenticationError("The Augno-Account header is required.")
	}
	if outboxRepo == nil {
		return apierror.NewInternalError(nil, "Audit publisher: outbox repo is required.")
	}

	requestID, _ := appctx.GetRequestID(ctx)
	var idempotencyKeyIDPtr *string
	if ikID, ok := appctx.GetIdempotencyKeyID(ctx); ok {
		idempotencyKeyIDPtr = &ikID
	}

	var sourceIP *string
	if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil && rl.ClientIPString != nil && *rl.ClientIPString != "" {
		sourceIP = rl.ClientIPString
	} else if ip, ok := appctx.GetPropagatedClientIP(ctx); ok {
		sourceIP = &ip
	}

	length := id.IDLength19
	typeID, apiErr := id.GenID(id.AuditEventIDPrefix, &length)
	if apiErr != nil {
		return apierror.NewInternalError(apiErr, "Audit publisher: failed to generate audit event ID.")
	}

	occurredAt := time.Now().UTC()

	payload := auditEventOutboxPayload{
		TypeID:           typeID,
		Action:           data.Action,
		ResourceType:     data.ResourceType,
		ResourceID:       data.ResourceID,
		RootResourceType: data.RootResourceType,
		RootResourceID:   data.RootResourceID,
		Changes:          data.Changes,
		Metadata:         data.Metadata,
		ServiceName:      data.ServiceName,
		IdempotencyKeyID: idempotencyKeyIDPtr,
		SourceIP:         sourceIP,
		OccurredAt:       occurredAt,
	}

	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return apierror.NewInternalError(err, "Audit publisher: failed to marshal audit payload.")
	}

	// Let the outbox repo generate its own MessageID if needed, but we supply a deterministic one to keep causal chains easy to trace.
	length2 := id.IDLength22
	msgID, apiErr := id.GenID(id.MessageIDPrefix, &length2)
	if apiErr != nil {
		return apierror.NewInternalError(apiErr, "Audit publisher: failed to generate message ID.")
	}

	amqpMsg := contracts.AmqpMessage{
		Identity:  identity,
		RequestID: requestID,
		Data:      dataBytes,
		MessageID: msgID,
	}

	_, apiErr2 := outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: data.ServiceName,
		MessageType: string(contracts.PlatformEventAuditLogged),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.PlatformEventAuditLogged),
		Payload:     amqpMsg,
	})
	if apiErr2 != nil {
		return apierror.NewInternalError(apiErr2, "Audit publisher: failed to enqueue outbox message.")
	}

	return nil
}
