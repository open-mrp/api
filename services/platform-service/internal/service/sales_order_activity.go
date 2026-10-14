package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authtypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

// Sales-order activity notifications tell an order's owner — its sales rep, or the user who created it
// when no rep is set — about milestones other people reach on it:
//
//   - the order is issued (including created already issued), fulfilled, shipped, or moved back a status;
//   - the order is deleted;
//   - its promised date or ship-by date changes;
//   - while it is issued, a line is added or removed, or a line's quantity or product changes.
//
// Everything else stays quiet: estimate-stage edits, notes, prices and costs, contacts, sent flags, picks,
// shipping cases, shipment and invoice bookkeeping, freight write-backs and payments. A system actor
// (`system:*`) reports a status, shipment or deletion milestone but not date or line changes, which it only
// makes as a cascade of something else. The owner is never told about their own change, and system actors
// and agents are never owners.
//
// The audit trail is the only record platform-service has of the order, so owner and status are read from
// it: the latest recorded sales_rep_id and sales_order_status_code, and the create event's actor.

// orderActivityKind classifies an audit event against the milestone policy.
type orderActivityKind int

const (
	orderActivityNone orderActivityKind = iota
	// orderActivityMilestone alerts regardless of the order's status.
	orderActivityMilestone
	// orderActivityLineChange alerts only while the order is issued.
	orderActivityLineChange
)

// orderMilestoneFields are order fields whose change is a milestone for a person (not a system actor).
var orderMilestoneFields = map[string]bool{
	"promised_at":  true,
	"ship_by_date": true,
}

// orderLineMilestoneFields are line fields that change what the customer gets.
var orderLineMilestoneFields = map[string]bool{
	"quantity_value":   true,
	"quantity_unit_id": true,
	"product_id":       true,
	"item_id":          true,
}

