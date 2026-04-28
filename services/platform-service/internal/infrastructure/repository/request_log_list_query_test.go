package repository

import (
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

	mustContain(t, sql, "WHERE rl.target_account_id = ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")

	forbidden := []string{
		"rl.identity_type IN",
		"rl.method IN",
		"rl.status_code IN",
		"rl.error_code IN",
		"rl.account_id IN",
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
	mustContain(t, sql, "LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'")
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

func TestBuildListQuery_FullModeInlinesAllJoinsWithoutDerivedTable(t *testing.T) {
	sql, _ := buildListQuery(queryModeFull, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	if strings.Contains(sql, " FROM (") {
		t.Errorf("full mode should not wrap in derived table; SQL:\n%s", sql)
	}

	required := []string{
		"LEFT JOIN `user` u",
		"LEFT JOIN api_key ak",
		"LEFT JOIN account_user au",
		"LEFT JOIN role r_user",
		"LEFT JOIN role r_key",
		"LEFT JOIN account a",
		"LEFT JOIN idempotency_key ik",
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
	f.AccountIDs = []string{"acct_a", "acct_b"}
	f.ActorIDs = []string{"u_1", "u_2", "u_3"}
	f.ActorTypes = []string{"user"}
	f.NormalizedRoutes = []string{"/a", "/b"}
	f.Hosts = []string{"api.example.com"}

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "rl.method IN (?, ?)")
	mustContain(t, sql, "rl.status_code IN (?, ?, ?)")
	mustContain(t, sql, "rl.error_code IN (?)")
	mustContain(t, sql, "rl.account_id IN (?, ?)")
	mustContain(t, sql, "rl.actor_id IN (?, ?, ?)")
	mustContain(t, sql, "rl.identity_type IN (?)")
	mustContain(t, sql, "rl.normalized_route IN (?, ?)")
	mustContain(t, sql, "rl.host IN (?)")

	// 3 JSON-include booleans (query/request/response) + target_account_id +
	// 2 methods + 3 status codes + 1 error code + 2 account ids +
	// 3 actor ids + 1 actor type + 2 routes + 1 host + limit = 20
	want := 3 + 1 + 2 + 3 + 1 + 2 + 3 + 1 + 2 + 1 + 1
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
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
