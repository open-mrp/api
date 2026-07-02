package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRootForTest resolves the monorepo root from the apidocs package dir (tools/apidocs).
func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// parsePermissionConstants reads permissions.go and returns suffix→value maps for domains and actions (e.g. "Customers"→"customers", "Read"→"read").
func parsePermissionConstants(t *testing.T, root string) (domains, actions map[string]string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "services/auth-service/pkg/types/permissions.go"))
	if err != nil {
		t.Fatal(err)
	}
	domains, actions = map[string]string{}, map[string]string{}
	domRe := regexp.MustCompile(`PermissionDomain([A-Za-z]+)\s+PermissionDomain\s*=\s*"([^"]+)"`)
	for _, m := range domRe.FindAllStringSubmatch(string(b), -1) {
		domains[m[1]] = m[2]
	}
	actRe := regexp.MustCompile(`Action([A-Za-z]+)\s+Action\s*=\s*"([^"]+)"`)
	for _, m := range actRe.FindAllStringSubmatch(string(b), -1) {
		actions[m[1]] = m[2]
	}
	return domains, actions
}

type coreCheck struct {
	perms map[string]bool // "<domain>:<action>" the function checks with literal args
	admin bool            // function calls CheckIsAdmin
}

// parseServiceChecks scans the given service trees and maps each function name to
// the permissions it checks and whether it requires admin. It captures both direct
// literal CheckHasPermission/CheckIsAdmin calls AND calls to relation-permission
// helpers (check<Entity>Read/WritePermission): a function that delegates its
// authorization to such a helper is credited with every permission that helper
// checks, so relation-variable endpoints are VERIFIED against their declared OR-set
// rather than excused. Helper bodies contain one CheckHasPermission per owner branch
// (resource / customers / suppliers), so following them yields the full OR-set.
func parseServiceChecks(t *testing.T, root string, domains, actions map[string]string, dirs ...string) map[string]coreCheck {
	t.Helper()
	out := map[string]coreCheck{}
	// calls[fn] = set of permission-helper functions fn invokes.
	calls := map[string]map[string]bool{}
	funcRe := regexp.MustCompile(`^func (?:\([^)]*\) )?([A-Za-z0-9_]+)\(`)
	// Allow an optional package qualifier before PermissionDomain/Action so aliased
	// imports (e.g. authtypes.PermissionDomainAuditEvents in the audit-events service)
	// are matched and verified rather than appearing as unaccounted.
	permRe := regexp.MustCompile(`CheckHasPermission\(\s*(?:[A-Za-z0-9_]+\.)?PermissionDomain([A-Za-z]+),\s*(?:[A-Za-z0-9_]+\.)?Action([A-Za-z]+)`)
	// A call to a relation-permission helper, e.g. checkMaterialReadPermission(...),
	// s.checkSalesOrderReadPermission(...), checkAccountUserWritePermission(...).
	helperCallRe := regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(check[A-Za-z0-9_]*Permission)\(`)
	for _, dir := range dirs {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, _ := os.ReadFile(path)
			cur := ""
			for _, line := range strings.Split(string(b), "\n") {
				if m := funcRe.FindStringSubmatch(line); m != nil {
					cur = m[1]
					continue
				}
				if cur == "" {
					continue
				}
				if m := permRe.FindStringSubmatch(line); m != nil {
					dom, dok := domains[m[1]]
					act, aok := actions[m[2]]
					if dok && aok {
						c := out[cur]
						if c.perms == nil {
							c.perms = map[string]bool{}
						}
						c.perms[dom+":"+act] = true
						out[cur] = c
					}
				}
				if strings.Contains(line, "CheckIsAdmin(") || strings.Contains(line, "CheckAPIKeyAccess(") {
					c := out[cur]
					c.admin = true
					out[cur] = c
				}
				if m := helperCallRe.FindStringSubmatch(line); m != nil && m[1] != cur {
					if calls[cur] == nil {
						calls[cur] = map[string]bool{}
					}
					calls[cur][m[1]] = true
				}
			}
			return nil
		})
	}
	// Fold each helper's permissions/admin into the functions that call it. Iterate
	// to a fixpoint so a function calling a helper that itself calls a helper is
	// still credited (bounded by the number of functions).
	for i := 0; i < len(calls)+1; i++ {
		changed := false
		for fn, helpers := range calls {
			c := out[fn]
			for helper := range helpers {
				hc, ok := out[helper]
				if !ok {
					continue
				}
				if hc.admin && !c.admin {
					c.admin = true
					changed = true
				}
				for p := range hc.perms {
					if c.perms == nil {
						c.perms = map[string]bool{}
					}
					if !c.perms[p] {
						c.perms[p] = true
						changed = true
					}
				}
			}
			out[fn] = c
		}
		if !changed {
			break
		}
	}
	return out
}

type endpointDecl struct {
	slug      string
	folder    string
	handler   string
	perms     map[string]bool
	roleAdmin bool
}

// parseEndpointDecls scans the api-gateway endpoints for AgentTool endpoints and returns each one's folder, handler method, declared permissions, and role-type.
func parseEndpointDecls(t *testing.T, root string, domains, actions map[string]string) []endpointDecl {
	t.Helper()
	handlerRe := regexp.MustCompile(`svc\.\([A-Za-z]+\)\.([A-Za-z0-9_]+)`)
	permRe := regexp.MustCompile(`\{(?:Domain: )?types\.PermissionDomain([A-Za-z]+), (?:Action: )?types\.Action([A-Za-z]+)\}`)
	var out []endpointDecl
	_ = filepath.WalkDir(filepath.Join(root, "services/api-gateway/endpoints"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, _ := os.ReadFile(path)
		src := string(b)
		if !strings.Contains(src, "AgentTool:") || !regexp.MustCompile(`AgentTool:\s*true`).MatchString(src) {
			return nil
		}
		e := endpointDecl{
			slug:      strings.TrimPrefix(strings.TrimSuffix(filepath.Base(path), ".go"), "endpoint_"),
			folder:    filepath.Base(filepath.Dir(path)),
			perms:     map[string]bool{},
			roleAdmin: regexp.MustCompile(`RequiredRoleType:\s*constants\.RoleTypeAdmin`).MatchString(src),
		}
		if m := handlerRe.FindStringSubmatch(src); m != nil {
			e.handler = m[1]
		}
		for _, m := range permRe.FindAllStringSubmatch(src, -1) {
			if dom, ok := domains[m[1]]; ok {
				if act, ok := actions[m[2]]; ok {
					e.perms[dom+":"+act] = true
				}
			}
		}
		out = append(out, e)
		return nil
	})
	return out
}

// parseGatewayHandlerTargets maps, per endpoint folder, each gateway service-impl method name to the downstream client methods it invokes (coreClient.X / authClient.X / billingClient.X / platformClient.X / agentClient.X). This lets the drift guard follow a gateway handler whose name differs from the core function that actually performs the permission check (e.g. gateway RetrieveCustomer -> coreClient.GetCustomer -> core GetCustomer's checks). Keyed by folder so identically-named methods in different folders never collide.
func parseGatewayHandlerTargets(t *testing.T, root string) map[string]map[string][]string {
	t.Helper()
	methodRe := regexp.MustCompile(`^func \([a-z_]+ \*?[A-Za-z0-9_]+\) ([A-Z][A-Za-z0-9_]+)\(`)
	funcRe := regexp.MustCompile(`^func `)
	clientCallRe := regexp.MustCompile(`[Cc]lient\.([A-Z][A-Za-z0-9_]+)\(`)
	out := map[string]map[string][]string{}
	_ = filepath.WalkDir(filepath.Join(root, "services/api-gateway/endpoints"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		folder := filepath.Base(filepath.Dir(path))
		b, _ := os.ReadFile(path)
		cur := ""
		for _, line := range strings.Split(string(b), "\n") {
			if m := methodRe.FindStringSubmatch(line); m != nil {
				cur = m[1]
				continue
			}
			if funcRe.MatchString(line) { // a non-method func (or any new top-level func) ends the current method scope
				cur = ""
				continue
			}
			if cur == "" {
				continue
			}
			if m := clientCallRe.FindStringSubmatch(line); m != nil {
				if out[folder] == nil {
					out[folder] = map[string][]string{}
				}
				out[folder][cur] = append(out[folder][cur], m[1])
			}
		}
		return nil
	})
	return out
}

// unverifiableEndpoints is the residual set of AgentTool endpoints that the static
// guard cannot prove by joining handler -> literal/relation-helper service checks.
// The guard now FOLLOWS relation-permission helpers (check<Entity>Read/WritePermission),
// so the large relation-variable family (materials, parts, addresses, products,
// carriers, sales-orders, account-users create, etc.) is VERIFIED here, not excused.
// What remains is only endpoints with NO joinable domain permission, each for a
// documented, deliberate reason — these are reported to maintainers, not a way to
// skip a check that should exist:
//
//   - public-by-design: pre-auth utilities that call third-party services with no
//     tenant identity (address validation / autocomplete). Correct declaration: none.
//   - auth-only reference data: gated by CheckIsAssignedActor / CheckIsAuthenticated
//     with NO resource-specific permission domain in permissions.go. Correct: none.
//   - gateway-static enum: the gateway returns a hardcoded reference list with no
//     downstream service call. (Flagged: these are effectively unauthenticated; see
//     the authz report — candidates for a lightweight auth check.)
//   - external-exempt global lookup: the helper requires the domain for internal
//     actors but EXEMPTS external (customer/supplier) actors, who carry no role
//     permissions; declaring the domain would false-reject those readers at the gate.
//   - gateway private-helper indirection: the gateway method delegates to the core
//     client through a private helper the static join can't trace; enforcement is a
//     relation helper and the endpoint declares the matching OR-set (verified by hand).
//   - gateway-gated participant auth: messaging/chat endpoints declare messaging
//     permissions at the gateway; notification-service enforces conversation membership
//     via caller/requireParticipant/resolveParticipant with no literal CheckHasPermission.
//   - gateway-gated recipient auth: bell-notification endpoints declare messaging
//     permissions at the gateway; notification-service scopes rows to the caller's
//     account_user via recipient()/actor() with no literal CheckHasPermission.
//   - gateway loader indirection: the gateway handler loads the resource via a
//     resourceloader batch-get instead of the name-matched downstream RPC.
var unverifiableEndpoints = map[string]string{
	"validate":                 "public-by-design: ValidateAddress calls Google Address Validation with no tenant identity (pre-auth utility)",
	"list_address_suggestions": "public-by-design: AutocompleteAddress calls Google Places with no tenant identity (pre-auth utility)",

	"list_account_statuses":     "auth-only reference data: CheckIsAssignedActor, no account-status permission domain exists",
	"retrieve_account_status":   "auth-only reference data: CheckIsAssignedActor, no account-status permission domain exists",
	"list_sales_order_statuses": "auth-only reference data: CheckIsAuthenticated, no sales-order-status permission domain exists",

	"list_transaction_methods": "gateway-static enum: handler returns a hardcoded list; now gated by CheckIsAuthenticated at the gateway (auth-only, no domain permission)",
	"list_transaction_types":   "gateway-static enum: handler returns a hardcoded list; now gated by CheckIsAuthenticated at the gateway (auth-only, no domain permission)",

	"list_adjustment_types": "external-exempt global lookup: checkAdjustmentTypeReadPermission requires adjustment_types:read for internal actors but exempts external actors; declaring it would false-reject those readers at the gate",

	"search": "gateway per-type dynamic gate: Search loops its providers calling identity.CheckHasPermission(domain, read) per resource type and only includes types the caller can read; endpoint declares the {sales_orders,purchase_orders,invoices,customers,items,shipments,messaging,agents}:read OR-set; no downstream name-matched handler to verify against",

	"activate_account_user": "gateway private-helper indirection: ActivateAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserWritePermission; endpoint declares the {team,customers,suppliers} OR-set",
	"disable_account_user":  "gateway private-helper indirection: DisableAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserWritePermission; endpoint declares the {team,customers,suppliers} OR-set",
	"remove_account_user":   "gateway private-helper indirection: RemoveAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserWritePermission; endpoint declares the {team,customers,suppliers} OR-set",

	"create_conversation":          "gateway-gated participant auth: CreateConversation uses caller() + membership; gateway declares messaging:create",
	"list_conversations":           "gateway-gated participant auth: ListConversations uses caller(); gateway declares messaging:read",
	"mark_conversation_read":       "gateway-gated participant auth: MarkConversationRead uses resolveParticipant(); gateway declares messaging:update",
	"retrieve_conversation":        "gateway-gated participant auth: GetConversation uses resolveParticipant(); gateway declares messaging:read",
	"create_attachment_upload_url": "gateway-gated participant auth: CreateAttachmentUploadURL requires active participant; gateway declares messaging:create",
	"list_messages":                "gateway-gated participant auth: ListMessages uses resolveParticipant(); gateway declares messaging:read",
	"send_message":                 "gateway-gated participant auth: SendMessage uses resolveParticipant(); gateway declares messaging:create",
	"list_contacts":                "gateway-gated participant auth: ListContacts is auth/account-scoped only; gateway declares messaging:read",
	"cancel_scheduled":             "gateway-gated participant auth: CancelScheduledMessage uses requireParticipant(); gateway declares messaging:update",
	"list_scheduled":               "gateway-gated participant auth: ListScheduledMessages uses requireParticipant(); gateway declares messaging:read",

	"set_workflow_status": "gateway-gated admin auth: UpdateWorkflowStatus uses requireMessagingAdmin(update); gateway declares messaging:update",
	"assign":              "gateway-gated admin auth: AssignConversation uses requireMessagingAdmin(update); gateway declares messaging:update",
	"report":              "gateway-gated participant auth: ReportConversation reports via the chat service; gateway declares messaging:create",
	"add_link":            "gateway-gated admin auth: AddConversationLink uses requireCaseAdmin(update); gateway declares messaging:update",
	"remove_link":         "gateway-gated admin auth: RemoveConversationLink uses requireCaseAdmin(update); gateway declares messaging:update",
	"list_links":          "gateway-gated admin auth: ListConversationLinks uses requireCaseAdmin(read); gateway declares messaging:read",
	"create_draft":        "gateway-gated admin auth: CreateReplyDraft uses requireCaseAdmin(create); gateway declares messaging:create",
	"update_draft":        "gateway-gated admin auth: UpdateReplyDraft uses requireMessagingAdmin(update); gateway declares messaging:update",
	"reject_draft":        "gateway-gated admin auth: RejectReplyDraft uses requireMessagingAdmin(update); gateway declares messaging:update",
	"approve_send_draft":  "gateway-gated admin auth: ApproveAndSendReplyDraft uses caller() + requireMessagingAdmin path; gateway declares messaging:update",

	"list_notifications":    "gateway-gated recipient auth: ListNotifications scopes to recipient(); gateway declares messaging:read",
	"mark_all_seen":         "gateway-gated recipient auth: MarkAllSeen scopes to recipient(); gateway declares messaging:update",
	"mark_dismissed":        "gateway-gated recipient auth: MarkDismissed scopes to recipient(); gateway declares messaging:update",
	"mark_read":             "gateway-gated recipient auth: MarkRead scopes to recipient(); gateway declares messaging:update",
	"mark_seen":             "gateway-gated recipient auth: MarkSeen scopes to recipient(); gateway declares messaging:update",
	"retrieve_notification": "gateway-gated recipient auth: GetNotification scopes to recipient(); gateway declares messaging:read",
	"unread_count":          "gateway-gated recipient auth: GetUnreadCount scopes to recipient(); gateway declares messaging:read",
	"unread_summary":        "gateway-gated recipient auth: GetUnreadSummary scopes to actor(); gateway declares messaging:read",

	"retrieve_memory": "gateway loader indirection: GetMemory -> resourceloaders.LoadAgentMemories (BatchGetAgentMemoriesByIDs), not GetAgentMemory RPC; endpoint declares agent_memories:read",

	"list_announcements":          "gateway-gated recipient auth: ListAnnouncements uses recipient(); gateway declares messaging:read",
	"retrieve_announcement":       "gateway-gated recipient auth: GetAnnouncement uses recipient(); gateway declares messaging:read",
	"mark_announcement_seen":      "gateway-gated recipient auth: MarkAnnouncementSeen uses recipient(); gateway declares messaging:update",
	"mark_announcement_read":      "gateway-gated recipient auth: MarkAnnouncementRead uses recipient(); gateway declares messaging:update",
	"mark_announcement_dismissed": "gateway-gated recipient auth: MarkAnnouncementDismissed uses recipient(); gateway declares messaging:update",
}

// TestEndpointPermissionsCoverCoreChecks is the drift guard: for every endpoint whose handler name matches an internal-service function, the endpoint must DECLARE every permission that function actually checks (over-declaration is allowed — the gateway gate is OR/any-of), and must declare RequiredRoleType admin when the function requires admin. Endpoints whose handler has no name-matched function (different-named service method, relation-based variable-action checks, or genuinely unprotected) are not verifiable here and pass; the named-matchable majority is checked, so a wrong declaration on those fails the build.
func TestEndpointPermissionsCoverCoreChecks(t *testing.T) {
	root := repoRootForTest(t)
	domains, actions := parsePermissionConstants(t, root)
	if len(domains) == 0 || len(actions) == 0 {
		t.Fatal("failed to parse permission constants")
	}
	checks := parseServiceChecks(t, root, domains, actions,
		"services/core-service/internal",
		"services/platform-service/internal",
		"services/auth-service/internal",
		"services/notification-service/internal",
		"services/agent-service/internal",
	)
	targets := parseGatewayHandlerTargets(t, root)
	endpoints := parseEndpointDecls(t, root, domains, actions)

	verified, unverifiable := 0, 0
	for _, e := range endpoints {
		// Explicitly-accounted endpoints take precedence: skip verification entirely.
		// This covers cases where the guard CAN resolve a check but declaring it would
		// be wrong (e.g. external-exempt global lookups whose helper requires the domain
		// for internal actors only — declaring it would false-reject exempt readers).
		if _, allowed := unverifiableEndpoints[e.slug]; allowed {
			unverifiable++
			continue
		}
		// Resolve the downstream check(s). Try the gateway handler name first
		// (often shared with the core function), then fall back to following the
		// gateway service.go to the real downstream client method(s) it calls so a
		// renamed handler (RetrieveCustomer -> GetCustomer) is still verified.
		c, ok := checks[e.handler]
		if !ok {
			for _, m := range targets[e.folder][e.handler] {
				cc, found := checks[m]
				if !found {
					continue
				}
				if c.perms == nil {
					c.perms = map[string]bool{}
				}
				for p := range cc.perms {
					c.perms[p] = true
				}
				c.admin = c.admin || cc.admin
				ok = true
			}
		}
		if !ok {
			// No literal downstream check found anywhere on the path. This is NOT a
			// silent pass: the endpoint must be explicitly accounted for, otherwise
			// it is a real coverage hole (declaration would never be checked).
			if _, allowed := unverifiableEndpoints[e.slug]; !allowed {
				t.Errorf("%s (folder %s): UNACCOUNTED — handler %q (downstream %v) has no literal permission/admin check the guard can match, and it is not in unverifiableEndpoints. Either declare the permissions it actually requires, or add it to unverifiableEndpoints with a reason (relation-variable or unprotected).", e.slug, e.folder, e.handler, targets[e.folder][e.handler])
			} else {
				unverifiable++
			}
			continue
		}
		matched := false
		for p := range c.perms {
			matched = true
			if !e.perms[p] {
				t.Errorf("%s: handler %s checks %q in the service but the endpoint does not declare it (declared: %v)", e.slug, e.handler, p, keys(e.perms))
			}
		}
		if c.admin && !e.roleAdmin {
			t.Errorf("%s: handler %s requires admin (CheckIsAdmin) but the endpoint does not declare RequiredRoleType admin", e.slug, e.handler)
		}
		if matched || c.admin {
			verified++
		}
	}
	if verified < 30 {
		t.Fatalf("drift guard only verified %d endpoints against service checks; the name-join likely broke", verified)
	}
	t.Logf("accounted for %d AgentTool endpoints: %d verified against service checks, %d explicitly unverifiable (allowlisted)", len(endpoints), verified, unverifiable)
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