// notifySalesOrderOwner fans a bell notification out to the order's owner when a sales-order-related
// audit event is a milestone under the policy above. Non-order events and non-milestones no-op.
func (s *auditEventSvcImpl) notifySalesOrderOwner(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	orderID := salesOrderIDForEvent(event)
	if orderID == "" || event.AccountID == "" {
		return nil
	}
	kind := classifyOrderActivity(event)
	if kind == orderActivityNone {
		return nil
	}

	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.notify_sales_order_owner")
	defer span.End()

	created, apiErr := s.auditEventRepo.GetResourceCreateEvent(ctx, event.AccountID, constants.ObjectTypeSalesOrder, orderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if kind == orderActivityLineChange {
		// Lines written by the request that created the order are part of creating it.
		if created != nil && sameRequest(created.RequestID, event.RequestID) {
			return nil
		}
		status, _, apiErr := s.auditEventRepo.GetLatestResourceFieldValue(ctx, event.AccountID, constants.ObjectTypeSalesOrder, orderID, "sales_order_status_code")
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if !isLiveOrderStatus(decodeString(status)) {
			return nil
		}
	}

	ownerID, apiErr := s.salesOrderOwnerUserID(ctx, event.AccountID, orderID, created)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if ownerID == "" || ownerID == event.ActorID {
		return nil
	}

	var orderNumber string
	if created != nil {
		orderNumber = decodeString(changeValues(created.Changes, false)["number"])
	}

	data := messaging.AlertFanoutData{
		AccountID:        event.AccountID,
		Category:         string(constants.NotificationCategoryOrderUpdated),
		Kind:             "alert",
		Title:            salesOrderActivityTitle(event, orderNumber),
		Body:             s.salesOrderActivityBody(ctx, event),
		LinkResourceType: string(constants.ObjectTypeSalesOrder),
		LinkResourceID:   orderID,
		Priority:         string(constants.NotificationPriorityNormal),
		SenderType:       string(constants.NotificationSenderTypeSystem),
		RecipientUserIDs: []string{ownerID},
		DedupeKey:        salesOrderActivityDedupeKey(orderID),
		CoalesceQuietly:  true,
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

// classifyOrderActivity applies the milestone policy to one event without touching the database; line
// changes still depend on the order's current status.
func classifyOrderActivity(event *domain.AuditEvent) orderActivityKind {
	system := isSystemActorEvent(event)

	switch event.ResourceType {
	case constants.ObjectTypeSalesOrder:
		switch event.Action {
		case constants.AuditActionDelete:
			return orderActivityMilestone
		case constants.AuditActionCreate:
			if isLiveOrderStatus(decodeString(changeValues(event.Changes, false)["sales_order_status_code"])) {
				return orderActivityMilestone
			}
			return orderActivityNone
		}
		for _, change := range event.Changes {
			if change.Field == "sales_order_status_code" || change.Field == "first_ship_at" {
				return orderActivityMilestone
			}
			if !system && orderMilestoneFields[change.Field] {
				return orderActivityMilestone
			}
		}
	case constants.ObjectTypeSalesOrderLine:
		if system {
			return orderActivityNone
		}
		if event.Action == constants.AuditActionCreate || event.Action == constants.AuditActionDelete {
			return orderActivityLineChange
		}
		for _, change := range event.Changes {
			if orderLineMilestoneFields[change.Field] {
				return orderActivityLineChange
			}
		}
	}
	return orderActivityNone
}

// isLiveOrderStatus reports whether the order has been issued (fulfilled orders were issued first).
func isLiveOrderStatus(status string) bool {
	switch constants.SalesOrderStatusCode(status) {
	case constants.SalesOrderStatusCodeIssued, constants.SalesOrderStatusCodeFulfilled:
		return true
	}
	return false
}

func isSystemActorEvent(event *domain.AuditEvent) bool {
	if event.IdentityType == string(authtypes.IdentityActorTypeSystem) {
		return true
	}
	_, ok := constants.ParseSystemActorID(event.ActorID)
	return ok
}

func sameRequest(a, b *string) bool {
	return a != nil && b != nil && *a != "" && *a == *b
}

// salesOrderOwnerUserID resolves the user who owns the order: its current sales rep (an account_user),
// or the user who created it when no rep is set. Empty when neither resolves to a user — a rep whose
// membership is gone, or an order created by an API key, agent or system actor.
func (s *auditEventSvcImpl) salesOrderOwnerUserID(ctx context.Context, accountID, orderID string, created *domain.ResourceCreateEvent) (string, *apierror.APIError) {
	rep, _, apiErr := s.auditEventRepo.GetLatestResourceFieldValue(ctx, accountID, constants.ObjectTypeSalesOrder, orderID, "sales_rep_id")
	if apiErr != nil {
		return "", apiErr
	}
	if repID := decodeString(rep); repID != "" {
		return s.auditEventRepo.GetAccountUserUserID(ctx, accountID, repID)
	}
	if created == nil || created.IdentityType != string(authtypes.IdentityActorTypeUser) {
		return "", nil
	}
	return created.ActorID, nil
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

// salesOrderActivityMessageID keys the fan-out to one notification per (order, originating request): a single API call may emit several audit events against the same order (the header plus each line row), and the outbox's unique message_id collapses the extras, so the owner gets one notification per edit rather than one per row. Events with no request id fall back to the audit event id (one per event). Determinism also makes retries and redeliveries idempotent end-to-end — the notification-service derives per-recipient notification ids from this id.
func salesOrderActivityMessageID(event *domain.AuditEvent, orderID string) string {
	if event.RequestID != nil && *event.RequestID != "" {
		return "msg_ordact_" + *event.RequestID + "_" + orderID
	}
	return "msg_ordact_" + event.ID
}

// salesOrderActivityDedupeKey collapses a day's milestones on one order into a single bell row for the
// owner. notification-service coalesces every fan-out sharing this key onto one notification: the first
// milestone of the day inserts it and raises a realtime alert, and later ones refresh its title, body and
// change count without alerting or re-marking it unread (CoalesceQuietly). An owner who dismisses the row
// mutes the order for the rest of the day. The day is bucketed in UTC, which avoids an account timezone
// lookup on this hot path; an account working across UTC midnight gets a second row that day.
func salesOrderActivityDedupeKey(orderID string) string {
	return "ordact_" + orderID + "_" + time.Now().UTC().Format("20060102")
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
	} else if name := systemActorName(event.ActorID); name != "" {
		actorName = name
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

// systemActorName names a background process as the subject of a sentence ("Stripe marked the order fulfilled."), or "" for any other actor.
func systemActorName(actorID string) string {
	actor, ok := constants.ParseSystemActorID(actorID)
	if !ok {
		return ""
	}
	switch actor {
	case constants.SystemActorStripe:
		return "Stripe"
	case constants.SystemActorScheduler:
		return "The scheduler"
	case constants.SystemActorEmailIngest:
		return "Inbound email"
	}
	return ""
}

// describeOrderEvent phrases a change to the order record itself. Update diffs are translated field by field into verb phrases ("issued the order", "changed the promised date to Jan 5, 2026"); derivative timestamp companions of a status change are skipped so "issued the order" isn't drowned in bookkeeping.
func describeOrderEvent(event *domain.AuditEvent) string {
	switch event.Action {
	case constants.AuditActionCreate:
		switch constants.SalesOrderStatusCode(decodeString(changeValues(event.Changes, false)["sales_order_status_code"])) {
		case constants.SalesOrderStatusCodeIssued:
			return "created and issued the order"
		case constants.SalesOrderStatusCodeFulfilled:
			return "created the order as fulfilled"
		}
		return "created the order"
	case constants.AuditActionDelete:
		return "deleted the order"
	}

	statusChanged := false
	for _, change := range event.Changes {
		if change.Field == "sales_order_status_code" {
			statusChanged = true
		}
	}

	phrases := make([]string, 0, len(event.Changes))
	for _, change := range event.Changes {
		// A shipment that also moves the status is told by the status phrase.
		if change.Field == "first_ship_at" && statusChanged {
			continue
		}
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
	case "issued_at", "completed_at", "expired_at":
		return ""
	case "first_ship_at":
		if string(change.NewValue) == "null" || len(change.NewValue) == 0 {
			return "marked the order unshipped"
		}
		return "shipped the order"
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
