package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

// Sales-order activity notifications: every user who creates or edits a sales order — or any record rooted at it (lines, shipments, invoices, picks) — implicitly follows the order, and is notified of each subsequent change made by anyone else. The audit trail doubles as the follower registry: a user's own audit events mark them a follower for everything that comes after, so nobody is notified of actions that preceded their first touch, and the creator hears about every edit from the moment the order exists.

// notifySalesOrderFollowers fans a bell notification out to the order's followers (prior user actors on the order's audit trail, minus the current actor) after a sales-order-related audit event is persisted. Non-order events no-op.
//
// The audit consumer runs concurrently, so two near-simultaneous first edits by different users can each miss the other's just-persisted follower registration — accepted: neither was a follower when the other's change was made, and both follow from then on.
func (s *auditEventSvcImpl) notifySalesOrderFollowers(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	orderID := salesOrderIDForEvent(event)
	if orderID == "" || event.AccountID == "" {
		return nil
	}

	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.notify_sales_order_followers")
	defer span.End()

	followers, apiErr := s.auditEventRepo.ListResourceUserActorIDs(ctx, event.AccountID, constants.ObjectTypeSalesOrder, orderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	recipients := make([]string, 0, len(followers))
	for _, followerID := range followers {
		if followerID != event.ActorID {
			recipients = append(recipients, followerID)
		}
	}
	if len(recipients) == 0 {
		return nil
	}

	data := messaging.AlertFanoutData{
		AccountID:        event.AccountID,
		Category:         string(constants.NotificationCategoryOrderUpdated),
		Kind:             "alert",
		Title:            salesOrderActivityTitle(event, s.salesOrderNumber(ctx, event.AccountID, orderID)),
		Body:             s.salesOrderActivityBody(ctx, event),
		LinkResourceType: string(constants.ObjectTypeSalesOrder),
		LinkResourceID:   orderID,
		Priority:         string(constants.NotificationPriorityNormal),
		SenderType:       string(constants.NotificationSenderTypeSystem),
		RecipientUserIDs: recipients,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal order activity fan-out payload."))
	}

	messageID := salesOrderActivityMessageID(event, orderID)
	msg := contracts.AmqpMessage{Data: dataJSON, MessageID: messageID}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   messageID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     msg,
	}); err != nil {
		// A duplicate message id means this request's edit was already fanned out (another audit event from the same request got there first, or a retry re-ran a completed step).
		if db.IsDuplicateEntry(err) {
			return nil
		}
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to enqueue order activity fan-out."))
	}

	return nil
}

// salesOrderIDForEvent resolves the sales order an audit event belongs to — the resource itself, or the root of the record tree for child resources. Empty when the event is unrelated to a sales order.
func salesOrderIDForEvent(event *domain.AuditEvent) string {
	if event.ResourceType == constants.ObjectTypeSalesOrder {
		return event.ResourceID
	}
	if event.RootResourceType == constants.ObjectTypeSalesOrder {
		return event.RootResourceID
	}
	return ""
}

// salesOrderActivityMessageID keys the fan-out to one notification per (order, originating request): a single API call may emit several audit events against the same order (the header plus each line row), and the outbox's unique message_id collapses the extras, so followers get one notification per edit rather than one per row. Events with no request id fall back to the audit event id (one per event). Determinism also makes retries and redeliveries idempotent end-to-end — the notification-service derives per-recipient notification ids from this id.
func salesOrderActivityMessageID(event *domain.AuditEvent, orderID string) string {
	if event.RequestID != nil && *event.RequestID != "" {
		return "msg_ordact_" + *event.RequestID + "_" + orderID
	}
	return "msg_ordact_" + event.ID
}

// salesOrderNumber recovers the order's human-facing number from its create audit event's field snapshot. Best-effort: platform-service has no core-service client, and an order that predates auditing simply yields "".
func (s *auditEventSvcImpl) salesOrderNumber(ctx context.Context, accountID, orderID string) string {
	return decodeString(s.resourceCreateValues(ctx, accountID, constants.ObjectTypeSalesOrder, orderID)["number"])
}

