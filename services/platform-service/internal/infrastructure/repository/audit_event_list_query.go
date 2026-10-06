package repository

import (
	"database/sql"
	"slices"
	"strings"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

const (
	auditEventAccountIndex       = "audit_event_account_idx"
	auditEventTargetAccountIndex = "audit_event_target_account_idx"
	auditEventResourceIDIndex    = "audit_event_account_id_resource_id_occurred_at_type_id_idx"
	auditEventActorIDIndex       = "audit_event_account_id_actor_id_occurred_at_type_id_idx"
	auditEventResourceTypeIndex  = "audit_event_account_id_resource_type_occurred_at_type_id_idx"
	auditEventActionIndex        = "audit_event_account_id_action_occurred_at_type_id_idx"
	auditEventRequestIDIndex     = "audit_event_account_id_request_id_occurred_at_type_id_idx"
	auditEventRootIndex          = "audit_event_root_idx"
	auditEventActorIndex         = "audit_event_actor_idx"
	auditEventResourceIndex      = "audit_event_resource_idx"

	auditEventTargetResourceIDIndex   = "audit_event_target_resource_id_idx"
	auditEventTargetRequestIDIndex    = "audit_event_target_request_id_idx"
	auditEventTargetResourceTypeIndex = "audit_event_target_resource_type_idx"
	auditEventTargetActionIndex       = "audit_event_target_action_idx"
	auditEventTargetRootIndex         = "audit_event_target_root_idx"
)

// auditEventMaxArms is the most values of one filter read as separate arms, each stopping at a page.
const auditEventMaxArms = 8

// auditEventListColumns matches sqlc.FindAuditEventByIDRow, which scanAuditEventListRows scans into.
var auditEventListColumns = []string{
	"ae.type_id",
	"ae.actor_id AS actor_id",
	"ae.actor_type",
	"ae.identity_type",
	"ae.account_id",
	"ae.action",
	"ae.resource_type",
	"ae.resource_id",
	"COALESCE(CASE WHEN ? THEN ae.changes ELSE NULL END, '') AS changes",
	"COALESCE(CASE WHEN ? THEN ae.metadata ELSE NULL END, '') AS metadata",
	"ae.service_name",
	"ae.request_id",
	"ae.idempotency_key_id",
	"ae.source_ip",
	"ae.occurred_at",
	"ae.created_at",
	"ae.target_account_id",
	"a.name AS account_name",
	"a.created_at AS account_created_at",
	"a.updated_at AS account_updated_at",
	"u.name AS user_name",
	"u.email AS user_email",
	"ak.name AS api_key_name",
	"ak.redacted_value AS api_key_redacted_value",
	"ik.idempotency_key",
}

// auditEventPin is a column a search pins to one value, with the key in each scope that yields that value's
// events in list order.
type auditEventPin struct {
	column, value, accountIndex, targetIndex string
}

// auditEventKeyFilter is an equality filter and the key that yields one of its values' events in list order.
type auditEventKeyFilter struct {
	column, index string
	values        []string
}

// auditEventListQuery builds one page of audit events.
type auditEventListQuery struct {
	callerAccountID string
	filter          *domain.ListAuditEventsFilter
	dir             pagination.Direction
	cursor          *pagination.StringCursor
	limit           int32
	args            []any
}

// buildAuditEventListQuery picks the page from audit_event alone, one keyset branch per scope column, and joins the enrichment onto it.
// `account_id = ? OR target_account_id = ?` walks neither scope key in order, so MySQL would filesort every event the account can see; the UNION drops an event both branches return.
func buildAuditEventListQuery(
	dir pagination.Direction,
	callerAccountID string,
	f *domain.ListAuditEventsFilter,
	includeChanges, includeMetadata bool,
	cursor *pagination.StringCursor,
	limit int32,
) (string, []any) {
	q := &auditEventListQuery{callerAccountID: callerAccountID, filter: f, dir: dir, cursor: cursor, limit: limit}
	q.args = []any{includeChanges, includeMetadata}

	var b strings.Builder
	b.WriteString("SELECT ")
	b.WriteString(strings.Join(auditEventListColumns, ", "))
	b.WriteString(" FROM (SELECT ks.id FROM (")
	for i, pin := range q.searchPins() {
		if i > 0 {
			b.WriteString(" UNION ")
		}
		q.writeAccountBranches(&b, pin)
		b.WriteString(" UNION ")
		q.writeTargetBranch(&b, pin)
	}
	b.WriteString(") ks")
	q.writeOrderAndLimit(&b, "ks.")
	b.WriteString(") page")
	// Unsorted: sorting rows that carry the JSON columns can exhaust the sort buffer, so sortAuditEventPage restores keyset order.
	b.WriteString(" JOIN audit_event ae ON ae.id = page.id")
	b.WriteString(" LEFT JOIN `user` u ON ae.actor_id = u.id AND ae.identity_type = 'user'")
	b.WriteString(" LEFT JOIN api_key ak ON ae.actor_id = ak.type_id AND ae.identity_type = 'api_key'")
	b.WriteString(" LEFT JOIN idempotency_key ik ON ae.idempotency_key_id = ik.type_id")
	b.WriteString(" LEFT JOIN account a ON ae.target_account_id = a.id")
	return b.String(), q.args
}

// keyFilters are the filters a key can pin within the acting account's events, most selective first.
func (q *auditEventListQuery) keyFilters() []auditEventKeyFilter {
	f := q.filter
	filters := []auditEventKeyFilter{
		{column: "ae.resource_id", index: auditEventResourceIDIndex, values: f.ResourceIDs},
		{column: "ae.actor_id", index: auditEventActorIDIndex, values: f.ActorIDs},
		{column: "ae.target_account_id", index: auditEventTargetAccountIndex, values: f.TargetAccountIDs},
		{column: "ae.resource_type", index: auditEventResourceTypeIndex, values: f.ResourceTypes},
		{column: "ae.action", index: auditEventActionIndex, values: f.Actions},
	}
	for i := range filters {
		filters[i].values = slices.Compact(slices.Sorted(slices.Values(filters[i].values)))
	}
	return filters
}

// searchPins are the columns a search matches, each pinned to the term. A search matches the column
// exactly: a resource or request id as given, a resource type or action as its code ("Sales Order" finds
// sales_order). Every pin reads its own key in both scopes, so a search that matches nothing reads
// nothing. With no search there is one empty pin and the branches read as the other filters choose.
func (q *auditEventListQuery) searchPins() []auditEventPin {
	if q.filter.Query == nil || strings.TrimSpace(*q.filter.Query) == "" {
		return []auditEventPin{{}}
	}
	term := strings.TrimSpace(*q.filter.Query)
	code := auditEventSearchCode(term)
	return []auditEventPin{
		{column: "ae.resource_id", value: term, accountIndex: auditEventResourceIDIndex, targetIndex: auditEventTargetResourceIDIndex},
		{column: "ae.request_id", value: term, accountIndex: auditEventRequestIDIndex, targetIndex: auditEventTargetRequestIDIndex},
		{column: "ae.resource_type", value: code, accountIndex: auditEventResourceTypeIndex, targetIndex: auditEventTargetResourceTypeIndex},
		{column: "ae.action", value: code, accountIndex: auditEventActionIndex, targetIndex: auditEventTargetActionIndex},
	}
}

// auditEventSearchCode turns a searched name into the code form resource types and actions are stored in.
func auditEventSearchCode(term string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(term), func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	}), "_")
}

