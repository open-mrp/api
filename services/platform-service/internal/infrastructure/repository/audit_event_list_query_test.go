package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func buildAuditList(t *testing.T, dir pagination.Direction, f *domain.ListAuditEventsFilter, cursor *pagination.StringCursor) (string, []any) {
	t.Helper()
	q, args := buildAuditEventListQuery(dir, "ac_caller", f, false, false, cursor, 26)
	if n := strings.Count(q, "?"); n != len(args) {
		t.Fatalf("%d placeholders, %d args; SQL:\n%s", n, len(args), q)
	}
	return q, args
}

// auditBranches splits the page into its keyset branches.
func auditBranches(t *testing.T, q string) []string {
	t.Helper()
	start := strings.Index(q, "FROM ((SELECT ")
	end := strings.Index(q, ") ks")
	if start < 0 || end < 0 {
		t.Fatalf("expected a UNION of keyset branches; SQL:\n%s", q)
	}
	body := q[start+len("FROM (") : end]
	return strings.Split(body, " UNION ")
}

func branchesScopedTo(branches []string, scope string) []string {
	var out []string
	for _, b := range branches {
		if strings.Contains(b, "WHERE ae."+scope+" = ?") {
			out = append(out, b)
		}
	}
	return out
}

func forcedIndexes(t *testing.T, branch string) []string {
	t.Helper()
	m := regexp.MustCompile(`FORCE INDEX \(([^)]*)\)`).FindStringSubmatch(branch)
	if m == nil {
		t.Fatalf("branch forces no index: %s", branch)
	}
	return strings.Split(m[1], ", ")
}

func TestBuildAuditEventListQuery_ScopeIsAUnionNotAnOr(t *testing.T) {
	q, args := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{}, nil)

	if strings.Contains(q, "account_id = ? OR ") {
		t.Errorf("scope must be one branch per account column, not an OR; SQL:\n%s", q)
	}
	branches := auditBranches(t, q)
	if len(branches) != 2 {
		t.Fatalf("expected an acting-account and a target-account branch, got %d; SQL:\n%s", len(branches), q)
	}
	if got := forcedIndexes(t, branchesScopedTo(branches, "account_id")[0]); len(got) != 1 || got[0] != auditEventAccountIndex {
		t.Errorf("unfiltered acting-account branch should walk %s, got %v", auditEventAccountIndex, got)
	}
	if got := forcedIndexes(t, branchesScopedTo(branches, "target_account_id")[0]); len(got) != 1 || got[0] != auditEventTargetAccountIndex {
		t.Errorf("unfiltered target branch should walk %s, got %v", auditEventTargetAccountIndex, got)
	}
	for _, b := range branches {
		mustContain(t, b, "ORDER BY ae.occurred_at DESC, ae.type_id DESC LIMIT ?")
	}
	mustContain(t, q, ") ks ORDER BY ks.occurred_at DESC, ks.type_id DESC LIMIT ?")
	if args[2] != "ac_caller" {
		t.Errorf("first branch should bind the caller's account after the include flags; args=%#v", args)
	}
}

func TestBuildAuditEventListQuery_EnrichmentAndJSONStayOutsideThePage(t *testing.T) {
	q, args := buildAuditEventListQuery(pagination.DirectionForward, "ac_caller", &domain.ListAuditEventsFilter{}, true, true, nil, 26)

	pageEnd := strings.Index(q, ") page")
	page := q[strings.Index(q, "FROM (SELECT ks.id"):pageEnd]
	for _, wide := range []string{"changes", "metadata", "LEFT JOIN"} {
		if strings.Contains(page, wide) {
			t.Errorf("%q leaked into the page; SQL:\n%s", wide, q)
		}
	}
	if strings.Contains(q[pageEnd:], "ORDER BY") {
		t.Errorf("the query carrying the JSON columns must not sort; SQL:\n%s", q)
	}
	mustContain(t, q[pageEnd:], "JOIN audit_event ae ON ae.id = page.id")
	if args[0] != true || args[1] != true {
		t.Errorf("the include flags bind first, to the outer SELECT list; args=%#v", args)
	}
}