// resourceCreateValues returns a field → value map from the resource's create audit event snapshot (its audited fields at creation). Best-effort: an empty map when no create event is on record.
func (s *auditEventSvcImpl) resourceCreateValues(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID string) map[string]json.RawMessage {
	changes, apiErr := s.auditEventRepo.GetResourceCreateChanges(ctx, accountID, resourceType, resourceID)
	if apiErr != nil {
		return map[string]json.RawMessage{}
	}
	return changeValues(changes, false)
}

func salesOrderActivityTitle(event *domain.AuditEvent, orderNumber string) string {
	verb := "updated"
	if event.ResourceType == constants.ObjectTypeSalesOrder && event.Action == constants.AuditActionDelete {
		verb = "deleted"
	}
	if orderNumber == "" {
		return "Sales order " + verb
	}
	return fmt.Sprintf("Sales order %s %s", orderNumber, verb)
}

// salesOrderActivityBody renders "<actor> <did what>." in plain English — e.g. "Blake Doe issued the order." or "Casey Doe updated line 1 (WIDGET-BLUE) to 2 pairs." — so the recipient understands the change without opening the order.
func (s *auditEventSvcImpl) salesOrderActivityBody(ctx context.Context, event *domain.AuditEvent) string {
	actorName := "Someone"
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil && identity.Actor != nil && identity.Actor.Name != nil && *identity.Actor.Name != "" {
		actorName = *identity.Actor.Name
	}

	var action string
	switch event.ResourceType {
	case constants.ObjectTypeSalesOrder:
		action = describeOrderEvent(event)
	case constants.ObjectTypeSalesOrderLine:
		action = s.describeOrderLineEvent(ctx, event)
	default:
		action = s.describeChildResourceEvent(ctx, event)
	}

	return actorName + " " + action + "."
}

// describeOrderEvent phrases a change to the order record itself. Update diffs are translated field by field into verb phrases ("issued the order", "changed the promised date to Jan 5, 2026"); derivative timestamp companions of a status change are skipped so "issued the order" isn't drowned in bookkeeping.
func describeOrderEvent(event *domain.AuditEvent) string {
	switch event.Action {
	case constants.AuditActionCreate:
		return "created the order"
	case constants.AuditActionDelete:
		return "deleted the order"
	}

	phrases := make([]string, 0, len(event.Changes))
	for _, change := range event.Changes {
		if phrase := orderChangePhrase(change); phrase != "" {
			phrases = append(phrases, phrase)
		}
	}
	if len(phrases) == 0 {
		return "updated the order"
	}
	return joinPhrases(phrases)
}

// orderChangePhrase turns one order field transition into a verb phrase, or "" for fields that shouldn't be narrated.
func orderChangePhrase(change domain.AuditFieldChange) string {
	newValue := decodeString(change.NewValue)

	switch change.Field {
	case "sales_order_status_code":
		switch constants.SalesOrderStatusCode(newValue) {
		case constants.SalesOrderStatusCodeIssued:
			return "issued the order"
		case constants.SalesOrderStatusCodeFulfilled:
			return "marked the order fulfilled"
		case constants.SalesOrderStatusCodeEstimate:
			return "reverted the order to an estimate"
		default:
			return "changed the status to " + humanizeToken(newValue)
		}
	// Derivative timestamps that accompany a status transition — the status phrase already tells the story.
	case "issued_at", "completed_at", "expired_at", "first_ship_at":
		return ""
	case "is_acknowledgment_sent":
		if string(change.NewValue) == "true" {
			return "sent the order acknowledgement"
		}
		return ""
	case "promised_at":
		return "changed the promised date to " + renderDate(change.NewValue)
	case "ship_by_date":
		return "changed the ship-by date to " + renderDate(change.NewValue)
	case "note":
		return "updated the order note"
	case "customer_po_number":
		return "changed the customer PO number to " + renderValue(change.NewValue)
	}

	// Reference fields carry opaque ids — say what changed, not the id.
	if strings.HasSuffix(change.Field, "_id") {
		return "changed the " + humanizeToken(strings.TrimSuffix(change.Field, "_id"))
	}

	return fmt.Sprintf("changed the %s to %s", humanizeToken(strings.TrimSuffix(change.Field, "_code")), renderValue(change.NewValue))
}