func (q *auditEventListQuery) hasRootFilter() bool {
	return q.filter.RootResourceType != "" && q.filter.RootResourceID != ""
}

// writeAccountBranches reads the acting account's events through keys that yield them in list order; a few values of the leading filter are one arm per value, since no key yields several in order.
// The unfiltered key is offered only alone: beside a filter's key, the planner trades that key's range for a full scan of it.
func (q *auditEventListQuery) writeAccountBranches(b *strings.Builder, pin auditEventPin) {
	filters := q.keyFilters()
	// A pinned search column reads its own key alone: the other filters are rare beside a matching id and
	// dense beside a matching type or action, and either way the pinned key yields the page in order.
	if pin.column != "" {
		q.writeBranch(b, "account_id", []string{pin.accountIndex}, filters, -1, "", pin)
		return
	}
	split := -1
	for i, f := range filters {
		if n := len(f.values); n > 1 && n <= auditEventMaxArms {
			split = i
		}
		if len(f.values) > 0 {
			break
		}
	}
	var indexes, long []string
	for i, f := range filters {
		switch {
		case len(f.values) == 1 || i == split:
			indexes = append(indexes, f.index)
		case len(f.values) > auditEventMaxArms:
			long = append(long, f.index)
		}
	}
	if q.hasRootFilter() {
		indexes = append(indexes, auditEventRootIndex)
	}
	switch {
	case len(indexes) > 0:
		indexes = append(indexes, long...)
	case len(long) > 0:
		indexes = long
	default:
		indexes = []string{auditEventAccountIndex}
	}

	if split < 0 {
		q.writeBranch(b, "account_id", indexes, filters, -1, "", pin)
		return
	}
	for i, v := range filters[split].values {
		if i > 0 {
			b.WriteString(" UNION ")
		}
		q.writeBranch(b, "account_id", indexes, filters, split, v, pin)
	}
}