func TestBuildAuditEventListQuery_SingleValuedFilterForcesItsComposite(t *testing.T) {
	q, args := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{
		ActorIDs: []string{"us_one"},
		Actions:  []string{"update"},
	}, nil)

	branches := auditBranches(t, q)
	account := branchesScopedTo(branches, "account_id")
	if len(account) != 1 {
		t.Fatalf("one actor needs no arms; SQL:\n%s", q)
	}
	got := forcedIndexes(t, account[0])
	want := []string{auditEventActorIDIndex, auditEventActionIndex}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("forced %v, want %v", got, want)
	}
	target := forcedIndexes(t, branchesScopedTo(branches, "target_account_id")[0])
	if strings.Join(target, ",") != auditEventTargetAccountIndex+","+auditEventActorIndex {
		t.Errorf("target branch forced %v", target)
	}
	mustContain(t, account[0], "ae.actor_id IN (?)")
	mustContain(t, account[0], "ae.action IN (?)")
	if !containsArg(args, "us_one") || !containsArg(args, "update") {
		t.Errorf("filter values missing from args: %#v", args)
	}
}

func TestBuildAuditEventListQuery_FewValuesOfTheLeadingFilterAreArms(t *testing.T) {
	q, _ := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{
		ActorIDs: []string{"us_b", "us_a", "us_b"},
		Actions:  []string{"create", "delete"},
	}, nil)

	account := branchesScopedTo(auditBranches(t, q), "account_id")
	if len(account) != 2 {
		t.Fatalf("expected one arm per distinct actor, got %d; SQL:\n%s", len(account), q)
	}
	for _, arm := range account {
		mustContain(t, arm, "ae.actor_id = ?")
		mustContain(t, arm, "ae.action IN (?, ?)")
		if got := forcedIndexes(t, arm); len(got) != 1 || got[0] != auditEventActorIDIndex {
			t.Errorf("arm forced %v; the residual action filter's key must be withheld", got)
		}
	}
	target := branchesScopedTo(auditBranches(t, q), "target_account_id")
	if len(target) != 1 {
		t.Fatalf("the target branch has no composite to split on; SQL:\n%s", q)
	}
	mustContain(t, target[0], "ae.actor_id IN (?, ?)")
}

func TestBuildAuditEventListQuery_LongListRangesItsKey(t *testing.T) {
	ids := []string{"r1", "r2", "r3", "r4", "r5", "r6", "r7", "r8", "r9"}
	q, _ := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{ResourceIDs: ids}, nil)

	account := branchesScopedTo(auditBranches(t, q), "account_id")
	if len(account) != 1 {
		t.Fatalf("a long list is one IN, not arms; SQL:\n%s", q)
	}
	if got := forcedIndexes(t, account[0]); len(got) != 1 || got[0] != auditEventResourceIDIndex {
		t.Errorf("forced %v", got)
	}
	mustContain(t, account[0], "ae.resource_id IN ("+placeholders(len(ids))+")")
}

func TestBuildAuditEventListQuery_RootAndAccountFiltersUseTheirKeys(t *testing.T) {
	q, _ := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{
		RootResourceType: "sales_order",
		RootResourceID:   "or_root",
		TargetAccountIDs: []string{"ac_other"},
		ActorAccountIDs:  []string{"ac_caller"},
	}, nil)

	branches := auditBranches(t, q)
	account := branchesScopedTo(branches, "account_id")[0]
	if got := strings.Join(forcedIndexes(t, account), ","); got != auditEventTargetAccountIndex+","+auditEventRootIndex {
		t.Errorf("acting-account branch forced %s", got)
	}
	target := branchesScopedTo(branches, "target_account_id")[0]
	if got := strings.Join(forcedIndexes(t, target), ","); got != auditEventTargetRootIndex {
		t.Errorf("target branch forced %s; the root key yields a root's events in order without walking the account's", got)
	}
	for _, b := range branches {
		mustContain(t, b, "ae.root_resource_type = ? AND ae.root_resource_id = ?")
		mustContain(t, b, "ae.target_account_id IN (?)")
		mustContain(t, b, "ae.account_id IN (?)")
	}
}

func TestBuildAuditEventListQuery_EveryFilterReachesBothDirections(t *testing.T) {
	start := time.Unix(1_700_000_000, 0).UTC()
	end := start.Add(time.Hour)
	query := "50%_off"
	f := &domain.ListAuditEventsFilter{
		ActorTypes: []string{"user"},
		StartDate:  &start,
		EndDate:    &end,
		Query:      &query,
	}
	cursor := &pagination.StringCursor{OccurredAt: start.Add(time.Minute), ID: "auev_cursor"}

	for _, dir := range []pagination.Direction{pagination.DirectionForward, pagination.DirectionBackward} {
		q, args := buildAuditList(t, dir, f, cursor)
		for _, b := range auditBranches(t, q) {
			mustContain(t, b, "ae.identity_type IN (?)")
			mustContain(t, b, "ae.occurred_at >= ?")
			mustContain(t, b, "ae.occurred_at <= ?")
			mustNotContain(t, b, "LIKE")
		}
		if !containsArg(args, "50%_off") {
			t.Errorf("search matches the term as given; args=%#v", args)
		}
	}
}

