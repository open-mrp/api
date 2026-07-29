package repository

import (
	"database/sql"
	"regexp"
	"slices"
	"strings"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/pagination"
)

// routeParamPattern matches a single `{param_name}` path-parameter token.
//
// The stored normalized_route comes from the Go router's registered templates, which use snake_case param names (e.g. `{unit_group_id}`). Callers — notably the dashboard endpoint filter — derive their templates from the Stainless public OpenAPI spec, which camelCases multi-word path params (`{unitGroupId}`) and is otherwise free to rename them. Param names are cosmetic (path params are positional), so we collapse every token to a bare `{}` on both the filter input and the stored value before comparing, matching on route *shape*.
var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// normalizeRouteParams collapses every `{param}` token in a route template to a bare `{}` placeholder so the comparison ignores param-name spelling/casing.
func normalizeRouteParams(route string) string {
	return routeParamPattern.ReplaceAllString(route, "{}")
}

// normalizedRouteColumnExpr is the SQL counterpart to normalizeRouteParams: it collapses `{param}` tokens in the stored rl.normalized_route column to `{}` with the same shape so the IN comparison lines up. The doubled backslashes escape the regex braces for MySQL's REGEXP_REPLACE (ICU) engine.
const normalizedRouteColumnExpr = `REGEXP_REPLACE(rl.normalized_route, '\\{[^}]+\\}', '{}')`

// requestLogRLBaseColumns is the request_log SELECT list used by every list mode. Kept as a single slice so SELECT list and row scanners can't drift.
//
// JSON columns (query_json, request_body_json, response_body_json) are emitted as COALESCE(CASE WHEN ? THEN col ELSE NULL END, ”) so includeQueryJson / includeRequestBodyJson / includeResponseBodyJson boolean args control whether the payload is returned without changing the column count.
//
// This projection must only ever appear in the outermost SELECT, after all ORDER BYs have run — see buildListQuery.
var requestLogRLBaseColumns = []string{
	"rl.id",
	"rl.method",
	"rl.host",
	"rl.path",
	"rl.normalized_route",
	"COALESCE(CASE WHEN ? THEN rl.query_json ELSE NULL END, '') AS query_json",
	"rl.status_code",
	"rl.latency_us",
	"rl.api_version",
	"rl.actor_id AS actor_id",
	"rl.actor_type",
	"rl.identity_type",
	"rl.client_ip_string",
	"rl.user_agent",
	"rl.referrer",
	"rl.error_code",
	"rl.error_message",
	"rl.occurred_at",
	"rl.created_at",
	"rl.idempotency_key_id",
	"COALESCE(CASE WHEN ? THEN rl.request_body_json ELSE NULL END, '') AS request_body_json",
	"COALESCE(CASE WHEN ? THEN rl.response_body_json ELSE NULL END, '') AS response_body_json",
	"rl.target_account_id",
}