// writeTargetBranch reads the targeted account's events. Its scope key pins no filter, so the keys that pin one across accounts are offered beside it for a rare value.
func (q *auditEventListQuery) writeTargetBranch(b *strings.Builder, pin auditEventPin) {
	f := q.filter
	if pin.column != "" {
		q.writeBranch(b, "target_account_id", []string{pin.targetIndex}, q.keyFilters(), -1, "", pin)
		return
	}
	// A resource or root filter pins rows the scope key would only reach by walking every event the
	// account was acted upon in; their keys yield them in order instead.
	if len(f.ResourceIDs) == 1 {
		q.writeBranch(b, "target_account_id", []string{auditEventTargetResourceIDIndex}, q.keyFilters(), -1, "", pin)
		return
	}
	if q.hasRootFilter() {
		q.writeBranch(b, "target_account_id", []string{auditEventTargetRootIndex}, q.keyFilters(), -1, "", pin)
		return
	}
	indexes := []string{auditEventTargetAccountIndex}
	if len(f.ActorAccountIDs) > 0 {
		indexes = append(indexes, auditEventAccountIndex)
	}
	if len(f.ActorIDs) > 0 {
		indexes = append(indexes, auditEventActorIndex)
	}
	if len(f.ResourceTypes) > 0 {
		indexes = append(indexes, auditEventResourceIndex)
	}
	if len(f.ResourceIDs) > 1 {
		indexes = append(indexes, auditEventTargetResourceIDIndex)
	}
	q.writeBranch(b, "target_account_id", indexes, q.keyFilters(), -1, "", pin)
}

// writeBranch writes one parenthesized keyset branch selecting the page's keys; filters[split] is pinned to value.
func (q *auditEventListQuery) writeBranch(b *strings.Builder, scopeColumn string, indexes []string, filters []auditEventKeyFilter, split int, value string, pin auditEventPin) {
	b.WriteString("(SELECT ae.id, ae.type_id, ae.occurred_at FROM audit_event ae FORCE INDEX (")
	b.WriteString(strings.Join(indexes, ", "))
	b.WriteString(") WHERE ae." + scopeColumn + " = ?")
	q.args = append(q.args, q.callerAccountID)
	if pin.column != "" {
		b.WriteString(" AND " + pin.column + " = ?")
		q.args = append(q.args, pin.value)
	}
	for i, f := range filters {
		if i == split {
			b.WriteString(" AND " + f.column + " = ?")
			q.args = append(q.args, value)
			continue
		}
		q.writeIn(b, f.column, f.values)
	}
	q.writeResidualFilters(b)
	q.writeCursor(b)
	q.writeOrderAndLimit(b, "ae.")
	b.WriteString(")")
}

