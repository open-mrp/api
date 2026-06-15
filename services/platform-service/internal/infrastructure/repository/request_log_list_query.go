package repository

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/pagination"
)

// routeParamPattern matches a single `{param_name}` path-parameter token.
//
// The stored normalized_route comes from the Go router's registered templates,
// which use snake_case param names (e.g. `{unit_group_id}`). Callers — notably
// the dashboard endpoint filter — derive their templates from the Stainless
// public OpenAPI spec, which camelCases multi-word path params (`{unitGroupId}`)
// and is otherwise free to rename them. Param names are cosmetic (path params
// are positional), so we collapse every token to a bare `{}` on both the filter
// input and the stored value before comparing, matching on route *shape*.
var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// normalizeRouteParams collapses every `{param}` token in a route template to a
// bare `{}` placeholder so the comparison ignores param-name spelling/casing.
func normalizeRouteParams(route string) string {
	return routeParamPattern.ReplaceAllString(route, "{}")
}

// normalizedRouteColumnExpr is the SQL counterpart to normalizeRouteParams: it
// collapses `{param}` tokens in the stored rl.normalized_route column to `{}`
// with the same shape so the IN comparison lines up. The doubled backslashes
// escape the regex braces for MySQL's REGEXP_REPLACE (ICU) engine.
const normalizedRouteColumnExpr = `REGEXP_REPLACE(rl.normalized_route, '\\{[^}]+\\}', '{}')`

// requestLogRLBaseColumns is the request_log SELECT list used by every list
// mode. Kept as a single slice so SELECT list and row scanners can't drift.
//
// JSON columns (query_json, request_body_json, response_body_json) are emitted
// as COALESCE(CASE WHEN ? THEN col ELSE NULL END, ”) so includeQueryJson /
// includeRequestBodyJson / includeResponseBodyJson boolean args control whether
// the payload is returned without changing the column count.
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