// lineNarrationSkip lists line diff fields that shouldn't be narrated on their own: fulfillment aggregates change as a side effect of picks/packs/invoices (which produce their own events), and unit/product companion fields are already folded into the quantity or product phrase.
var lineNarrationSkip = map[string]bool{
	"quantity_picked_value":            true,
	"quantity_packed_value":            true,
	"quantity_invoiced_value":          true,
	"quantity_unit_id":                 true,
	"quantity_unit_name":               true,
	"quantity_unit_abbreviation":       true,
	"quantity_unit_type":               true,
	"unit_price_numerator_unit_id":     true,
	"unit_price_numerator_unit_abbr":   true,
	"unit_price_denominator_unit_id":   true,
	"unit_price_denominator_unit_abbr": true,
	"unit_cost_numerator_unit_id":      true,
	"unit_cost_numerator_unit_abbr":    true,
	"unit_cost_denominator_unit_id":    true,
	"unit_cost_denominator_unit_abbr":  true,
	"product_id":                       true,
	"product_description":              true,
	"product_type_code":                true,
	"item_id":                          true,
	"item_sku":                         true,
	"edi_line_item_id":                 true,
	"line_item_number":                 true,
}

// describeOrderLineEvent phrases a line change the way a person would say it: "updated line 1 (WIDGET-BLUE) to 2 pairs". Line identity (number, SKU) and the quantity unit come from the line's create snapshot, overlaid with the current diff for fields that just changed.
func (s *auditEventSvcImpl) describeOrderLineEvent(ctx context.Context, event *domain.AuditEvent) string {
	facts := changeValues(event.Changes, event.Action == constants.AuditActionDelete)
	if event.Action != constants.AuditActionCreate && event.Action != constants.AuditActionDelete {
		snapshot := s.resourceCreateValues(ctx, event.AccountID, constants.ObjectTypeSalesOrderLine, event.ResourceID)
		for field, value := range facts {
			snapshot[field] = value
		}
		facts = snapshot
	}

	lineRef := "a line"
	if number := renderValue(facts["line_item_number"]); number != "" {
		lineRef = "line " + number
		if sku := decodeString(facts["product_sku"]); sku != "" {
			lineRef += " (" + sku + ")"
		}
	} else if sku := decodeString(facts["product_sku"]); sku != "" {
		lineRef = "the " + sku + " line"
	}

	quantity := quantityPhrase(facts)

	switch event.Action {
	case constants.AuditActionCreate:
		if quantity != "" {
			return "added " + lineRef + " to the order — " + quantity
		}
		return "added " + lineRef + " to the order"
	case constants.AuditActionDelete:
		return "removed " + lineRef + " from the order"
	}

	phrases := make([]string, 0, len(event.Changes))
	for _, change := range event.Changes {
		switch change.Field {
		case "quantity_value":
			if quantity != "" {
				phrases = append(phrases, "quantity to "+quantity)
			}
		case "unit_price_value":
			phrases = append(phrases, "unit price to "+renderValue(change.NewValue))
		case "unit_cost_value":
			phrases = append(phrases, "unit cost to "+renderValue(change.NewValue))
		case "product_sku":
			phrases = append(phrases, "product to "+decodeString(change.NewValue))
		default:
			if lineNarrationSkip[change.Field] {
				continue
			}
			phrases = append(phrases, fmt.Sprintf("%s to %s", humanizeToken(change.Field), renderValue(change.NewValue)))
		}
	}

	if len(phrases) == 0 {
		return "updated " + lineRef
	}
	// The common single-field edit reads as one sentence: "updated line 1 (WIDGET-BLUE) to 2 pairs".
	if len(phrases) == 1 && strings.HasPrefix(phrases[0], "quantity to ") {
		return "updated " + lineRef + " " + strings.TrimPrefix(phrases[0], "quantity ")
	}
	return "updated " + lineRef + ": " + joinPhrases(phrases)
}