func (q *auditEventListQuery) writeIn(b *strings.Builder, column string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString(" AND " + column + " IN (" + placeholders(len(values)) + ")")
	for _, v := range values {
		q.args = append(q.args, v)
	}
}

func (q *auditEventListQuery) writeResidualFilters(b *strings.Builder) {
	f := q.filter
	if q.hasRootFilter() {
		b.WriteString(" AND ae.root_resource_type = ? AND ae.root_resource_id = ?")
		q.args = append(q.args, f.RootResourceType, f.RootResourceID)
	}
	q.writeIn(b, "ae.account_id", f.ActorAccountIDs)
	q.writeIn(b, "ae.identity_type", f.ActorTypes)
	if f.StartDate != nil {
		b.WriteString(" AND ae.occurred_at >= ?")
		q.args = append(q.args, *f.StartDate)
	}
	if f.EndDate != nil {
		b.WriteString(" AND ae.occurred_at <= ?")
		q.args = append(q.args, *f.EndDate)
	}
}

// writeCursor bounds a page past the cursor: forward pages older events, backward newer.
func (q *auditEventListQuery) writeCursor(b *strings.Builder) {
	if q.cursor == nil {
		return
	}
	if q.dir == pagination.DirectionBackward {
		b.WriteString(" AND (ae.occurred_at > ? OR (ae.occurred_at = ? AND ae.type_id > ?))")
	} else {
		b.WriteString(" AND (ae.occurred_at < ? OR (ae.occurred_at = ? AND ae.type_id < ?))")
	}
	q.args = append(q.args, q.cursor.OccurredAt, q.cursor.OccurredAt, q.cursor.ID)
}

func (q *auditEventListQuery) writeOrderAndLimit(b *strings.Builder, alias string) {
	order := " DESC"
	if q.dir == pagination.DirectionBackward {
		order = " ASC"
	}
	b.WriteString(" ORDER BY " + alias + "occurred_at" + order + ", " + alias + "type_id" + order + " LIMIT ?")
	q.args = append(q.args, q.limit)
}

// sortAuditEventPage puts a page back in query order: newest first going forward, oldest first going backward.
// Ids are a lowercase prefix, an underscore and lowercase alphanumerics, so byte order agrees with the column's collation.
func sortAuditEventPage(events []*domain.AuditEventRead, dir pagination.Direction) {
	slices.SortFunc(events, func(a, b *domain.AuditEventRead) int {
		c := a.OccurredAt.Compare(b.OccurredAt)
		if c == 0 {
			c = strings.Compare(a.ID, b.ID)
		}
		if dir == pagination.DirectionBackward {
			return c
		}
		return -c
	})
}

func scanAuditEventListRows(rows *sql.Rows) ([]*domain.AuditEventRead, error) {
	var out []*domain.AuditEventRead
	for rows.Next() {
		var r sqlc.FindAuditEventByIDRow
		if err := rows.Scan(
			&r.TypeID, &r.ActorID, &r.ActorType, &r.IdentityType, &r.AccountID,
			&r.Action, &r.ResourceType, &r.ResourceID, &r.Changes, &r.Metadata,
			&r.ServiceName, &r.RequestID, &r.IdempotencyKeyID, &r.SourceIp,
			&r.OccurredAt, &r.CreatedAt, &r.TargetAccountID,
			&r.AccountName, &r.AccountCreatedAt, &r.AccountUpdatedAt,
			&r.UserName, &r.UserEmail, &r.ApiKeyName, &r.ApiKeyRedactedValue, &r.IdempotencyKey,
		); err != nil {
			return nil, err
		}
		out = append(out, mapAuditEventRowToRead(&r))
	}
	return out, rows.Err()
}