func TestBuildAuditEventListQuery_CursorDirection(t *testing.T) {
	cursor := &pagination.StringCursor{OccurredAt: time.Unix(1_700_000_000, 0).UTC(), ID: "auev_cursor"}

	fwd, _ := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{}, cursor)
	mustContain(t, fwd, "(ae.occurred_at < ? OR (ae.occurred_at = ? AND ae.type_id < ?))")
	if strings.Contains(fwd, "ASC") {
		t.Errorf("forward pages read newest first; SQL:\n%s", fwd)
	}

	bwd, _ := buildAuditList(t, pagination.DirectionBackward, &domain.ListAuditEventsFilter{}, cursor)
	mustContain(t, bwd, "(ae.occurred_at > ? OR (ae.occurred_at = ? AND ae.type_id > ?))")
	mustContain(t, bwd, "ORDER BY ae.occurred_at ASC, ae.type_id ASC LIMIT ?")
	mustContain(t, bwd, ") ks ORDER BY ks.occurred_at ASC, ks.type_id ASC LIMIT ?")
	if strings.Contains(bwd, "DESC") {
		t.Errorf("backward pages read oldest first; SQL:\n%s", bwd)
	}
}

func TestSortAuditEventPage_RestoresKeysetOrder(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0).UTC()
	mk := func(id string, at time.Time) *domain.AuditEventRead {
		return &domain.AuditEventRead{AuditEvent: domain.AuditEvent{ID: id, OccurredAt: at}}
	}
	scrambled := func() []*domain.AuditEventRead {
		return []*domain.AuditEventRead{mk("auev_b", t0), mk("auev_c", t0.Add(2*time.Second)), mk("auev_a", t0), mk("auev_d", t0.Add(time.Second))}
	}
	want := []string{"auev_c", "auev_d", "auev_b", "auev_a"}

	forward := scrambled()
	sortAuditEventPage(forward, pagination.DirectionForward)
	for i, id := range want {
		if forward[i].ID != id {
			t.Fatalf("forward[%d] = %q, want %q", i, forward[i].ID, id)
		}
	}
	backward := scrambled()
	sortAuditEventPage(backward, pagination.DirectionBackward)
	for i := range want {
		if backward[i].ID != want[len(want)-1-i] {
			t.Fatalf("backward[%d] = %q, want %q", i, backward[i].ID, want[len(want)-1-i])
		}
	}
}

func auditListRow(id string, at time.Time) []driver.Value {
	return []driver.Value{
		id, "us_actor", "internal", "user", "ac_caller", "update", "item", "it_1", "", "",
		"core-service", nil, nil, nil, at, at, "ac_caller",
		"Caller", at, at, "A User", "user@example.com", nil, nil, nil,
	}
}