// buildListQuery assembles the dynamic list SQL for a request_log listing. Filter predicates are omitted entirely when the caller did not supply a value — no OR-sentinel or CASE-WHEN wrappers — so MySQL sees only the predicates that actually narrow the result set.
//
// The dual-scope security filter (a log is visible when the caller's account is either the acting account `rl.account_id` OR the request's target `rl.target_account_id`) is expressed as a UNION of two single-scope keyset branches rather than `WHERE (account_id = ? OR target_account_id = ?)`. The OR form forces MySQL/Vitess into an index_merge that cannot satisfy `ORDER BY occurred_at DESC`, so it filesorts the caller's entire partition on every page — the second page (and any page with selective filters) times out on a large request_log. Splitting the scope lets each branch walk its own `(scope, occurred_at DESC, id DESC)` composite index in order, use the cursor as a real range bound, and stop at LIMIT before any enrichment join. UNION (not UNION ALL) drops the duplicate that appears for rows whose acting account IS the target account (the common single-account case); branches select the unique rl.id, so two genuinely different rows can never collapse.
//
// Every sorted set — each branch's ORDER BY, the UNION's dedupe temp table, and the merged re-sort — carries only (id, occurred_at). The wide columns, in particular the three JSON payload columns, are projected by the outermost SELECT via a primary-key join back to request_log AFTER all ordering has happened. MySQL's filesort cannot pack JSON/TEXT addon columns: one oversized request/response body in a sort row blows sort_buffer_size and the whole query dies with error 1038 "Out of sort memory" (it cannot spill a single row to disk), so payload columns must never pass through an ORDER BY or a UNION temp table. This also makes each branch fully covered by its (scope, occurred_at DESC, id DESC) index. The trade-off: the outermost query has no ORDER BY (the join back to request_log does not guarantee it would preserve derived-table order anyway), so the caller must re-sort the ≤ limit+1 scanned rows by (occurred_at, id) in Go — see sortListResults.
//
// limit is applied inside each branch and again on the merged id set; callers pass limit+1 to support "has next page" detection. Each branch returns at most limit+1 rows, so the temp table MySQL builds for the UNION is bounded at 2*(limit+1) tiny rows regardless of table size, and the primary-key join back fans in at most limit+1 rows.
func buildListQuery(
	mode queryMode,
	dir pagination.Direction,
	callerAccountID string,
	f *domain.ListRequestLogsFilter,
	includeQueryJSON, includeRequestBody, includeResponseBody bool,
	cursor *pagination.StringCursor,
	limit int32,
) (string, []any) {
	// unionArgs collects binds for the id-page derived table only; the JSON-include booleans bind to the outer SELECT list, which precedes the derived table in the final SQL text, so the two groups are concatenated in text order at the end.
	var unionArgs []any

	// writeScopeBranch emits one keyset branch scoped to a single account column (rl.account_id or rl.target_account_id). It selects only the keyset pair (id, occurred_at) from request_log alone — no payload columns, no enrichment joins — so the WHERE + ORDER BY + LIMIT ride the branch's (scope, occurred_at DESC, id DESC) composite as a covering index and any residual-filter filesort handles only tiny rows. It appends the branch's bind args (scope id, filter values, cursor values, LIMIT) to unionArgs in the exact left-to-right order the placeholders appear.
	writeScopeBranch := func(scopeColumn string) string {
		var b strings.Builder
		b.WriteString("SELECT rl.id, rl.occurred_at")
		b.WriteString(" FROM request_log rl")
		b.WriteString(" WHERE rl.")
		b.WriteString(scopeColumn)
		b.WriteString(" = ?")
		unionArgs = append(unionArgs, callerAccountID)
		// Hidden logs (e.g. high-frequency polling endpoints flagged HideFromRequestLog) are persisted but omitted from listings. hidden is low-cardinality, so this rides the branch's cursor index as a residual filter rather than needing its own index.
		b.WriteString(" AND rl.hidden = FALSE")
		writeFilterPredicates(&b, &unionArgs, f)
		writeCursorPredicate(&b, &unionArgs, dir, cursor)
		writeScopeOrderAndLimit(&b, &unionArgs, "rl.", dir, limit)
		return b.String()
	}

	// Merge the two scope branches, restore the global keyset order over the ≤ 2*(limit+1) merged ids (each branch is individually ordered + capped, but UNION does not preserve order), and trim to limit+1. This is the query's final sort; everything after it is a key lookup.
	var page strings.Builder
	page.WriteString("SELECT ks.id, ks.occurred_at FROM (")
	page.WriteString("(" + writeScopeBranch("account_id") + ") UNION (" + writeScopeBranch("target_account_id") + ")")
	page.WriteString(") ks")
	writeScopeOrderAndLimit(&page, &unionArgs, "ks.", dir, limit)

	var outer strings.Builder
	outer.WriteString("SELECT ")
	outer.WriteString(strings.Join(requestLogRLBaseColumns, ", "))
	switch mode {
	case queryModeBase:
		// Base mode pulls idempotency_key via a LEFT JOIN over the page; that join is indexed and cheap.
		outer.WriteString(", ik.idempotency_key")
	case queryModeActor, queryModeFull:
		// actor_id is the raw actor key exposed by the API: the user_id for a user actor (the outer user join keys on u.id = rl.actor_id, and the account_user join used for the role keys on au.user_id = rl.actor_id) or the api_key.type_id for an api_key actor (the api_key join keys on ak.type_id).
		outer.WriteString(", u.email AS user_email, u.name AS user_name, ")
		outer.WriteString("ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value, ak.name AS api_key_name")
		if mode == queryModeFull {
			outer.WriteString(
				", au.role_id AS user_role_id, r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code, " +
					"r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code, " +
					"a.name AS account_name, a.created_at AS account_created_at, a.updated_at AS account_updated_at, ik.idempotency_key",
			)
		}
	}
	outer.WriteString(" FROM (")
	outer.WriteString(page.String())
	outer.WriteString(") page")
	outer.WriteString(" JOIN request_log rl ON rl.id = page.id")
	switch mode {
	case queryModeBase:
		outer.WriteString(" LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id")
	case queryModeActor, queryModeFull:
		outer.WriteString(
			" LEFT JOIN `user` u ON u.id = rl.actor_id AND rl.identity_type = 'user'" +
				" LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'",
		)
		if mode == queryModeFull {
			outer.WriteString(
				" LEFT JOIN account_user au ON au.user_id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'" +
					" LEFT JOIN role r_user ON au.role_id = r_user.id" +
					" LEFT JOIN role r_key ON ak.role_id = r_key.id" +
					" LEFT JOIN account a ON rl.target_account_id = a.id" +
					" LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id",
			)
		}
	}

	// Final args in SQL text order: the outer SELECT list's three JSON-include booleans come first, then the id-page derived table's binds.
	args := make([]any, 0, len(unionArgs)+3)
	args = append(args, includeQueryJSON, includeRequestBody, includeResponseBody)
	args = append(args, unionArgs...)
	return outer.String(), args
}

