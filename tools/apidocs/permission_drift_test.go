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
// checks, so relation-variable endpoints are VERIFIED against their declared own and
// counterparty permissions rather than excused. Helper bodies contain one CheckHasPermission
// per owner branch (resource / customers / suppliers), so following them yields all three.
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
	// s.checkSalesOrderReadPermission(...), checkAccountUserCreatePermission(...).
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

// parseEndpointDecls scans the api-gateway endpoints for AgentTool endpoints and returns each one's folder, handler method, declared permissions (counterparty ones included), and role-type.
func parseEndpointDecls(t *testing.T, root string, domains, actions map[string]string) []endpointDecl {
	t.Helper()
	handlerRe := regexp.MustCompile(`svc\.\([A-Za-z]+\)\.([A-Za-z0-9_]+)`)
	permRe := regexp.MustCompile(`\{(?:Domain: )?types\.PermissionDomain([A-Za-z]+), (?:Action: )?types\.Action([A-Za-z]+)\}`)
	counterpartiesRe := regexp.MustCompile(`apiendpoint\.Counterparties\(types\.Action([A-Za-z]+)\)`)
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
		for _, m := range counterpartiesRe.FindAllStringSubmatch(src, -1) {
			if act, ok := actions[m[1]]; ok {
				e.perms[domains["Customers"]+":"+act] = true
				e.perms[domains["Suppliers"]+":"+act] = true
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
//     relation helper and the endpoint declares the matching permissions (verified by hand).
//   - gateway-gated participant auth: messaging/chat endpoints declare messaging
//     permissions at the gateway; notification-service enforces conversation membership
//     via caller/requireParticipant/resolveParticipant with no literal CheckHasPermission.
//   - gateway-gated recipient auth: bell-notification endpoints declare messaging
//     permissions at the gateway; notification-service scopes rows to the caller's
//     account_user via recipient()/actor() with no literal CheckHasPermission.
//   - gateway loader indirection: the gateway handler loads the resource via a
//     resourceloader batch-get instead of the name-matched downstream RPC.
//   - service scope-helper indirection: the service function delegates to a private
//     scope helper (writeIdentity, portalDomainScope) that is not named
//     check<Entity>Permission, and often passes the action as a variable rather than
//     a literal, so neither the helper-follow nor the literal regex can see it.
//     Enforcement is real and the endpoint declares the matching permission.
//   - conditional admin gate: the handler calls CheckIsAdmin only on a branch (an
//     already-completed or shipped order), so the guard's flat "requires admin" read
//     is wrong. Declaring RequiredRoleType admin would false-reject the ordinary case.
var unverifiableEndpoints = map[string]string{
	"validate":                 "public-by-design: ValidateAddress calls Google Address Validation with no tenant identity (pre-auth utility)",
	"list_address_suggestions": "public-by-design: AutocompleteAddress calls Google Places with no tenant identity (pre-auth utility)",

	"list_account_statuses":     "auth-only reference data: CheckIsAssignedActor, no account-status permission domain exists",
	"retrieve_account_status":   "auth-only reference data: CheckIsAssignedActor, no account-status permission domain exists",
	"list_sales_order_statuses": "auth-only reference data: CheckIsAuthenticated, no sales-order-status permission domain exists",

	"list_transaction_methods": "gateway-static enum: handler returns a hardcoded list; now gated by CheckIsAuthenticated at the gateway (auth-only, no domain permission)",
	"list_transaction_types":   "gateway-static enum: handler returns a hardcoded list; now gated by CheckIsAuthenticated at the gateway (auth-only, no domain permission)",

	"list_adjustment_types": "external-exempt global lookup: checkAdjustmentTypeReadPermission requires adjustment_types:read for internal actors but exempts external actors; declaring it would false-reject those readers at the gate",

	"export_customer_price_list":         "variable-domain export gate: ExportPriceList -> enqueueExport -> authorizeExport -> exportSpec.checkPermission(), which closes over spec.PermissionDomain (discounts) and calls CheckHasPermission(domainName, ActionRead); the domain is a variable, so the literal matcher cannot resolve it. Endpoint declares discounts:read, matching the spec",
	"start_inventory_change_logs_export": "variable-domain export gate: StartInventoryChangeLogsExport -> enqueueExport -> authorizeExport -> exportSpec.checkPermission(), which closes over spec.PermissionDomain (inventory_logs) and calls CheckHasPermission(domainName, ActionRead); the domain is a variable, so the literal matcher cannot resolve it. Endpoint declares inventory_logs:read, matching the spec",
	"analyze_realized_margins":           "service call-chain indirection: AnalyzeRealizedMargins delegates the read to AnalyzeSales, which checks CheckIsInternalActor + invoices:read; the guard folds in check*Permission helpers but does not follow service-to-service calls. Endpoint declares invoices:read, matching AnalyzeSales",

	"search": "gateway per-type dynamic gate: Search loops its providers calling identity.CheckHasPermission(domain, read) per resource type and only includes types the caller can read; endpoint declares the {sales_orders,purchase_orders,invoices,customers,items,shipments,messaging,agents}:read OR-set; no downstream name-matched handler to verify against",

	"activate_account_user": "gateway private-helper indirection: ActivateAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserUpdatePermission; endpoint declares team:update with customers:update / suppliers:update for counterparty accounts",
	"disable_account_user":  "gateway private-helper indirection: DisableAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserUpdatePermission; endpoint declares team:update with customers:update / suppliers:update for counterparty accounts",
	"remove_account_user":   "gateway private-helper indirection: RemoveAccountUser -> transitionAccountUserStatus -> coreClient.UpdateAccountUserStatus -> checkAccountUserDeletePermission; endpoint declares team:delete with customers:delete / suppliers:delete for counterparty accounts",

	"create_conversation":          "gateway-gated participant auth: CreateConversation uses caller() + membership; gateway declares messaging:create",
	"list_conversations":           "gateway-gated participant auth: ListConversations uses caller(); gateway declares messaging:read",
	"mark_conversation_read":       "gateway-gated participant auth: MarkConversationRead uses resolveParticipant(); gateway declares messaging:update",
	"retrieve_conversation":        "gateway-gated participant auth: GetConversation uses resolveParticipant(); gateway declares messaging:read",
	"create_attachment_upload_url": "gateway-gated participant auth: CreateAttachmentUploadURL requires active participant; gateway declares messaging:create",
	"list_messages":                "gateway-gated participant auth: ListMessages uses resolveParticipant(); gateway declares messaging:read",
	"send_message":                 "gateway-gated participant auth: SendMessage uses resolveParticipant(); gateway declares messaging:create",
	"list_contacts":                "gateway-gated participant auth: ListContacts is auth/account-scoped only; gateway declares messaging:read",
	"cancel_scheduled":             "gateway-gated participant auth: CancelScheduledMessage uses requireParticipant(); gateway declares messaging:update",
	"reschedule_message":           "gateway-gated sender auth: RescheduleMessage uses caller() and moves only the caller's own scheduled message; gateway declares messaging:update",
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

	"list_email_domains":  "gateway-gated account-scoped auth: ListDomains resolves the target account via accountID() and scopes the read to it, with no permission check; gateway declares messaging:read",
	"get_email_domain":    "gateway-gated account-scoped auth: GetDomain resolves the target account via accountID() and scopes the read to it, with no permission check; gateway declares messaging:read",
	"list_email_inboxes":  "gateway-gated account-scoped auth: ListInboxes resolves the target account via accountID() and scopes the read to it, with no permission check; gateway declares messaging:read",
	"get_email_inbox":     "gateway-gated account-scoped auth: GetInbox resolves the target account via accountID() and scopes the read to it, with no permission check; gateway declares messaging:read",
	"get_email_sender":    "gateway-gated account-scoped auth: GetSender resolves the target account via accountID() and scopes the read to it, with no permission check; gateway declares messaging:read",
	"set_email_sender":    "gateway-gated account-scoped auth: SetSender resolves the target account via accountID() and scopes the write to it, with no permission check; gateway declares messaging:update",
	"delete_email_sender": "gateway-gated account-scoped auth: DeleteSender resolves the target account via accountID() and scopes the delete to it, with no permission check; gateway declares messaging:delete",

	"list_portal_domains": "service scope-helper indirection: ListPortalDomains -> portalDomainReadScope -> portalDomainScope, which checks CheckIsInternalActor + account:<action> with the action as a variable; endpoint declares account:read",
	"get_portal_domain":   "service scope-helper indirection: GetPortalDomain -> portalDomainReadScope -> portalDomainScope, which checks CheckIsInternalActor + account:<action> with the action as a variable; endpoint declares account:read",

	"update_production_schedule_settings":         "service scope-helper indirection: writeIdentity(ctx, ActionUpdate) checks CheckIsInternalActor + production_schedules:<action>; endpoint declares production_schedules:update",
	"upsert_production_schedule_item_setting":     "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"delete_production_schedule_item_setting":     "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"upsert_production_schedule_resource_setting": "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"delete_production_schedule_resource_setting": "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"apply_fulfillment_recommendations":           "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"create_production_schedule_line":             "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"update_production_schedule_line":             "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"delete_production_schedule_line":             "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",

	"issue_sales_order":   "gateway private-helper indirection: IssueSalesOrder -> changeSalesOrderStatus -> coreClient.ChangeSalesOrderStatus, which checks CheckIsInternalActor + sales_orders:update; endpoint declares sales_orders:update",
	"unissue_sales_order": "gateway private-helper indirection: UnissueSalesOrder -> changeSalesOrderStatus -> coreClient.ChangeSalesOrderStatus; endpoint declares sales_orders:update",
	"close_sales_order":   "gateway private-helper indirection: CloseSalesOrder -> changeSalesOrderStatus -> coreClient.ChangeSalesOrderStatus; endpoint declares sales_orders:update",
	"open_sales_order":    "gateway private-helper indirection: OpenSalesOrder -> changeSalesOrderStatus -> coreClient.ChangeSalesOrderStatus; endpoint declares sales_orders:update",

	"delete_sales_order_line": "conditional admin gate: DeleteSalesOrderLine calls CheckIsAdmin only when the order is completed or has a shipped shipment; the ordinary path needs only the declared sales_orders:update (or its counterparty permission), so RequiredRoleType admin would false-reject it",

	"list_announcements":          "gateway-gated recipient auth: ListAnnouncements uses recipient(); gateway declares messaging:read",
	"retrieve_announcement":       "gateway-gated recipient auth: GetAnnouncement uses recipient(); gateway declares messaging:read",
	"mark_announcement_seen":      "gateway-gated recipient auth: MarkAnnouncementSeen uses recipient(); gateway declares messaging:update",
	"mark_announcement_read":      "gateway-gated recipient auth: MarkAnnouncementRead uses recipient(); gateway declares messaging:update",
	"mark_announcement_dismissed": "gateway-gated recipient auth: MarkAnnouncementDismissed uses recipient(); gateway declares messaging:update",

	"retrieve_details":  "public-by-design: GetPlaceDetails calls Google Places with no tenant identity (pre-auth utility, pairs with autocomplete)",
	"get_stripe_status": "auth-only counterparty lookup: HasStripeIntegration checks CheckIsAssignedActor + CheckCounterpartyReadAccess for external targets, no domain permission; customer checkout calls it, so declaring integrations:read would false-reject relation actors at the gate",
	"get_pricing_plans": "public reference data: billing-service ListPricingPlans returns the plan catalog with no tenant data and no permission check",
	"get_account_usage": "unscanned service: billing-service GetAccountUsage (not in the guard's service set) rejects portal actors and withholds agent_spend unless the caller holds billing:read; endpoint declares account:read",
	"get_spending_cap":  "gateway-enforced gate: GetSpendingCap reads the cap through the unguarded GetAccountContext RPC, so the declared billing:read at the gateway is the check",

	"gen_pack_list": "multi-record utility: GenPackList reads the shipment (checkShipmentReadPermission) and its sales order (checkSalesOrderReadPermission), each routing to customers:read / suppliers:read in a counterparty account; the endpoint keeps the shipments:read / sales_orders:read any-of gate multi-record utilities use, and counterparty routing cannot pair with two own permissions",

	"list_purchase_order_statuses": "auth-only reference data: delegates to ListSalesOrderStatuses (CheckIsAuthenticated), no purchase-order-status permission domain exists",

	"analyze_manufacturing": "conditional metric gate: AnalyzeManufacturing requires invoices:read and adds costs:read only for type=costsPerUnit|margin; declaring costs:read in the any-of set would admit costs-only callers at the gate. Endpoint declares invoices:read",
	"bulk_create_items":     "conditional row-level probe: BulkCreateItems requires items:create; items:update is only probed (canUpdateItems) so duplicate-SKU rows that would update an existing item fail per row; declaring it would admit update-only callers at the gate. Endpoint declares items:create",

	"analyze_demand_forecast":  "gRPC handler rename: coreClient.AnalyzeDemandForecast -> analyticsSvc.GetDemandForecast, which checks CheckIsInternalActor + invoices:read; the guard joins on the RPC name. Endpoint declares invoices:read",
	"analyze_new_customers":    "gRPC handler rename: coreClient.AnalyzeNewCustomers -> analyticsSvc.GetNewCustomersAnalytics, which checks CheckIsInternalActor + customers:read; the guard joins on the RPC name. Endpoint declares customers:read",
	"connect_production_steps": "gRPC handler rename: ConnectProductionStepsByScanningStation -> ConnectProductionStepsByName, which checks CheckIsInternalActor + scanners:update; endpoint declares scanners:update",

	"analyze_open_orders_summary":   "service scope-helper indirection: AnalyzeOpenOrdersSummary -> orderReportAccessFor(invoices), which checks CheckIsInternalActor + <domain>:read with the domain as a variable; endpoint declares invoices:read",
	"analyze_open_orders_breakdown": "service scope-helper indirection: AnalyzeOpenOrderProducts -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"list_open_orders":              "service scope-helper indirection: ListOpenOrders -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"list_open_order_lines":         "service scope-helper indirection: ListOpenOrderLines -> orderReportAccessFor(sales_orders); endpoint declares sales_orders:read",
	"export_open_order_lines":       "service scope-helper indirection: ExportOpenOrderLines -> orderReportAccessFor(sales_orders) and an exportSpec over sales_orders; endpoint declares sales_orders:read",
	"analyze_sales_summary":         "service scope-helper indirection: AnalyzeSalesSummary -> salesReportAccessFor -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"analyze_sales_breakdown":       "service scope-helper indirection: AnalyzeSalesBreakdown -> salesReportAccessFor -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"analyze_sales_invoices":        "service scope-helper indirection: AnalyzeSalesInvoices -> salesReportAccessFor -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"list_sales_lines":              "service scope-helper indirection: ListSalesLines -> salesReportAccessFor -> orderReportAccessFor(invoices); endpoint declares invoices:read",
	"export_sales_lines":            "service scope-helper indirection: ExportSalesLines -> salesReportAccessFor and salesExportSpec over invoices; endpoint declares invoices:read",

	"initialize_batch":       "service scope-helper indirection: InitializeBatch -> batchActor(ctx, ActionCreate) checks CheckIsInternalActor + batches:<action> with the action as a variable; scanners:update is checked only for a type_override / consume_materials=false supervisor correction. Endpoint declares batches:create",
	"merge_batches":          "service scope-helper indirection: MergeBatches -> batchActor(ctx, ActionCreate); endpoint declares batches:create",
	"move_batches":           "service scope-helper indirection: MoveBatches -> batchActor(ctx, ActionCreate); endpoint declares batches:create",
	"split_batch":            "service scope-helper indirection: SplitBatch -> batchActor(ctx, ActionCreate); endpoint declares batches:create",
	"get_consumption":        "service scope-helper indirection: GetScanningStationConsumption -> batchActor(ctx, ActionRead); scanners:update is checked only for a type_override. Endpoint declares batches:read",
	"get_remaining_to_split": "service scope-helper indirection: GetRemainingQuantityToSplit -> batchActor(ctx, ActionRead); endpoint declares batches:read",

	"export_customers":               "variable-domain export gate: ExportCustomers -> enqueueExport -> authorizeExport -> exportSpec.checkPermission() over customers; the domain is a variable, so the literal matcher cannot resolve it. Endpoint declares customers:read",
	"list_notification_recipients":   "service scope-helper indirection: ListCustomerNotificationRecipients -> authorizeCustomerNotificationRecipientAccess, which checks customers|suppliers:read with domain and action as variables; endpoint declares customers:read with counterparties",
	"update_notification_recipients": "service scope-helper indirection: UpdateCustomerNotificationRecipients -> authorizeCustomerNotificationRecipientAccess, which checks customers|suppliers:update with domain and action as variables; endpoint declares customers:update with counterparties",

	"list_invoices":          "service scope-helper indirection: ListInvoices -> checkInvoiceAccess(identity, ActionRead), which checks CheckIsInternalActor + invoices:<action> with the action as a variable; endpoint declares invoices:read",
	"list_customer_invoices": "service scope-helper indirection: ListCustomerInvoices -> checkInvoiceAccess(identity, ActionRead); endpoint declares invoices:read",
	"retrieve_invoice":       "service scope-helper indirection: GetInvoice -> checkInvoiceAccess(identity, ActionRead); endpoint declares invoices:read",
	"update_invoice":         "service scope-helper indirection: UpdateInvoice -> checkInvoiceAccess(identity, ActionUpdate); endpoint declares invoices:update",

	"archive_production_schedule":      "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"publish_production_schedule":      "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"release_production_schedule_week": "service scope-helper indirection: writeIdentity(ctx, ActionUpdate); endpoint declares production_schedules:update",
	"delete_production_schedule":       "service scope-helper indirection: writeIdentity(ctx, ActionDelete); endpoint declares production_schedules:delete",

	"bulk_create_production_runs":   "variable-domain bulk gate: BulkCreateProductionRuns -> enqueueBulkOperation, which checks CheckHasPermission(spec.PermissionDomain, a) for each spec action (production_runs [create]); endpoint declares production_runs:create",
	"bulk_upsert_production_steps":  "variable-domain bulk gate: enqueueBulkOperation checks every spec action (production_steps [create, update]); endpoint requires both",
	"bulk_upsert_scanning_stations": "variable-domain bulk gate: enqueueBulkOperation checks every spec action (scanners [create, update]); endpoint requires both",
	"bulk_upsert_departments":       "variable-domain bulk gate: enqueueBulkOperation checks every spec action (departments [create, update]); endpoint requires both",
	"bulk_upsert_item_categories":   "variable-domain bulk gate: enqueueBulkOperation checks every spec action (item_categories [create, update]); endpoint requires both",
	"bulk_upsert_locations":         "variable-domain bulk gate: enqueueBulkOperation checks every spec action (locations [create, update]); endpoint requires both",
	"bulk_upsert_machines":          "variable-domain bulk gate: enqueueBulkOperation checks every spec action (machines [create, update]); endpoint requires both",
	"bulk_upsert_materials":         "variable-domain bulk gate: enqueueBulkOperation checks every spec action (materials [create, update]); endpoint requires both",
	"bulk_upsert_parts":             "variable-domain bulk gate: enqueueBulkOperation checks every spec action (parts [create, update]); endpoint requires both",
	"bulk_upsert_product_lines":     "variable-domain bulk gate: enqueueBulkOperation checks every spec action (product_lines [create, update]); endpoint requires both",
	"bulk_upsert_products":          "variable-domain bulk gate: enqueueBulkOperation checks every spec action (items [create, update]); endpoint requires both",
	"bulk_upsert_properties":        "variable-domain bulk gate: enqueueBulkOperation checks every spec action (properties [create, update]); endpoint requires both",
	"bulk_upsert_unit_groups":       "variable-domain bulk gate: enqueueBulkOperation checks every spec action (unit_groups [create, update]); endpoint requires both",
	"bulk_upsert_units":             "variable-domain bulk gate: enqueueBulkOperation checks every spec action (units [create, update]); endpoint requires both",

	"export_production_runs":   "variable-domain export gate: enqueueExport -> authorizeExport -> exportSpec.checkPermission() over production_runs; endpoint declares production_runs:read",
	"export_production_steps":  "variable-domain export gate: exportSpec over production_steps; endpoint declares production_steps:read",
	"export_scanning_stations": "variable-domain export gate: exportSpec over scanners; endpoint declares scanners:read",
	"export_departments":       "variable-domain export gate: exportSpec over departments; endpoint declares departments:read",
	"export_item_categories":   "variable-domain export gate: exportSpec over item_categories; endpoint declares item_categories:read",
	"export_locations":         "variable-domain export gate: exportSpec over locations; endpoint declares locations:read",
	"export_machines":          "variable-domain export gate: exportSpec over machines; endpoint declares machines:read",
	"export_product_lines":     "variable-domain export gate: exportSpec over product_lines; endpoint declares product_lines:read",
	"export_properties":        "variable-domain export gate: exportSpec over properties; endpoint declares properties:read",
	"export_unit_groups":       "variable-domain export gate: exportSpec over unit_groups; endpoint declares unit_groups:read",
	"export_units":             "variable-domain export gate: exportSpec over units; endpoint declares units:read",
	"export_materials":         "export helper indirection: exportSpec.CheckPermission = checkMaterialReadPermission (materials:read, or customers/suppliers:read in a counterparty account); endpoint declares materials:read with counterparties",
	"export_parts":             "export helper indirection: exportSpec.CheckPermission = checkPartReadPermission (parts:read, or customers/suppliers:read in a counterparty account); endpoint declares parts:read with counterparties",
	"export_products":          "export helper indirection: exportSpec.CheckPermission = checkProductReadPermission (items:read, or customers/suppliers:read in a counterparty account); endpoint declares items:read with counterparties",

	"redact_conversation": "gateway-gated admin auth: RedactConversation uses requireMessagingAdmin(delete); gateway declares messaging:delete",
	"set_support_route":   "gateway-gated admin auth: SetSupportRoute uses requireMessagingAdmin(update); gateway declares messaging:update",
	"clear_support_route": "gateway-gated admin auth: ClearSupportRoute uses requireMessagingAdmin(update); gateway declares messaging:update",
	"get_support_route":   "gateway-gated admin auth: GetSupportRoute uses requireMessagingAdmin(read); gateway declares messaging:read",
	"create_email_domain": "gateway-gated account-scoped auth: CreateDomain resolves the target account via accountID() (CheckIsInternalActor) and scopes the write to it, with no permission check; gateway declares messaging:create",
	"verify_email_domain": "gateway-gated account-scoped auth: VerifyDomain resolves the target account via accountID() and scopes the write to it; gateway declares messaging:update",
	"delete_email_domain": "gateway-gated account-scoped auth: DeleteDomain resolves the target account via accountID() and scopes the delete to it; gateway declares messaging:delete",
	"create_email_inbox":  "gateway-gated account-scoped auth: CreateInbox resolves the target account via accountID() and scopes the write to it; gateway declares messaging:create",
	"update_email_inbox":  "gateway-gated account-scoped auth: UpdateInbox resolves the target account via accountID() and scopes the write to it; gateway declares messaging:update",
	"delete_email_inbox":  "gateway-gated account-scoped auth: DeleteInbox resolves the target account via accountID() and scopes the delete to it; gateway declares messaging:delete",

	"start_hubspot_sync":                   "service scope-helper indirection: StartSync -> coreClient.StartHubspotBackfill -> StartBackfill -> authorize(ctx, ActionUpdate), which checks CheckIsInternalActor + integrations:<action> with the action as a variable; endpoint declares integrations:update",
	"get_current_hubspot_sync":             "service scope-helper indirection: GetCurrentJob -> authorize(ctx, ActionRead); endpoint declares integrations:read",
	"get_hubspot_sync_job":                 "service scope-helper indirection: GetJob -> authorize(ctx, ActionRead); endpoint declares integrations:read",
	"list_hubspot_company_reviews":         "service scope-helper indirection: ListReviews -> authorize(ctx, ActionRead); endpoint declares integrations:read",
	"list_hubspot_sync_records":            "service scope-helper indirection: ListRecords -> authorize(ctx, ActionRead); endpoint declares integrations:read",
	"create_new_hubspot_company_review":    "service scope-helper indirection: coreClient.ResolveHubspotCompanyReview -> ResolveReview -> authorize(ctx, ActionUpdate); endpoint declares integrations:update",
	"link_hubspot_company_review":          "service scope-helper indirection: coreClient.ResolveHubspotCompanyReview -> ResolveReview -> authorize(ctx, ActionUpdate); endpoint declares integrations:update",
	"skip_hubspot_company_review":          "service scope-helper indirection: coreClient.ResolveHubspotCompanyReview -> ResolveReview -> authorize(ctx, ActionUpdate); endpoint declares integrations:update",
	"execute_hubspot_sync":                 "service scope-helper indirection: StartExecute -> authorize(ctx, ActionUpdate); endpoint declares integrations:update",
	"cancel_hubspot_sync":                  "service scope-helper indirection: CancelJob -> authorize(ctx, ActionUpdate); endpoint declares integrations:update",
	"bulk_resolve_hubspot_company_reviews": "variable-domain bulk gate: BulkResolveReviews -> enqueueBulkOperation, which checks every spec action (integrations [update]); endpoint declares integrations:update",
	"export_hubspot_company_reviews":       "service scope-helper indirection: ExportReviews -> authorize(ctx, ActionRead) then enqueueExport over integrations; endpoint declares integrations:read",

	"create_portal_domain": "service scope-helper indirection: CreatePortalDomain -> portalDomainWriteScope -> portalDomainScope, which checks CheckIsInternalActor + account:<action> with the action as a variable; endpoint declares account:update",
	"delete_portal_domain": "service scope-helper indirection: DeletePortalDomain -> portalDomainWriteScope -> portalDomainScope; endpoint declares account:update",
	"verify_portal_domain": "service scope-helper indirection: VerifyPortalDomain -> portalDomainWriteScope -> portalDomainScope; endpoint declares account:update",
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
