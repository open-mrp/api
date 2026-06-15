package repository

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/pagination"
)

// emptyFilter represents the no-filter case — only target_account_id, cursor,
// ORDER BY, and LIMIT should appear in the generated SQL.
func emptyFilter() *domain.ListRequestLogsFilter {
	return &domain.ListRequestLogsFilter{}
}

func TestBuildListQuery_NoFiltersEmitsOnlyBaselinePredicates(t *testing.T) {
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	mustContain(t, sql, "WHERE (rl.account_id = ? OR rl.target_account_id = ?)")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")

	forbidden := []string{
		"rl.identity_type IN",
		"rl.method IN",
		"rl.status_code IN",
		"rl.error_code IN",
		"rl.account_id IN",
		"rl.target_account_id IN",
		"rl.actor_id IN",
		"rl.normalized_route IN",
		"rl.host IN",
		"rl.latency_us >=",
		"rl.public_endpoint =",
		"rl.occurred_at >=",
		"rl.occurred_at <=",
	}
	for _, f := range forbidden {
		if strings.Contains(sql, f) {
			t.Errorf("no-filter query unexpectedly contains predicate %q; SQL:\n%s", f, sql)
		}
	}
}

// actor_id now stores the public id (account_user.id for user actors), so the
// SELECT reads it directly — no COALESCE / account_user translation. The scan
// target is a sql.NullString, so a NULL actor_id (e.g. unauthenticated request)
// is handled by the scanner rather than a SQL-side default.
func TestBuildListQuery_ActorIDSelectedDirectly(t *testing.T) {
	for _, mode := range []queryMode{queryModeBase, queryModeActor, queryModeFull} {
		sql, _ := buildListQuery(mode, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)
		mustContain(t, sql, "rl.actor_id AS actor_id")
		if strings.Contains(sql, "COALESCE(au.id") {
			t.Errorf("query should no longer translate actor_id via account_user; SQL:\n%s", sql)
		}
	}
}

func TestBuildListQuery_ActorTypesEmitsInPredicate(t *testing.T) {
	f := emptyFilter()
	f.ActorTypes = []string{"user", "api_key"}

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "rl.identity_type IN (?, ?)")
	if !containsArg(args, "user") || !containsArg(args, "api_key") {
		t.Errorf("expected 'user' and 'api_key' in args; got %#v", args)
	}
}

func TestBuildListQuery_ForwardCursorEmitsLessThanComparison(t *testing.T) {
	cur := &pagination.StringCursor{
		OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
		ID:         "rlog_cursor",
		Direction:  pagination.DirectionForward,
	}
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, cur, 101)

	mustContain(t, sql, "rl.occurred_at < ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")
	if strings.Contains(sql, "rl.occurred_at > ?") {
		t.Errorf("forward cursor produced backward comparison; SQL:\n%s", sql)
	}
}

func TestBuildListQuery_BackwardCursorEmitsGreaterThanComparison(t *testing.T) {
	cur := &pagination.StringCursor{
		OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
		ID:         "rlog_cursor",
		Direction:  pagination.DirectionBackward,
	}
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionBackward, "acc_1", emptyFilter(), false, false, false, cur, 101)

	mustContain(t, sql, "rl.occurred_at > ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at ASC, rl.id ASC LIMIT ?")
	if strings.Contains(sql, "rl.occurred_at < ?") {
		t.Errorf("backward cursor produced forward comparison; SQL:\n%s", sql)
	}
}

func TestBuildListQuery_ActorModeWrapsInDerivedTableWithUserAndApiKeyJoins(t *testing.T) {
	sql, _ := buildListQuery(queryModeActor, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	mustContain(t, sql, " FROM (")
	mustContain(t, sql, ") rl")
	mustContain(t, sql, "LEFT JOIN `user` u ON u.id = rl.actor_id AND rl.identity_type = 'user'")
	mustContain(t, sql, "LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'")

	forbidden := []string{
		"LEFT JOIN account_user au",
		"LEFT JOIN role r_user",
		"LEFT JOIN account a",
	}
	for _, f := range forbidden {
		if strings.Contains(sql, f) {
			t.Errorf("actor mode unexpectedly emitted %q; SQL:\n%s", f, sql)
		}
	}
}

func TestBuildListQuery_IdempotencyKeyUsesExistsOnLinkedRow(t *testing.T) {
	f := emptyFilter()
	key := "550e8400-e29b-41d4-a716-446655440000"
	f.IdempotencyKey = &key

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "EXISTS (SELECT 1 FROM idempotency_key ik2 WHERE ik2.type_id = rl.idempotency_key_id AND ik2.idempotency_key = ?)")
	if !containsArg(args, key) {
		t.Errorf("expected idempotency key in args; got %#v", args)
	}
}