// buildListQuery assembles the dynamic list SQL for a request_log listing.
// Filter predicates are omitted entirely when the caller did not supply a
// value — no OR-sentinel or CASE-WHEN wrappers — so MySQL sees only the
// predicates that actually narrow the result set.
//
// limit is applied inside the ORDER BY block; callers are expected to pass
// limit+1 to support "has next page" detection.
func buildListQuery(
	mode queryMode,
	dir pagination.Direction,
	callerAccountID string,
	f *domain.ListRequestLogsFilter,
	includeQueryJSON, includeRequestBody, includeResponseBody bool,
	cursor *pagination.StringCursor,
	limit int32,
) (string, []any) {
	var inner strings.Builder
	var args []any

	// SELECT + FROM for the inner query (for actor mode this is the derived
	// table body; for base/full it's the final query).
	inner.WriteString("SELECT ")
	inner.WriteString(strings.Join(requestLogRLBaseColumns, ", "))
	args = append(args, includeQueryJSON, includeRequestBody, includeResponseBody)

	switch mode {
	case queryModeActor, queryModeFull:
		// Both actor and full mode wrap the inner block in a derived table (see
		// the wrapper below). The inner block selects only the rl.* base columns
		// from request_log alone — actor_id is the raw value the API exposes, so
		// no account_user translation join is needed here. Keeping the expensive
		// user / api_key / role / account / idempotency_key joins out of the inner
		// block lets the WHERE + ORDER BY + LIMIT run against request_log alone, so
		// MySQL/Vitess uses the (target_account_id, occurred_at DESC, id DESC)
		// index and LIMITs before any enrichment join happens.
		inner.WriteString(" FROM request_log rl")
	case queryModeBase:
		// Base mode still pulls idempotency_key via a LEFT JOIN; that join is
		// indexed and cheap.
		inner.WriteString(", ik.idempotency_key")
		inner.WriteString(
			" FROM request_log rl" +
				" LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id",
		)
	}

	// WHERE — only emit predicates the caller supplied. The security scope
	// returns every log where the caller's account is either the acting account
	// (rl.account_id) or the account the request targeted (rl.target_account_id).
	// MySQL/Vitess satisfies the OR via index_merge over the per-side cursor
	// indexes: (account_id, occurred_at DESC, id DESC) and
	// (target_account_id, occurred_at DESC, id DESC).
	inner.WriteString(" WHERE (rl.account_id = ? OR rl.target_account_id = ?)")
	args = append(args, callerAccountID, callerAccountID)

	if f.Query != nil && *f.Query != "" {
		like := "%" + db.EscapeLike(*f.Query) + "%"
		// Match the log's own id (exact) plus a substring search across the
		// request route (both the literal path and the normalized route) and the
		// error message. Searching rl.path lets a caller paste a resource id that
		// appeared in a URL (e.g. /v1/catalog/items/it_123) and find every log
		// that touched it; rl.normalized_route covers route-template searches
		// (e.g. "catalog/items").
		inner.WriteString(" AND (rl.id = ? OR rl.path LIKE ? OR rl.normalized_route LIKE ? OR rl.error_message LIKE ?)")
		args = append(args, *f.Query, like, like, like)
	}
	if f.StartDate != nil {
		inner.WriteString(" AND rl.occurred_at >= ?")
		args = append(args, *f.StartDate)
	}
	if f.EndDate != nil {
		inner.WriteString(" AND rl.occurred_at <= ?")
		args = append(args, *f.EndDate)
	}
	if len(f.Methods) > 0 {
		inner.WriteString(" AND rl.method IN (")
		inner.WriteString(placeholders(len(f.Methods)))
		inner.WriteString(")")
		for _, m := range f.Methods {
			args = append(args, m)
		}
	}
	if len(f.StatusCodes) > 0 || len(f.StatusCodeClasses) > 0 {
		// Specific codes and whole classes are OR'd together (then AND'd with the
		// rest of the filters): status_codes=401 + status_code_classes=5 matches
		// 401 and any 5xx. Classes use FLOOR(status_code/100) so a class matches
		// every code in its range, not just the curated ones the UI lists.
		inner.WriteString(" AND (")
		if len(f.StatusCodes) > 0 {
			inner.WriteString("rl.status_code IN (")
			inner.WriteString(placeholders(len(f.StatusCodes)))
			inner.WriteString(")")
			for _, sc := range f.StatusCodes {
				args = append(args, sc)
			}
		}
		if len(f.StatusCodeClasses) > 0 {
			if len(f.StatusCodes) > 0 {
				inner.WriteString(" OR ")
			}
			inner.WriteString("FLOOR(rl.status_code / 100) IN (")
			inner.WriteString(placeholders(len(f.StatusCodeClasses)))
			inner.WriteString(")")
			for _, c := range f.StatusCodeClasses {
				args = append(args, c)
			}
		}
		inner.WriteString(")")
	}
	if len(f.ErrorCodes) > 0 {
		inner.WriteString(" AND rl.error_code IN (")
		inner.WriteString(placeholders(len(f.ErrorCodes)))
		inner.WriteString(")")
		for _, ec := range f.ErrorCodes {
			args = append(args, ec)
		}
	}
	if len(f.ActorAccountIDs) > 0 {
		// Narrow to logs whose acting account is one of these (within scope).
		inner.WriteString(" AND rl.account_id IN (")
		inner.WriteString(placeholders(len(f.ActorAccountIDs)))
		inner.WriteString(")")
		for _, id := range f.ActorAccountIDs {
			args = append(args, id)
		}
	}
	if len(f.TargetAccountIDs) > 0 {
		// Narrow to logs whose target account is one of these (within scope).
		inner.WriteString(" AND rl.target_account_id IN (")
		inner.WriteString(placeholders(len(f.TargetAccountIDs)))
		inner.WriteString(")")
		for _, id := range f.TargetAccountIDs {
			args = append(args, id)
		}
	}
	if len(f.ActorIDs) > 0 {
		// Filter on the bare rl.actor_id column so the predicate is sargable and
		// can use the (target_account_id, actor_id, occurred_at DESC, id DESC)
		// index. actor_id stores the raw id the API exposes (user_id for user
		// actors, api_key.type_id for api_key actors), so the caller's ids match
		// directly — no translation needed.
		inner.WriteString(" AND rl.actor_id IN (")
		inner.WriteString(placeholders(len(f.ActorIDs)))
		inner.WriteString(")")
		for _, id := range f.ActorIDs {
			args = append(args, id)
		}
	}
	if len(f.ActorTypes) > 0 {
		inner.WriteString(" AND rl.identity_type IN (")
		inner.WriteString(placeholders(len(f.ActorTypes)))
		inner.WriteString(")")
		for _, t := range f.ActorTypes {
			args = append(args, t)
		}
	}
	if len(f.NormalizedRoutes) > 0 {
		// Compare on route shape (param names collapsed to `{}`) so the filter
		// is immune to param-name drift between the stored router templates and
		// the spec-derived templates callers send. See normalizeRouteParams.
		inner.WriteString(" AND ")
		inner.WriteString(normalizedRouteColumnExpr)
		inner.WriteString(" IN (")
		inner.WriteString(placeholders(len(f.NormalizedRoutes)))
		inner.WriteString(")")
		for _, r := range f.NormalizedRoutes {
			args = append(args, normalizeRouteParams(r))
		}
	}
	if len(f.Hosts) > 0 {
		inner.WriteString(" AND rl.host IN (")
		inner.WriteString(placeholders(len(f.Hosts)))
		inner.WriteString(")")
		for _, h := range f.Hosts {
			args = append(args, h)
		}
	}
	if f.MinLatencyUs != nil {
		inner.WriteString(" AND rl.latency_us >= ?")
		args = append(args, *f.MinLatencyUs)
	}
	if f.PublicEndpoint != nil {
		inner.WriteString(" AND rl.public_endpoint = ?")
		args = append(args, *f.PublicEndpoint)
	}
	if f.IdempotencyKey != nil && *f.IdempotencyKey != "" {
		inner.WriteString(
			" AND EXISTS (SELECT 1 FROM idempotency_key ik2 WHERE ik2.type_id = rl.idempotency_key_id AND ik2.idempotency_key = ?)",
		)
		args = append(args, *f.IdempotencyKey)
	}

	// Cursor predicate — matches the direction semantics used by the previous
	// sqlc queries: forward pages older (DESC), backward pages newer (ASC).
	if cursor != nil {
		switch dir {
		case pagination.DirectionBackward:
			inner.WriteString(" AND (rl.occurred_at > ? OR (rl.occurred_at = ? AND rl.id > ?))")
			args = append(args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
		default:
			inner.WriteString(" AND (rl.occurred_at < ? OR (rl.occurred_at = ? AND rl.id < ?))")
			args = append(args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
		}
	}

	if dir == pagination.DirectionBackward {
		inner.WriteString(" ORDER BY rl.occurred_at ASC, rl.id ASC LIMIT ?")
	} else {
		inner.WriteString(" ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")
	}
	args = append(args, limit)

	if mode == queryModeBase {
		return inner.String(), args
	}

	// Actor and full mode: wrap the inner block as a derived table so
	// MySQL/Vitess picks up the (target_account_id, occurred_at DESC, id DESC)
	// index, runs the LIMIT, *then* nested-loop joins the enrichment tables on
	// only the (≤ limit) matching rows. Without this the optimizer filesorts the
	// whole target_account_id partition before the LIMIT, which times out on a
	// large request_log table.
	//
	// actor_id is the raw actor key exposed by the API: the user_id for a user
	// actor (the outer user join keys on u.id = rl.actor_id, and the account_user
	// join used for the role keys on au.user_id = rl.actor_id) or the
	// api_key.type_id for an api_key actor (the api_key join keys on ak.type_id).
	var outer strings.Builder
	outer.WriteString("SELECT " + derivedRequestLogRLColumns + ", ")
	outer.WriteString("u.email AS user_email, u.name AS user_name, ")
	outer.WriteString("ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value, ak.name AS api_key_name")
	if mode == queryModeFull {
		outer.WriteString(
			", au.role_id AS user_role_id, r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code, " +
				"r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code, " +
				"a.name AS account_name, a.created_at AS account_created_at, a.updated_at AS account_updated_at, ik.idempotency_key",
		)
	}
	outer.WriteString(" FROM (")
	outer.WriteString(inner.String())
	outer.WriteString(
		") rl" +
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
	if dir == pagination.DirectionBackward {
		outer.WriteString(" ORDER BY rl.occurred_at ASC, rl.id ASC")
	} else {
		outer.WriteString(" ORDER BY rl.occurred_at DESC, rl.id DESC")
	}
	return outer.String(), args
}

// derivedRequestLogRLColumns is the rl.* projection the actor/full outer query
// selects from the derived table. It mirrors requestLogRLBaseColumns column-for
// -column (the JSON columns are already COALESCE'd inside the derived table, so
// here they are plain rl.<alias> references).
const derivedRequestLogRLColumns = "rl.id, rl.method, rl.host, rl.path, rl.normalized_route, " +
	"rl.query_json, rl.status_code, rl.latency_us, rl.api_version, rl.actor_id, " +
	"rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent, " +
	"rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at, " +
	"rl.idempotency_key_id, rl.request_body_json, rl.response_body_json, rl.target_account_id"

// placeholders returns "?, ?, ?, ..." with n placeholders.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}

// scanBaseListRows scans *sql.Rows produced by the queryModeBase builder output
// into domain objects, reusing the base FindByID mapper for column → field
// translation.
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

// scanActorListRows scans *sql.Rows produced by the queryModeActor builder
// output. Actor mode selects a subset of the full column set and leaves role /
// account fields unset — mapRowToRequestLogRead treats their zero NullStrings
// as nil in the domain object, matching the old per-variant mappers.
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