// writeFilterPredicates appends the caller-supplied filter predicates (and their bind args) shared by both scope branches. Predicates are omitted entirely when the caller did not supply a value, so MySQL sees only the predicates that actually narrow the result set. Both branches must emit identical filter SQL so the UNION column/placeholder shapes line up.
func writeFilterPredicates(sb *strings.Builder, args *[]any, f *domain.ListRequestLogsFilter) {
	if f.Query != nil && *f.Query != "" {
		like := "%" + db.EscapeLike(*f.Query) + "%"
		// Match the log's own id (exact) plus a substring search across the request route (both the literal path and the normalized route) and the error message. Searching rl.path lets a caller paste a resource id that appeared in a URL (e.g. /v1/catalog/items/it_123) and find every log that touched it; rl.normalized_route covers route-template searches (e.g. "catalog/items").
		sb.WriteString(" AND (rl.id = ? OR rl.path LIKE ? OR rl.normalized_route LIKE ? OR rl.error_message LIKE ?)")
		*args = append(*args, *f.Query, like, like, like)
	}
	if f.StartDate != nil {
		sb.WriteString(" AND rl.occurred_at >= ?")
		*args = append(*args, *f.StartDate)
	}
	if f.EndDate != nil {
		sb.WriteString(" AND rl.occurred_at <= ?")
		*args = append(*args, *f.EndDate)
	}
	if len(f.Methods) > 0 {
		sb.WriteString(" AND rl.method IN (")
		sb.WriteString(placeholders(len(f.Methods)))
		sb.WriteString(")")
		for _, m := range f.Methods {
			*args = append(*args, m)
		}
	}
	if len(f.StatusCodes) > 0 || len(f.StatusCodeClasses) > 0 {
		// Specific codes and whole classes are OR'd together (then AND'd with the rest of the filters): status_codes=401 + status_code_classes=5 matches 401 and any 5xx. Classes use FLOOR(status_code/100) so a class matches every code in its range, not just the curated ones the UI lists.
		sb.WriteString(" AND (")
		if len(f.StatusCodes) > 0 {
			sb.WriteString("rl.status_code IN (")
			sb.WriteString(placeholders(len(f.StatusCodes)))
			sb.WriteString(")")
			for _, sc := range f.StatusCodes {
				*args = append(*args, sc)
			}
		}
		if len(f.StatusCodeClasses) > 0 {
			if len(f.StatusCodes) > 0 {
				sb.WriteString(" OR ")
			}
			sb.WriteString("FLOOR(rl.status_code / 100) IN (")
			sb.WriteString(placeholders(len(f.StatusCodeClasses)))
			sb.WriteString(")")
			for _, c := range f.StatusCodeClasses {
				*args = append(*args, c)
			}
		}
		sb.WriteString(")")
	}
	if len(f.ErrorCodes) > 0 {
		sb.WriteString(" AND rl.error_code IN (")
		sb.WriteString(placeholders(len(f.ErrorCodes)))
		sb.WriteString(")")
		for _, ec := range f.ErrorCodes {
			*args = append(*args, ec)
		}
	}
	if len(f.ExcludeErrorCodes) > 0 {
		// Drop logs whose error_code is in this set. error_code is NULL for successful requests, and `NULL NOT IN (...)` is NULL (not TRUE), which would drop every 2xx row — so the IS NULL disjunct explicitly keeps them, ensuring a default "hide expired_token" filter still shows all non-error traffic.
		sb.WriteString(" AND (rl.error_code IS NULL OR rl.error_code NOT IN (")
		sb.WriteString(placeholders(len(f.ExcludeErrorCodes)))
		sb.WriteString("))")
		for _, ec := range f.ExcludeErrorCodes {
			*args = append(*args, ec)
		}
	}
	if len(f.ActorAccountIDs) > 0 {
		// Narrow to logs whose acting account is one of these (within scope).
		sb.WriteString(" AND rl.account_id IN (")
		sb.WriteString(placeholders(len(f.ActorAccountIDs)))
		sb.WriteString(")")
		for _, id := range f.ActorAccountIDs {
			*args = append(*args, id)
		}
	}
	if len(f.TargetAccountIDs) > 0 {
		// Narrow to logs whose target account is one of these (within scope).
		sb.WriteString(" AND rl.target_account_id IN (")
		sb.WriteString(placeholders(len(f.TargetAccountIDs)))
		sb.WriteString(")")
		for _, id := range f.TargetAccountIDs {
			*args = append(*args, id)
		}
	}
	if len(f.ActorIDs) > 0 {
		// Filter on the bare rl.actor_id column so the predicate is sargable and can use the (target_account_id, actor_id, occurred_at DESC, id DESC) index. actor_id stores the raw id the API exposes (user_id for user actors, api_key.type_id for api_key actors), so the caller's ids match directly — no translation needed.
		sb.WriteString(" AND rl.actor_id IN (")
		sb.WriteString(placeholders(len(f.ActorIDs)))
		sb.WriteString(")")
		for _, id := range f.ActorIDs {
			*args = append(*args, id)
		}
	}
	if len(f.ActorTypes) > 0 {
		sb.WriteString(" AND rl.identity_type IN (")
		sb.WriteString(placeholders(len(f.ActorTypes)))
		sb.WriteString(")")
		for _, t := range f.ActorTypes {
			*args = append(*args, t)
		}
	}
	if len(f.NormalizedRoutes) > 0 {
		// Compare on route shape (param names collapsed to `{}`) so the filter is immune to param-name drift between the stored router templates and the spec-derived templates callers send. See normalizeRouteParams.
		sb.WriteString(" AND ")
		sb.WriteString(normalizedRouteColumnExpr)
		sb.WriteString(" IN (")
		sb.WriteString(placeholders(len(f.NormalizedRoutes)))
		sb.WriteString(")")
		for _, r := range f.NormalizedRoutes {
			*args = append(*args, normalizeRouteParams(r))
		}
	}
	if len(f.Hosts) > 0 {
		sb.WriteString(" AND rl.host IN (")
		sb.WriteString(placeholders(len(f.Hosts)))
		sb.WriteString(")")
		for _, h := range f.Hosts {
			*args = append(*args, h)
		}
	}
	if f.MinLatencyUs != nil {
		sb.WriteString(" AND rl.latency_us >= ?")
		*args = append(*args, *f.MinLatencyUs)
	}
	if f.PublicEndpoint != nil {
		sb.WriteString(" AND rl.public_endpoint = ?")
		*args = append(*args, *f.PublicEndpoint)
	}
	if f.IdempotencyKey != nil && *f.IdempotencyKey != "" {
		sb.WriteString(
			" AND EXISTS (SELECT 1 FROM idempotency_key ik2 WHERE ik2.type_id = rl.idempotency_key_id AND ik2.idempotency_key = ?)",
		)
		*args = append(*args, *f.IdempotencyKey)
	}
}