func TestAuditEventRepoList_SortsThePageAndBuildsItsCursor(t *testing.T) {
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	t0 := time.Unix(1_700_000_000, 0).UTC()
	rows := sqlmock.NewRows(auditEventListColumnNames()).
		AddRow(auditListRow("auev_old", t0)...).
		AddRow(auditListRow("auev_new", t0.Add(2*time.Second))...).
		AddRow(auditListRow("auev_mid", t0.Add(time.Second))...)
	mock.ExpectQuery(regexp.QuoteMeta("FROM (SELECT ks.id FROM ((SELECT")).WillReturnRows(rows)

	repo := NewAuditEventRepo(sqlc.New(conn))
	got, apiErr := repo.List(context.Background(), "ac_caller", &domain.ListAuditEventsFilter{Limit: 2}, nil)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if len(got.AuditEvents) != 2 || got.AuditEvents[0].ID != "auev_new" || got.AuditEvents[1].ID != "auev_mid" {
		t.Fatalf("expected the two newest, newest first; got %+v", got.AuditEvents)
	}
	if !got.PageInfo.HasNextPage || got.PageInfo.NextCursor == nil {
		t.Fatalf("a third row means another page: %+v", got.PageInfo)
	}
	next, err := pagination.DecodeStringCursor(*got.PageInfo.NextCursor)
	if err != nil || next.ID != "auev_mid" {
		t.Fatalf("next cursor should follow the last row shown: %+v, %v", next, err)
	}
	if got.AuditEvents[0].Actor == nil || got.AuditEvents[0].Actor.Name == nil || *got.AuditEvents[0].Actor.Name != "A User" {
		t.Errorf("actor should be enriched from the joined user: %+v", got.AuditEvents[0].Actor)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func auditEventListColumnNames() []string {
	names := make([]string, len(auditEventListColumns))
	for i, c := range auditEventListColumns {
		if _, alias, ok := strings.Cut(c, " AS "); ok {
			names[i] = alias
			continue
		}
		_, names[i], _ = strings.Cut(c, ".")
	}
	return names
}

// A search pins each column it matches in turn, reading that column's key in both scopes, so a term that
// matches nothing reads nothing instead of walking every event the account has.
func TestBuildAuditEventListQuery_SearchReadsEachColumnsKeyInBothScopes(t *testing.T) {
	query := "  rq_abc123 "
	q, args := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{Query: &query}, nil)

	want := map[string]string{
		"WHERE ae.account_id = ? AND ae.resource_id = ?":          auditEventResourceIDIndex,
		"WHERE ae.account_id = ? AND ae.request_id = ?":           auditEventRequestIDIndex,
		"WHERE ae.account_id = ? AND ae.resource_type = ?":        auditEventResourceTypeIndex,
		"WHERE ae.account_id = ? AND ae.action = ?":               auditEventActionIndex,
		"WHERE ae.target_account_id = ? AND ae.resource_id = ?":   auditEventTargetResourceIDIndex,
		"WHERE ae.target_account_id = ? AND ae.request_id = ?":    auditEventTargetRequestIDIndex,
		"WHERE ae.target_account_id = ? AND ae.resource_type = ?": auditEventTargetResourceTypeIndex,
		"WHERE ae.target_account_id = ? AND ae.action = ?":        auditEventTargetActionIndex,
	}
	branches := auditBranches(t, q)
	if len(branches) != len(want) {
		t.Fatalf("%d branches, want one per column and scope:\n%s", len(branches), strings.Join(branches, "\n"))
	}
	for _, b := range branches {
		matched := false
		for pinned, index := range want {
			if strings.Contains(b, pinned) {
				matched = true
				if got := strings.Join(forcedIndexes(t, b), ","); got != index {
					t.Errorf("branch pinning %q forced %s, want %s", pinned, got, index)
				}
			}
		}
		if !matched {
			t.Errorf("branch pins no searched column: %s", b)
		}
		mustNotContain(t, b, "LIKE")
	}
	if !containsArg(args, "rq_abc123") {
		t.Errorf("ids are matched as given, trimmed; args=%#v", args)
	}
}

func TestAuditEventSearchCode_MatchesHowTypesAndActionsAreStored(t *testing.T) {
	for in, want := range map[string]string{
		"Sales Order":      "sales_order",
		"sales_order":      "sales_order",
		" sales-order ":    "sales_order",
		"UPDATE":           "update",
		"Production  Step": "production_step",
		"rq_01abcDEF":      "rq_01abcdef",
	} {
		if got := auditEventSearchCode(in); got != want {
			t.Errorf("auditEventSearchCode(%q) = %q, want %q", in, got, want)
		}
	}
}

// One resource id is read through the target key that pins it, not by walking every event the account was
// acted upon in; several ids offer that key beside the scope key.
func TestBuildAuditEventListQuery_TargetBranchReadsAResourceIDThroughItsKey(t *testing.T) {
	q, _ := buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{ResourceIDs: []string{"or_1"}}, nil)
	target := branchesScopedTo(auditBranches(t, q), "target_account_id")[0]
	if got := strings.Join(forcedIndexes(t, target), ","); got != auditEventTargetResourceIDIndex {
		t.Errorf("target branch forced %s", got)
	}

	q, _ = buildAuditList(t, pagination.DirectionForward, &domain.ListAuditEventsFilter{ResourceIDs: []string{"or_1", "or_2"}}, nil)
	target = branchesScopedTo(auditBranches(t, q), "target_account_id")[0]
	if got := strings.Join(forcedIndexes(t, target), ","); got != auditEventTargetAccountIndex+","+auditEventTargetResourceIDIndex {
		t.Errorf("target branch forced %s", got)
	}
}

func mustNotContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected SQL not to contain %q, got:\n%s", needle, haystack)
	}
}