// Full mode must wrap in a derived table just like actor mode so the
// WHERE + ORDER BY + LIMIT run against request_log (using the
// target_account_id/occurred_at/id index) before the enrichment joins. The
// flat join-everything-then-LIMIT shape filesorts the whole account partition
// and times out on a large request_log table.
func TestBuildListQuery_FullModeWrapsInDerivedTableWithAllJoins(t *testing.T) {
	sql, _ := buildListQuery(queryModeFull, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	mustContain(t, sql, " FROM (")
	mustContain(t, sql, ") rl")

	// The LIMIT must live inside the derived table, ahead of the enrichment
	// joins, so it is applied before any nested-loop join.
	derivedTable := sql[strings.Index(sql, "FROM (")+len("FROM (") : strings.Index(sql, ") rl")]
	mustContain(t, derivedTable, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")
	for _, joined := range []string{"LEFT JOIN role r_user", "LEFT JOIN account a", "LEFT JOIN idempotency_key ik"} {
		if strings.Contains(derivedTable, joined) {
			t.Errorf("enrichment join %q leaked into the derived table; SQL:\n%s", joined, sql)
		}
	}

	required := []string{
		"LEFT JOIN `user` u ON u.id = rl.actor_id AND rl.identity_type = 'user'",
		"LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'",
		"LEFT JOIN account_user au ON au.user_id = rl.actor_id AND au.account_id = rl.target_account_id AND rl.identity_type = 'user'",
		"LEFT JOIN role r_user ON au.role_id = r_user.id",
		"LEFT JOIN role r_key ON ak.role_id = r_key.id",
		"LEFT JOIN account a ON rl.target_account_id = a.id",
		"a.created_at AS account_created_at",
		"a.updated_at AS account_updated_at",
		"au.role_id AS user_role_id",
		"LEFT JOIN idempotency_key ik ON rl.idempotency_key_id = ik.type_id",
	}
	for _, r := range required {
		mustContain(t, sql, r)
	}
}

func TestBuildListQuery_SliceFiltersExpandToMatchingPlaceholderCounts(t *testing.T) {
	f := emptyFilter()
	f.Methods = []string{"GET", "POST"}
	f.StatusCodes = []int32{200, 404, 500}
	f.ErrorCodes = []string{"not_found"}
	f.ActorAccountIDs = []string{"acct_a", "acct_b"}
	f.TargetAccountIDs = []string{"acct_c"}
	f.ActorIDs = []string{"actu_1", "actu_2", "actu_3"}
	f.ActorTypes = []string{"user"}
	f.NormalizedRoutes = []string{"/a", "/b"}
	f.Hosts = []string{"api.example.com"}

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "rl.method IN (?, ?)")
	mustContain(t, sql, "rl.status_code IN (?, ?, ?)")
	mustContain(t, sql, "rl.error_code IN (?)")
	mustContain(t, sql, "rl.account_id IN (?, ?)")
	mustContain(t, sql, "rl.target_account_id IN (?)")
	mustContain(t, sql, "rl.actor_id IN (?, ?, ?)")
	mustContain(t, sql, "rl.identity_type IN (?)")
	// normalized_route is compared on route shape (param tokens collapsed to
	// `{}`) via REGEXP_REPLACE, not on the bare column. See normalizeRouteParams.
	mustContain(t, sql, normalizedRouteColumnExpr+" IN (?, ?)")
	mustContain(t, sql, "rl.host IN (?)")

	// 3 JSON-include booleans (query/request/response) + 2 caller-account scope
	// binds (account_id OR target_account_id) + 2 methods + 3 status codes +
	// 1 error code + 2 actor account ids + 1 target account id + 3 actor ids +
	// 1 actor type + 2 routes + 1 host + limit = 22
	want := 3 + 2 + 2 + 3 + 1 + 2 + 1 + 3 + 1 + 2 + 1 + 1
	if len(args) != want {
		t.Errorf("unexpected arg count: got %d, want %d; args=%#v", len(args), want, args)
	}
}

// Helpers

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected SQL to contain %q, got:\n%s", needle, haystack)
	}
}

func containsArg(args []any, target any) bool {
	return slices.Contains(args, target)
}