// describeChildResourceEvent phrases a change to another record hanging off the order (shipment, invoice, pick, …), naming it by its number when the create snapshot has one: "added shipment SH-1001 to the order".
func (s *auditEventSvcImpl) describeChildResourceEvent(ctx context.Context, event *domain.AuditEvent) string {
	resource := humanizeToken(string(event.ResourceType))

	number := decodeString(changeValues(event.Changes, event.Action == constants.AuditActionDelete)["number"])
	if number == "" && event.Action != constants.AuditActionCreate && event.Action != constants.AuditActionDelete {
		number = decodeString(s.resourceCreateValues(ctx, event.AccountID, event.ResourceType, event.ResourceID)["number"])
	}

	ref := "a " + resource
	if number != "" {
		ref = resource + " " + number
	}

	switch event.Action {
	case constants.AuditActionCreate:
		return "added " + ref + " to the order"
	case constants.AuditActionDelete:
		return "removed " + ref + " from the order"
	default:
		return "updated " + ref + " on the order"
	}
}

// quantityPhrase renders "2 pairs" from a line's field values — the quantity plus its pluralized unit name.
func quantityPhrase(values map[string]json.RawMessage) string {
	quantity := renderValue(values["quantity_value"])
	if quantity == "" {
		return ""
	}
	unit := decodeString(values["quantity_unit_name"])
	if unit == "" {
		return quantity
	}
	if quantity != "1" && !strings.HasSuffix(unit, "s") {
		unit += "s"
	}
	return quantity + " " + unit
}

// changeValues flattens one side of an audit diff into a field → value map (the old side for deletes, whose new side is all null).
func changeValues(changes []domain.AuditFieldChange, useOld bool) map[string]json.RawMessage {
	values := make(map[string]json.RawMessage, len(changes))
	for _, change := range changes {
		if useOld {
			values[change.Field] = change.OldValue
		} else {
			values[change.Field] = change.NewValue
		}
	}
	return values
}

// joinPhrases lists up to three phrases in prose form, folding the rest into "and N more changes".
func joinPhrases(phrases []string) string {
	const maxPhrases = 3
	if len(phrases) > maxPhrases {
		phrases = append(phrases[:maxPhrases], fmt.Sprintf("and %d more changes", len(phrases)-maxPhrases))
	} else if len(phrases) > 1 {
		phrases = append(phrases[:len(phrases)-1], "and "+phrases[len(phrases)-1])
	}
	if len(phrases) == 2 {
		return strings.Join(phrases, " ")
	}
	return strings.Join(phrases, ", ")
}

// decodeString unwraps a JSON string fragment, or returns "" for anything that isn't one (including null).
func decodeString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// renderValue formats a JSON diff fragment for prose: strings lose their quotes, decimals lose trailing zeros, null reads as "(empty)", and long values are truncated.
func renderValue(raw json.RawMessage) string {
	const maxLen = 40

	if len(raw) == 0 {
		return ""
	}

	value := string(raw)
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		value = s
	}
	if value == "null" {
		value = "(empty)"
	}
	value = trimDecimal(value)
	if runes := []rune(value); len(runes) > maxLen {
		value = string(runes[:maxLen]) + "…"
	}
	return value
}

// renderDate formats a JSON timestamp fragment as "Jan 5, 2026", falling back to the raw value when it doesn't parse.
func renderDate(raw json.RawMessage) string {
	value := decodeString(raw)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format("Jan 2, 2006")
	}
	return renderValue(raw)
}

// trimDecimal drops the trailing zeros of a decimal string ("2.000" → "2") so quantities read naturally.
func trimDecimal(value string) string {
	if !strings.Contains(value, ".") {
		return value
	}
	for _, r := range value {
		if (r < '0' || r > '9') && r != '.' && r != '-' {
			return value
		}
	}
	value = strings.TrimRight(value, "0")
	return strings.TrimSuffix(value, ".")
}

// humanizeToken renders a snake_case field or resource name as prose ("customer_po_number" → "customer PO number").
func humanizeToken(token string) string {
	words := strings.Split(token, "_")
	for i, w := range words {
		switch w {
		case "po", "edi", "sku", "id":
			words[i] = strings.ToUpper(w)
		}
	}
	return strings.Join(words, " ")
}
