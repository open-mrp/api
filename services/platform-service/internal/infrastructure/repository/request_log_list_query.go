package repository

import (
	"database/sql"
	"strings"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/pagination"
)

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
	"COALESCE(au.id, rl.actor_id) AS actor_id",
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
	targetAccountID string,
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
	case queryModeFull:
		// Full mode joins everything upfront because actor_name filters over
		// u.name / ak.name and role / account data may be requested.
		inner.WriteString(
			", u.email AS user_email, u.name AS user_name, " +
				"ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value, ak.name AS api_key_name, " +
				"au.role_id AS user_role_id, r_user.name AS user_role_name, r_user.role_type_code AS user_role_type_code, " +
				"r_key.id AS api_key_role_id, r_key.name AS api_key_role_name, r_key.role_type_code AS api_key_role_type_code, " +
				"a.name AS account_name, ik.idempotency_key",
		)
		inner.WriteString(
			" FROM request_log rl" +
				" LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'" +
				" LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'" +
				" LEFT JOIN account_user au ON au.user_id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'" +
				" LEFT JOIN role r_user ON au.role_id = r_user.id" +
				" LEFT JOIN role r_key ON ak.role_id = r_key.id" +
				" LEFT JOIN account a ON rl.target_account_id = a.id" +
				" LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id",
		)
	case queryModeBase:
		// Base mode still pulls idempotency_key via a LEFT JOIN; that join is
		// indexed and cheap.
		inner.WriteString(", ik.idempotency_key")
		inner.WriteString(
			" FROM request_log rl" +
				" LEFT JOIN account_user au ON au.user_id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'" +
				" LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id",
		)
	case queryModeActor:
		// Actor mode selects only the rl.* columns inside the derived table.
		// The outer query joins user + api_key to the already-LIMIT'd set.
		inner.WriteString(
			" FROM request_log rl" +
				" LEFT JOIN account_user au ON au.user_id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'",
		)
	}

	// WHERE — only emit predicates the caller supplied.
	inner.WriteString(" WHERE rl.target_account_id = ?")
	args = append(args, targetAccountID)

	if f.Query != nil && *f.Query != "" {
		like := "%" + *f.Query + "%"
		inner.WriteString(" AND (rl.id = ? OR rl.path LIKE ? OR rl.error_message LIKE ?)")
		args = append(args, *f.Query, like, like)
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
	if len(f.StatusCodes) > 0 {
		inner.WriteString(" AND rl.status_code IN (")
		inner.WriteString(placeholders(len(f.StatusCodes)))
		inner.WriteString(")")
		for _, sc := range f.StatusCodes {
			args = append(args, sc)
		}
	}
	if len(f.ErrorCodes) > 0 {
		inner.WriteString(" AND rl.error_code IN (")
		inner.WriteString(placeholders(len(f.ErrorCodes)))
		inner.WriteString(")")
		for _, ec := range f.ErrorCodes {
			args = append(args, ec)
		}
	}
	if len(f.AccountIDs) > 0 {
		inner.WriteString(" AND rl.account_id IN (")
		inner.WriteString(placeholders(len(f.AccountIDs)))
		inner.WriteString(")")
		for _, id := range f.AccountIDs {
			args = append(args, id)
		}
	}
	if len(f.ActorIDs) > 0 {
		inner.WriteString(" AND COALESCE(au.id, rl.actor_id) IN (")
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
		inner.WriteString(" AND rl.normalized_route IN (")
		inner.WriteString(placeholders(len(f.NormalizedRoutes)))
		inner.WriteString(")")
		for _, r := range f.NormalizedRoutes {
			args = append(args, r)
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

	if mode != queryModeActor {
		return inner.String(), args
	}

	// Actor mode: wrap the inner block as a derived table so MySQL/Vitess
	// picks up the (target_account_id, occurred_at DESC, id DESC) index, runs
	// the LIMIT, *then* nested-loop joins user + api_key on only the matching
	// rows.
	var outer strings.Builder
	outer.WriteString(
		"SELECT rl.id, rl.method, rl.host, rl.path, rl.normalized_route, " +
			"rl.query_json, rl.status_code, rl.latency_us, rl.api_version, rl.actor_id, " +
			"rl.actor_type, rl.identity_type, rl.client_ip_string, rl.user_agent, " +
			"rl.referrer, rl.error_code, rl.error_message, rl.occurred_at, rl.created_at, " +
			"rl.idempotency_key_id, rl.request_body_json, rl.response_body_json, rl.target_account_id, " +
			"u.email AS user_email, u.name AS user_name, " +
			"ak.type_id AS api_key_type_id, ak.redacted_value AS api_key_redacted_value, ak.name AS api_key_name" +
			" FROM (",
	)
	outer.WriteString(inner.String())
	outer.WriteString(
		") rl" +
			" LEFT JOIN account_user au ON au.id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'" +
			" LEFT JOIN `user` u ON au.user_id = u.id AND rl.identity_type = 'user'" +
			" LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'",
	)
	if dir == pagination.DirectionBackward {
		outer.WriteString(" ORDER BY rl.occurred_at ASC, rl.id ASC")
	} else {
		outer.WriteString(" ORDER BY rl.occurred_at DESC, rl.id DESC")
	}
	return outer.String(), args
}

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
		read.Actor = buildMinimalActor(r.ActorID, r.IdentityType)
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
			&r.AccountName, &r.IdempotencyKey,
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
