package repository

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
)

// emptyFilter represents the no-filter case — only the scope branches, cursor,
// ORDER BY, and LIMIT should appear in the generated SQL.
func emptyFilter() *domain.ListRequestLogsFilter {
	return &domain.ListRequestLogsFilter{}
}

func TestBuildListQuery_NoFiltersEmitsOnlyBaselinePredicates(t *testing.T) {
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	// The dual-scope security filter is a UNION of two single-column keyset
	// branches, not a single `account_id = ? OR target_account_id = ?`. The OR
	// form forces an index_merge + filesort that times out on page 2 of a large
	// request_log; see buildListQuery.
	mustContain(t, sql, "WHERE rl.account_id = ?")
	mustContain(t, sql, "WHERE rl.target_account_id = ?")
	mustContain(t, sql, ") UNION (")
	if strings.Contains(sql, "rl.account_id = ? OR rl.target_account_id = ?") {
		t.Errorf("scope must be a UNION of single-column branches, not an OR; SQL:\n%s", sql)
	}
	// Hidden logs (e.g. HideFromRequestLog endpoints) are always excluded from listings.
	mustContain(t, sql, "rl.hidden = FALSE")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")
	// The merged id page is re-sorted and re-limited before the join back.
	mustContain(t, sql, "ORDER BY ks.occurred_at DESC, ks.id DESC LIMIT ?")

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

// ExcludeErrorCodes must emit a NOT IN guarded by an IS NULL disjunct. error_code
// is NULL for successful requests, and a bare `NULL NOT IN (...)` evaluates to NULL
// (falsy), which would silently drop every 2xx row — so the IS NULL branch is
// load-bearing, not cosmetic.
func TestBuildListQuery_ExcludeErrorCodesGuardsNullRows(t *testing.T) {
	f := emptyFilter()
	f.ExcludeErrorCodes = []string{"expired_token"}

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "(rl.error_code IS NULL OR rl.error_code NOT IN (?))")
	if !containsArg(args, "expired_token") {
		t.Errorf("expected 'expired_token' in args; got %#v", args)
	}
	// Guard against a regression to a bare NOT IN that would drop successful rows.
	if strings.Contains(sql, "rl.error_code NOT IN") && !strings.Contains(sql, "rl.error_code IS NULL OR") {
		t.Errorf("exclude predicate must keep NULL error_code rows; SQL:\n%s", sql)
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
	mustContain(t, sql, ") page")
	mustContain(t, sql, "JOIN request_log rl ON rl.id = page.id")
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
	mustContain(t, sql, ") page")
	mustContain(t, sql, "JOIN request_log rl ON rl.id = page.id")

	// The LIMIT must live inside the derived table, ahead of the enrichment
	// joins, so it is applied before any nested-loop join.
	derivedTable := sql[strings.Index(sql, "FROM (")+len("FROM (") : strings.Index(sql, ") page")]
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

	// The 3 JSON-include booleans (query/request/response) bind once, in the outer
	// SELECT list. The dual-scope filter is a UNION of two keyset branches, so every
	// per-branch bind appears twice. Per branch: 1 scope-account bind + 2 methods +
	// 3 status codes + 1 error code + 2 actor account ids + 1 target account id +
	// 3 actor ids + 1 actor type + 2 routes + 1 host + branch LIMIT = 18. Include
	// booleans (3) + two branches (36) + the merged id page's LIMIT (1) = 40.
	perBranch := 1 + 2 + 3 + 1 + 2 + 1 + 3 + 1 + 2 + 1 + 1
	want := 3 + perBranch*2 + 1
	if len(args) != want {
		t.Errorf("unexpected arg count: got %d, want %d; args=%#v", len(args), want, args)
	}
}

// Every ORDER BY in the generated SQL must sort only the (id, occurred_at)
// keyset pair. MySQL's filesort cannot pack JSON/TEXT addon columns, so a single
// oversized request/response body inside a sorted row exceeds sort_buffer_size
// and kills the query with error 1038 "Out of sort memory" — the production
// failure this shape exists to prevent. The JSON payload columns may appear only
// in the outermost SELECT, which joins back by primary key after all ordering.
func TestBuildListQuery_PayloadColumnsNeverEnterASortedSet(t *testing.T) {
	for _, mode := range []queryMode{queryModeBase, queryModeActor, queryModeFull} {
		sql, args := buildListQuery(mode, pagination.DirectionForward, "acc_1", emptyFilter(), true, true, true, nil, 101)

		pageEnd := strings.Index(sql, ") page")
		if pageEnd < 0 {
			t.Fatalf("expected an id-page derived table; SQL:\n%s", sql)
		}
		derived := sql[strings.Index(sql, "FROM (")+len("FROM (") : pageEnd]
		for _, payload := range []string{"query_json", "request_body_json", "response_body_json", "error_message", "user_agent"} {
			if strings.Contains(derived, payload) {
				t.Errorf("wide column %q leaked into the sorted id page; SQL:\n%s", payload, sql)
			}
		}

		// The outermost query (the only part that carries the JSON columns) must
		// not sort — keyset order is restored in Go by sortListResults.
		if outer := sql[pageEnd:]; strings.Contains(outer, "ORDER BY") {
			t.Errorf("outermost query must not ORDER BY payload-carrying rows; SQL:\n%s", sql)
		}

		// The JSON-include booleans bind to the outer SELECT list, which precedes
		// the derived table in text order — they must be the first three args.
		if len(args) < 3 || args[0] != true || args[1] != true || args[2] != true {
			t.Errorf("expected JSON-include booleans as leading args; got %#v", args)
		}
	}
}

func TestSortListResults_RestoresKeysetOrder(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0).UTC()
	mk := func(id string, at time.Time) *domain.RequestLogRead {
		return &domain.RequestLogRead{ID: id, OccurredAt: at}
	}
	scrambled := func() []*domain.RequestLogRead {
		return []*domain.RequestLogRead{
			mk("rlog_b", t0),
			mk("rlog_c", t0.Add(2*time.Second)),
			mk("rlog_a", t0),
			mk("rlog_d", t0.Add(time.Second)),
		}
	}

	forward := scrambled()
	sortListResults(forward, pagination.DirectionForward)
	wantForward := []string{"rlog_c", "rlog_d", "rlog_b", "rlog_a"}
	for i, want := range wantForward {
		if forward[i].ID != want {
			t.Fatalf("forward order[%d]: got %q, want %q", i, forward[i].ID, want)
		}
	}

	backward := scrambled()
	sortListResults(backward, pagination.DirectionBackward)
	for i, j := 0, len(wantForward)-1; i < len(backward); i, j = i+1, j-1 {
		if backward[i].ID != wantForward[j] {
			t.Fatalf("backward order[%d]: got %q, want %q", i, backward[i].ID, wantForward[j])
		}
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