// writeCursorPredicate appends the keyset cursor comparison (and its bind args). Direction semantics match the previous sqlc queries: forward pages older (DESC), backward pages newer (ASC).
func writeCursorPredicate(sb *strings.Builder, args *[]any, dir pagination.Direction, cursor *pagination.StringCursor) {
	if cursor == nil {
		return
	}
	switch dir {
	case pagination.DirectionBackward:
		sb.WriteString(" AND (rl.occurred_at > ? OR (rl.occurred_at = ? AND rl.id > ?))")
		*args = append(*args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	default:
		sb.WriteString(" AND (rl.occurred_at < ? OR (rl.occurred_at = ? AND rl.id < ?))")
		*args = append(*args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
}

// writeScopeOrderAndLimit appends the keyset ORDER BY and a `LIMIT ?` (binding limit) using the given table alias prefix (`"rl."` inside the scope branches, `"ks."` on the merged id page). Used in both places so the two stay in lockstep.
func writeScopeOrderAndLimit(sb *strings.Builder, args *[]any, alias string, dir pagination.Direction, limit int32) {
	if dir == pagination.DirectionBackward {
		sb.WriteString(" ORDER BY " + alias + "occurred_at ASC, " + alias + "id ASC LIMIT ?")
	} else {
		sb.WriteString(" ORDER BY " + alias + "occurred_at DESC, " + alias + "id DESC LIMIT ?")
	}
	*args = append(*args, limit)
}

// sortListResults restores keyset order over the scanned rows. The list query's outermost SELECT — the one carrying the JSON payload columns — deliberately has no ORDER BY (see buildListQuery), so row order out of the primary-key join is unspecified and pagination.BuildPageString needs its input in query order (forward = newest first, backward = oldest first; BuildPageString does the backward-page reversal itself). Ties on occurred_at break on the id column; ids are lowercase-alphanumeric, so byte order agrees with the column's utf8mb4_unicode_ci order the SQL cursor comparisons use.
func sortListResults(results []*domain.RequestLogRead, dir pagination.Direction) {
	slices.SortFunc(results, func(a, b *domain.RequestLogRead) int {
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

// placeholders returns "?, ?, ?, ..." with n placeholders.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}

// scanBaseListRows scans *sql.Rows produced by the queryModeBase builder output into domain objects, reusing the base FindByID mapper for column → field translation.
func scanBaseListRows(rows *sql.Rows) ([]*domain.RequestLogRead, error) {
	out := make([]*domain.RequestLogRead, 0, 16)
	for rows.Next() {
		var r sqlc.FindRequestLogBaseByIDRow
		if err := rows.Scan(
			&r.ID, &r.Method, &r.Host, &r.Path, &r.NormalizedRoute,
			&r.QueryJson,
			&r.StatusCode, &r.LatencyUs, &r.ApiVersion, &r.ActorID,
			&r.ActorType, &r.IdentityType, &r.ClientIpString, &r.UserAgent,
			&r.Referrer, &r.ErrorCode, &r.ErrorMessage, &r.OccurredAt, &r.CreatedAt,
			&r.IdempotencyKeyID,
			&r.RequestBodyJson, &r.ResponseBodyJson,
			&r.TargetAccountID,
			&r.IdempotencyKey,
		); err != nil {
			return nil, err
		}
		read := mapBaseRowToRequestLogRead(&r)
		read.Actor = buildMinimalActor(r.ActorID.String, r.IdentityType)
		out = append(out, read)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// scanFullListRows scans *sql.Rows produced by the queryModeFull builder output.
func scanFullListRows(rows *sql.Rows) ([]*domain.RequestLogRead, error) {
	out := make([]*domain.RequestLogRead, 0, 16)
	for rows.Next() {
		var r sqlc.FindRequestLogByIDRow
		if err := rows.Scan(
			&r.ID, &r.Method, &r.Host, &r.Path, &r.NormalizedRoute,
			&r.QueryJson,
			&r.StatusCode, &r.LatencyUs, &r.ApiVersion, &r.ActorID,
			&r.ActorType, &r.IdentityType, &r.ClientIpString, &r.UserAgent,
			&r.Referrer, &r.ErrorCode, &r.ErrorMessage, &r.OccurredAt, &r.CreatedAt,
			&r.IdempotencyKeyID,
			&r.RequestBodyJson, &r.ResponseBodyJson,
			&r.TargetAccountID,
			&r.UserEmail, &r.UserName,
			&r.ApiKeyTypeID, &r.ApiKeyRedactedValue, &r.ApiKeyName,
			&r.UserRoleID, &r.UserRoleName, &r.UserRoleTypeCode,
			&r.ApiKeyRoleID, &r.ApiKeyRoleName, &r.ApiKeyRoleTypeCode,
			&r.AccountName, &r.AccountCreatedAt, &r.AccountUpdatedAt, &r.IdempotencyKey,
		); err != nil {
			return nil, err
		}
		out = append(out, mapRowToRequestLogRead(&r))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// scanActorListRows scans *sql.Rows produced by the queryModeActor builder output. Actor mode selects a subset of the full column set and leaves role / account fields unset — mapRowToRequestLogRead treats their zero NullStrings as nil in the domain object, matching the old per-variant mappers.
func scanActorListRows(rows *sql.Rows) ([]*domain.RequestLogRead, error) {
	out := make([]*domain.RequestLogRead, 0, 16)
	for rows.Next() {
		var r sqlc.FindRequestLogByIDRow
		if err := rows.Scan(
			&r.ID, &r.Method, &r.Host, &r.Path, &r.NormalizedRoute,
			&r.QueryJson,
			&r.StatusCode, &r.LatencyUs, &r.ApiVersion, &r.ActorID,
			&r.ActorType, &r.IdentityType, &r.ClientIpString, &r.UserAgent,
			&r.Referrer, &r.ErrorCode, &r.ErrorMessage, &r.OccurredAt, &r.CreatedAt,
			&r.IdempotencyKeyID,
			&r.RequestBodyJson, &r.ResponseBodyJson,
			&r.TargetAccountID,
			&r.UserEmail, &r.UserName,
			&r.ApiKeyTypeID, &r.ApiKeyRedactedValue, &r.ApiKeyName,
		); err != nil {
			return nil, err
		}
		out = append(out, mapRowToRequestLogRead(&r))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
