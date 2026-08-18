//go:build e2e

package api_test

import "strings"

// Seed data constants from shared/db/seed/ and tools/apidocs/httpie_seed_data.go.
// These are IDs and values known to exist after seeding.
const (
	SeedAPIKey      = "aug_sk_prod_u6Xh5ZpaUruMAU12EPAs4z_rSA4zJM5NbRqAtalvXMoRWOUPohFKJtX7ZUFUOp36IVwdiUCZu"
	SeedAccountID   = "ac_01k0a5smf9ekb8rqg12555zjqa"
	SeedAccountSlug = "acme-inc" // account_portal.slug owned by SeedAccountID (registration account_slug tests)
	SeedUserID      = "us_1wjfmmbwg8l7"
	SeedAdmin2ID    = "us_2ndadmin0000" // Mike Johnson, second Admin in SeedAccountID
	SeedUser2ID     = "us_6p7460uuwibz"

	// Customers
	SeedCustomerAccountID  = "ac_01k09wm2fgevdsc344gpbcj30f"
	SeedCustomerName       = "Global Manufacturing Solutions"
	SeedCustomerNumber     = "45678" // acre_01seedcustomer00000 relation number (acme → GMS); existing-customer registration lookup
	SeedCustomerGroupID    = "acgp_01k0a413mjeth8pe1g70t0thax"
	SeedCustomerGroupName  = "DME"
	SeedAccountRelationID  = "acre_01seedcustomer00000"
	SeedRegistrationFlowID = "mock-registration-flow"

	// account_user belonging to SeedCustomerAccountID (the buyer/customer account),
	// user "Jane Doe" <dev@augno.com>. A valid order email-contact recipient, which
	// must resolve within the BUYER's account (not the seller/acting account).
	SeedCustomerAccountUserID = "acus_01seedcustuser00000"
	SeedCustomerUserEmail     = "dev@augno.com"

	// Customer portal API key (owned by customer account, targets vendor account)
	SeedCustomerAPIKey = "aug_sk_prod_CustPortalE2eTestKey1_CustomerPortalE2eTestSecretValueForAuthTestingPurpose12345efS0Og"
	SeedAddressID      = "ad_01k09wnac0e1ar211e0sy0ba4g"

	// Catalog
	SeedProductLineID   = "pdln_01k0a735ype5e8nrhv1n5dhq1q"
	SeedProductLineName = "Socks"
	SeedItemID          = "it_01k0a7100aeysrs9vxpeq14yxj"
	// The greige item the constraint department plans, as opposed to the finished goods it becomes.
	SeedGreigeItemID     = "it_01seedlknitem000000"
	SeedItemSKU          = "SCK-001"
	SeedItemDescription  = "Small white sock"
	SeedItemCategoryID   = "itcg_01seedsocks000000"
	SeedItemCategoryName = "Socks"
	SeedProductID        = "pd_01k0a65nx2e2crfxrvryyxnmdh"
	SeedProductTypeID    = "pdtp_01seedsale00000000"

	// PUT include walkers (tests/e2e/api/meta_includes_test.go): isolated fixtures for mutate-then-include coverage.
	SeedIncludePutAlternateItemCategoryID   = "itcg_01seedshipping000"
	SeedIncludePutProductLineChangeTargetID = "pdln_01gf7a8200ef99y3gj77z4q25z"
	SeedIncludePutChangeCategoryItemID      = "it_01seed_putinc_chgcat00"
	SeedIncludePutAddAttributeItemID        = "it_01seed_putinc_attradd0"
	SeedIncludePutChangeProductLineSaleID   = "pd_01seed_putinc_chprdln0"
	SeedIncludePutChangeProductLineItemID   = "it_01seed_putinc_chprdln_it"
	SeedIncludePutEstimateSalesOrderID      = "or_01k0a8bs2yfhev5begay245wez"
	SeedIncludePutEstimatePurchaseOrderID   = "or_01seed_putinc_po_es00"
	SeedPropertyID                          = "pp_01k0a7ntn1ez6aw8x850femxeh"
	SeedPropertyName                        = "Color"
	SeedAttributeID                         = "at_01seedbeige00000000"

	// Catalog search rank fixtures (shared/db/seed/e2e/0014_e2e_extras.sql; list ?q=621).
	SeedSearchRankQuery             = "621"
	SeedPartSearchRankExactSKU      = "621"
	SeedPartSearchRankTokenSKU      = "rkpt7f3a 621"
	SeedPartSearchRankPrefixSKU     = "621rkpt8f3a"
	SeedPartSearchRankLooseSKU      = "rkpt47562183"
	SeedMaterialSearchRankTokenSKU  = "rkmt9b2c 621"
	SeedMaterialSearchRankPrefixSKU = "621rkmt8f3a"
	SeedMaterialSearchRankLooseSKU  = "rkmt47562183"
	SeedProductSearchRankTokenSKU   = "rkrpd4e1f 621"
	SeedProductSearchRankPrefixSKU  = "621rkrp9pfx"
	SeedProductSearchRankLooseSKU   = "rkrpd56214z"

	itemsPath = "/v1/catalog/items"

	// Units
	SeedUnitID          = "un_01seedpair000000000"
	SeedSystemUnitID    = "each" // global system unit (account_id IS NULL, abbreviation "ea"); modify/delete → 400 "system units cannot be modified/deleted"
	SeedUnitGroupID     = "ungp_01k0a5ecy9edg9za40dnccw53n"
	SeedUnitGroupUnitID = "ungpun_01seedsocksea000"

	// Measures
	SeedRateID     = "rt_01seedwssunitval000"
	SeedQuantityID = "qu_01seediss_ln1_qty00"

	// Infrastructure
	SeedDepartmentID = "dp_01k0a5r01yfx3sj1vy9qgv3dc0"
	SeedMachineID    = "mc_01k0a52fb6eqhtbx9hdxj3vvnh"
	// Closed, dated ~60 days back so it never collides with the OEE window tests.
	SeedMachineDowntimeEventID   = "mcdt_01seede2edowntime01"
	SeedDemandOverrideID         = "deov_01seede2eoverride1"
	SeedProductionScheduleID     = "pnsc_01seede2eschedule"
	SeedProductionScheduleLineID = "pnscln_01seede2eline01" // generated in real use; seeded only so nested {id} paths resolve
	SeedLocationID               = "sglc_01seedbuilding0000"
	SeedLocationParentID         = "sglc_01seedcampus00000" // parent of SeedLocationID (type=campus); for ?include=parent on the stable fixture
	SeedScanningStationID        = "sgsn_01k0a8201zegarjfsjaw5n7yfv"

	// Operations
	SeedProductionRunID  = "pnrn_01seedprod_run0000"
	SeedProductionStepID = "prs_01k0a51qxceydax5036pegvzzy"
	// Sew Large Sock — has inbound graph edge (Knit) and a seeded machine; used for GET/PATCH /production-steps/{id}.
	SeedSewLargeProductionStepID = "prs_01k0a56yc1e8wag6wexn4pp8t9"
	// Small sock line — edges Knit Small → Wash Small in shared/db/seed/0009_production.sql
	SeedKnitSmallProductionStepID = "prs_01k0a575j3fqr97khk36v114nj"
	SeedWashSmallProductionStepID = "prs_01k0a57f3dfsmtzc8txbq43eth"
	SeedMaterialID                = "ml_01seedyrn1mat000000"
	SeedMaterialItemID            = "it_01seedyrn1item00000"
	// item_category_id on SeedMaterialItemID (yarn materials in 0007_items.sql), not SeedItemCategoryID (socks).
	SeedMaterialCategoryID = "itcg_01seedyarn0000000"
	SeedPartID             = "pt_01seedlknpart000000"
	SeedPartItemID         = "it_01seedlknitem000000"
	SeedLsnItemID          = "it_01seedlsnitem000000"
	SeedLknItemSKU         = "LKN" // Large Knitted Sock — initial subassembly (root step: Knit Large Sock)
	SeedSknItemID          = "it_01seedsknitem000000"
	SeedSknItemSKU         = "SKN" // Small Knitted Sock — initial subassembly (root step: Knit Small Sock)
	SeedLsnItemSKU         = "LSN" // Large Sewn Sock — downstream part (produced by Sew Large Sock, which has Knit as parent)
	SeedConsumptionID      = "cp_01seedcons_kl_yarn1"
	SeedProductionID       = "pn_01seedprod_knitlg00"

	// Orders
	SeedSalesOrderID     = "or_01k0a8bs2yejxbsvqhrx4drkq1" // ORD-001: issued, $50 partial settlement allocation -> partially_paid
	SeedSalesOrderLineID = "orln_01seediss_ln1_0000"
	// Payment-status parity orders.
	SeedEstimateOrderID      = "or_01k0a8bs2yfhev5begay245wez" // EST-001: estimate, no invoice/payment -> unpaid
	SeedFulfilledPaidOrderID = "or_01k0a8bs2yf909wjkd7ecd6x4z" // ORD-003: fulfilled, invoice paid in full -> paid
	// ORD-PAID-NOALLOC (0014_e2e_extras.sql): fulfilled, invoice paid in full but
	// NO settlement allocation -> paid. Regression fixture for the reported bug
	// where allocation-derived logic showed it unpaid.
	SeedPaidNoAllocOrderID = "or_01seedpaidnoalloc00"
	// List filter-coverage orders (shared/db/seed/e2e/0016_e2e_filter_coverage.sql).
	// Both use a far-future created_at so they stay on the first list page.
	SeedInternalSalesOrderID = "or_01seedfcsointernal" // buyer == owner == SeedAccountID
	SeedPOSalesOrderID       = "or_01seedfcsopo00000"  // carries SeedSalesOrderPONumber
	SeedSalesOrderPONumber   = "PO-E2E-EXACT-001"
	SeedShipmentID           = "sh_01k0a87w33emw8pmkz1mf86cg1"
	SeedDeliveryID           = "dv_01seeddelivery1_0000"
	SeedShipmentLineID       = "shln_01seedshpln1_00000"
	SeedShippingCaseID       = "shcs_01seedshcase1_00000"
	SeedPickID               = "pk_01k0a5tsn7f7psgagr1732fxqa"
	SeedPickLineID           = "pkln_01seediss_ln1_0000"
	SeedReceivingOrderLineID = "rcln_01seedrecvln1_0000"
	SeedInvoiceID            = "iv_01k09wnac0e1ar211e0sy0ba4g"
	// Purchase order (sales_order row, type=purchase_order; seeded in 0014_e2e_extras.sql).
	SeedPurchaseOrderID     = "or_01seedpurchord1_000"
	SeedPurchaseOrderNumber = "PO-001"

	// Auth
	SeedAdminRoleID                    = "rl_mtg88e6u6fbu"
	SeedSalesRepRoleID                 = "rl_hh6mrlkv08n8"
	SeedScannerRoleID                  = "rl_scanner"
	SeedAPIKeyID                       = "apky_pajbskcck3cabxajdh8h8"
	SeedAccountUserID                  = "acus_s83fjhyfmqen"
	SeedAdmin2AccountUserID            = "acus_2ndadmin000"    // Mike Johnson (us_2ndadmin0000, Admin) in SeedAccountID
	SeedAccountUser2ID                 = "acus_ubdx4zebgl6p"   // Sarah Martinez (us_6p7460uuwibz, Sales Rep) in SeedAccountID
	SeedSalesRepStaleFlagAccountUserID = "acus_e2esrep0flag00" // sales_rep role with is_commission_eligible=0 (pre-backfill shape)
	SeedJobID                          = "jb_01seedincludejob0"

	// Batches, sales targets, integrations (seeded in 0014_e2e_extras.sql)
	SeedBatchID              = "bt_01seedbatch1_0000000"
	SeedSalesTargetID        = "tg_01seedtarget1_000000"
	SeedAccountIntegrationID = "acin_01seedintegration1"

	// HubSpot sync (0014_e2e_extras.sql): one review_pending job with one
	// pending company review, so the hubspot-sync read endpoints resolve their
	// {id}/{review_id} path params and the company-reviews list returns an item.
	SeedHubspotSyncJobID       = "igjb_01seedhubspotjob1"
	SeedHubspotCompanyReviewID = "igrv_01seedhubspotrev1"
	SeedHubspotSyncRecordID    = "igrd_01seedhubspotrec1"

	// Priorities
	SeedPriorityID   = "pi_01seednormal0000000000"
	SeedPriorityCode = "normal"
	SeedPriorityName = "Normal"

	// Account Statuses (seeded in 0001_static_types.sql)
	SeedAccountStatusID   = "acss_01seednormal000000"
	SeedAccountStatusCode = "normal"
	SeedAccountStatusName = "Normal"

	// Request logs (0014_e2e_extras.sql)
	SeedRequestLogID             = "rqlog_01seedreqlog1_000" // linked from SeedAuditEventID for include=request tests
	SeedRequestLogIdempotencyKey = "e2e-seed-idempotency-key-01"
	// SeedRequestLogSearchToken is a distinctive resource id embedded in the path
	// of rqlog_01seedsearchtgt0 (path=/v1/catalog/items/it_01seedreqlogsrchtgt).
	// Searching ?q=<token> must return that log via the route/path search.
	SeedRequestLogSearchToken = "it_01seedreqlogsrchtgt"

	// --- Request-log filter cohorts (0014_e2e_extras.sql) ---
	// Each filter dimension has a dedicated 3-row cohort sharing a distinctive
	// scope value (a synthetic normalized_route, or a synthetic host where the
	// route is itself the dimension under test) that the e2e harness never emits.
	// A filter test ANDs the scope value with the filter under test so the result
	// set is exactly the cohort, then asserts two values are included and the
	// third is excluded. See crud_request_logs_test.go.

	// Shared synthetic host for cohorts scoped by normalized_route.
	SeedReqLogFilterHost = "rqlog-filter-e2e.test"

	// methods cohort (scope normalized_route=/filtertest/methods).
	SeedReqLogFilterMethodsRoute = "/filtertest/methods"
	SeedReqLogFilterMethodGet    = "rqlog_01fltmethget00" // method=GET
	SeedReqLogFilterMethodPost   = "rqlog_01fltmethpost0" // method=POST
	SeedReqLogFilterMethodPut    = "rqlog_01fltmethput00" // method=PUT (excluded)

	// status_codes cohort (scope normalized_route=/filtertest/status).
	SeedReqLogFilterStatusRoute = "/filtertest/status"
	SeedReqLogFilterStatus200   = "rqlog_01fltstat20000" // status_code=200
	SeedReqLogFilterStatus404   = "rqlog_01fltstat40400" // status_code=404
	SeedReqLogFilterStatus500   = "rqlog_01fltstat50000" // status_code=500 (excluded)

	// account_ids cohort (scope normalized_route=/filtertest/accounts).
	// account_id is the acting account; target_account_id is the seed account on
	// all three so they are visible. account_id is not surfaced in the response.
	SeedReqLogFilterAccountsRoute = "/filtertest/accounts"
	SeedReqLogFilterAccount1      = "rqlog_01fltacct1000" // account_id=SeedAccountID
	SeedReqLogFilterAccount2      = "rqlog_01fltacct2000" // account_id=SeedCustomerAccountID
	SeedReqLogFilterAccount3      = "rqlog_01fltacct3000" // account_id=SeedChildAccountID1 (excluded)

	// actor-or-target scope cohort (scope normalized_route=/filtertest/scope).
	// Caller is the seed account; rows cover every actor/target quadrant.
	SeedReqLogScopeRoute   = "/filtertest/scope"
	SeedReqLogScopeActor   = "rqlog_01fltscopeactr" // account_id=seed, target=customer (actor side)
	SeedReqLogScopeTarget  = "rqlog_01fltscopetgt0" // account_id=customer, target=seed (target side)
	SeedReqLogScopeBoth    = "rqlog_01fltscopeboth" // account_id=seed, target=seed
	SeedReqLogScopeNeither = "rqlog_01fltscopenone" // account_id=child, target=customer (out of scope)

	// actor_ids cohort (scope normalized_route=/filtertest/actorids).
	SeedReqLogFilterActorIDsRoute = "/filtertest/actorids"
	SeedReqLogFilterActorUser1    = "rqlog_01fltactid100" // actor_id=SeedUserID
	SeedReqLogFilterActorUser2    = "rqlog_01fltactid200" // actor_id=SeedUser2ID
	SeedReqLogFilterActorUser3    = "rqlog_01fltactid300" // actor_id=us_fltactor3 (excluded)
	SeedReqLogFilterActorUser3ID  = "us_fltactor3"

	// actor_types cohort (scope normalized_route=/filtertest/actortypes).
	SeedReqLogFilterActorTypesRoute = "/filtertest/actortypes"
	SeedReqLogFilterTypeUser        = "rqlog_01flttypeusr0" // identity_type=user
	SeedReqLogFilterTypeAPIKey      = "rqlog_01flttypekey0" // identity_type=api_key
	SeedReqLogFilterTypeInternal    = "rqlog_01flttypeint0" // identity_type=internal (excluded)

	// infra-scrub cohort (scope normalized_route=/filtertest/infra-scrub). The
	// agent log carries an internal host + pod IP that must be scrubbed in
	// customer-facing responses; the user log keeps its real (public) host + IP.
	// SeedAuditEventInfraAgentID embeds the agent log via request_id.
	SeedReqLogInfraScrubRoute = "/filtertest/infra-scrub"
	SeedReqLogInfraAgentID    = "rqlog_01infraagent0"       // identity_type=agent -> scrubbed
	SeedReqLogInfraAgentHost  = "api-gateway-internal:8091" // internal host that must NOT leak
	SeedReqLogInfraAgentIP    = "10.244.0.18"               // pod IP that must NOT leak
	// The agent actor on SeedReqLogInfraAgentID. Its name + handle(slug) are
	// resolved from agent-service (agdf_01infraseedagent), not joined in
	// platform-service, so ?include=actor must hydrate them.
	SeedReqLogInfraAgentActorID     = "agdf_01infraseedagent"
	SeedReqLogInfraAgentActorName   = "Infra Scrub Agent"
	SeedReqLogInfraAgentActorHandle = "infra_scrub_agent"
	// api_version the agent's internal call carried; must survive into the
	// customer-facing log (it is not internal infrastructure, so not scrubbed).
	SeedReqLogInfraAgentAPIVersion = "1.0.forge-preview.2"
	SeedReqLogInfraUserID          = "rqlog_01infrauser00" // identity_type=user -> preserved
	SeedReqLogInfraUserHost        = "api.augno.com"
	SeedReqLogInfraUserIP          = "198.51.100.7"
	// RedactedRequestLogHost mirrors apiresource.RedactedRequestLogHost (kept as a
	// literal so the e2e suite stays a black-box client of the API): the public API
	// host shown in place of the internal listener hostname for agent requests.
	SeedReqLogRedactedHost = "https://api.augno.com"

	// Audit event whose request_id points at SeedReqLogInfraAgentID, covering the
	// audit-event ?include=request expansion of an internal/agent request_log.
	SeedAuditEventInfraAgentID = "adev_01infraauditreq0"

	// normalized_routes cohort (scope host=rqlog-route-e2e.test).
	SeedReqLogFilterRouteHost = "rqlog-route-e2e.test"
	SeedReqLogFilterRouteA    = "/filtertest/route-a"
	SeedReqLogFilterRouteB    = "/filtertest/route-b"
	SeedReqLogFilterRouteC    = "/filtertest/route-c" // excluded
	SeedReqLogFilterRouteAID  = "rqlog_01fltroutea00"
	SeedReqLogFilterRouteBID  = "rqlog_01fltrouteb00"
	SeedReqLogFilterRouteCID  = "rqlog_01fltroutec00"

	// normalized_route param-name drift cohort (scope host=rqlog-drift-e2e.test).
	// The stored route uses the Go router's snake_case param name; the dashboard
	// endpoint filter sends the Stainless public-spec form, which camelCases
	// multi-word path params. The filter compares on route shape, so the
	// camelCase template must still match the snake_case stored row.
	SeedReqLogFilterDriftHost   = "rqlog-drift-e2e.test"
	SeedReqLogFilterDriftStored = "/v1/catalog/unit-groups/{unit_group_id}/units" // as the router stores it
	SeedReqLogFilterDriftCamel  = "/v1/catalog/unit-groups/{unitGroupId}/units"   // as the dashboard filter sends it
	SeedReqLogFilterDriftID     = "rqlog_01fltdrift000"

	// hosts cohort (scope normalized_route=/filtertest/hosts).
	SeedReqLogFilterHostsRoute = "/filtertest/hosts"
	SeedReqLogFilterHostA      = "rqlog-hosta-e2e.test"
	SeedReqLogFilterHostB      = "rqlog-hostb-e2e.test"
	SeedReqLogFilterHostC      = "rqlog-hostc-e2e.test" // excluded
	SeedReqLogFilterHostAID    = "rqlog_01flthosta000"
	SeedReqLogFilterHostBID    = "rqlog_01flthostb000"
	SeedReqLogFilterHostCID    = "rqlog_01flthostc000"

	// min_latency_us cohort (scope normalized_route=/filtertest/latency).
	SeedReqLogFilterLatencyRoute = "/filtertest/latency"
	SeedReqLogFilterLatencyLo    = "rqlog_01fltlatlo000" // latency_us=1000 (excluded by threshold)
	SeedReqLogFilterLatencyMid   = "rqlog_01fltlatmid00" // latency_us=50000
	SeedReqLogFilterLatencyHi    = "rqlog_01fltlathi000" // latency_us=100000

	// date-range cohort (scope normalized_route=/filtertest/dates).
	SeedReqLogFilterDatesRoute = "/filtertest/dates"
	SeedReqLogFilterDateOld    = "rqlog_01fltdateold0" // occurred_at=2023-01-01
	SeedReqLogFilterDateMid    = "rqlog_01fltdatemid0" // occurred_at=2023-06-01
	SeedReqLogFilterDateNew    = "rqlog_01fltdatenew0" // occurred_at=2023-12-01

	// error_codes cohort (scope normalized_route=/filtertest/errors).
	SeedReqLogFilterErrorsRoute   = "/filtertest/errors"
	SeedReqLogFilterErrorNotFound = "rqlog_01flterrnf000" // error_code=resource_not_found
	SeedReqLogFilterErrorValidate = "rqlog_01flterrvf000" // error_code=validation_failed
	SeedReqLogFilterErrorAuth     = "rqlog_01flterrua000" // error_code=invalid_credentials (excluded)

	// Sandboxes
	SeedSandboxAccountID = "ac_sandbox_01k0a5smf9ekb8rqg12555zjqb"
	SeedSandboxID        = "sbac_01seedsandbox000000" // sandbox_account.type_id, used by GET /v1/core/sandboxes/{id}

	// Payment/Shipping
	SeedPaymentTermID        = "pytm_01seednet3000000"
	SeedDefaultPaymentTermID = "pytm_01seeddefault00000"
	SeedShippingTermID       = "prepaid_billed"
	SeedCustomShippingTermID = "shtm_01seedcustflat000" // account-owned, seeded in 0014_e2e_extras.sql
	SeedCarrierID            = "delivery"
	SeedServiceLevelID       = "crop_01seedground000000"
	SeedSystemCarrierID      = "syscar_01seedsysdefault"
	SeedSystemServiceLevelID = "crop_01seedsysground000"

	// Carrier transit fixtures (0014_e2e_extras.sql). Unlike SeedCarrierID this carrier
	// has a Shippo account, so rating reaches the stub and lanes actually warm.
	SeedTransitCarrierID = "cr_01e2etransitcarrier"
	// Rated by the stub: ground 3 days, 2-day 2 days, overnight 1 day.
	SeedTransitGroundServiceLevelID    = "crop_01e2etransitgrnd00"
	SeedTransitTwoDayServiceLevelID    = "crop_01e2etransit2day0"
	SeedTransitOvernightServiceLevelID = "crop_01e2etransitovrn0"
	// No service level token, so no quote can ever match: the only transit these can
	// produce is the service level's own default (5 days), or none at all.
	SeedTransitDefaultOnlyServiceLevelID = "crop_01e2etransitdflt0"
	SeedTransitNoTransitServiceLevelID   = "crop_01e2etransitnone0"

	// System
	SeedSysPropertyID = "sypp_01seedtxnumber000"

	// Finance (seeded in 0013_finance.sql)
	SeedDCLocationID            = "dclc_01seeddc_location0"
	SeedSettlementID            = "sl_01seedsettlement000"
	SeedTransactionID           = "tx_01seedtransaction00"
	SeedTransactionAllocationID = "txal_01seedtxalloc0000"

	// Static transaction methods/types (hardcoded rows in
	// services/api-gateway/endpoints/transactions/service.go; not DB-seeded, IDs
	// derived via seedID(prefix, suffix)).
	SeedTransactionMethodCashID     = "txmd_01seedcash00000000"
	SeedTransactionMethodCheckID    = "txmd_01seedcheck0000000"
	SeedTransactionTypePaymentID    = "txtp_01seedpayment000000"
	SeedTransactionTypeCreditMemoID = "txtp_01seedcreditmemo000"
	SeedTransactionTypeAdjustmentID = "txtp_01seedadjustment000"
	SeedTransactionTypeRebateID     = "txtp_01seedrebate0000000"

	// Adjustment types (fixed rows, shared/db/seed/0001_static_types.sql).
	SeedAdjustmentTypeID   = "ajtp_01seeddiscount00000"
	SeedAdjustmentTypeCode = "discount"
	SeedAdjustmentTypeName = "Discount"

	// Discounts/Pricing (seeded in 0010/0011)
	SeedOrderDiscountID     = "ords_01seedpct10discount"
	SeedAccountPriceID      = "acpr_01seedaccprice0000"
	SeedVolumeDiscountID    = "quds_01seedvoldiscount0"
	SeedProductLineAccessID = "acrepdln_01seedcustpl0"

	// Suppliers (seeded in 0014_e2e_extras.sql)
	SeedSupplierAccountID  = "ac_01seedsupplier_acct0"
	SeedSupplierRelationID = "acre_01seedsupplier0000"
	SeedSupplierMaterialID = "spml_01seedsupmat1_0000"

	// Email Logs (seeded in 0014_e2e_extras.sql)
	SeedEmailLogID1 = "emlog_01seedemaillog1_0"
	SeedEmailLogID2 = "emlog_01seedemaillog2_0"

	// Email Bridge (domain + inbox seeded in 0014_e2e_extras.sql)
	SeedEmailDomainID = "emdom_01seeddomain1_00"
	SeedEmailInboxID  = "eminb_01seedinbox1_000"

	// Territories (seeded in 0014_e2e_extras.sql)
	SeedTerritoryID = "tr_01seedterritory1_000"

	// Receiving Orders (seeded in 0014_e2e_extras.sql)
	SeedReceivingOrderID = "rcor_01seedrecvorder1_0"

	// Child Accounts (seeded in 0014_e2e_extras.sql)
	SeedHouseAccountRelationID = "acre_01seedhouseacct0000"
	SeedChildAccountID1        = "ac_01seedchild_acct0001"
	SeedChildRelationID1       = "acre_01seedchild_rel001"
	SeedChildAccountID2        = "ac_01seedchild_acct0002"
	SeedChildRelationID2       = "acre_01seedchild_rel002"
	// account_user scoped to SeedChildAccountID1 (user "Blocked Child User"), for
	// the messaging_blocks cross-account block target.
	SeedChildAccountUserID = "acus_childblktgt"

	// Agents (seeded in agent-service 00001_seed_e2e_data.sql)
	SeedAgentDefinitionID         = "agdf_01seede2e_orderbot0"
	SeedCustomAgentDefinitionID   = "agdf_01seede2e_custom00"
	SeedInactiveAgentDefinitionID = "agdf_01seede2e_inact00" // custom, status=inactive, no role/tools
	SeedAgentConfigID             = "agcf_01seede2e_ordercfg0"
	SeedAgentMemoryID             = "agmm_01seede2e_memory01"
	SeedAgentRunID                = "agrn_01seede2e_run00001" // run #1 has agent_definition + config (needed for include=definition.config)
	// Dedicated terminal-state runs for the one-shot retry/continue happy paths.
	// Do not reuse for other tests — Retry/Continue flip them permanently.
	SeedAgentRunFailedID        = "agrn_01seede2e_runfail1" // status=failed, for retry-success path
	SeedAgentRunAwaitingInputID = "agrn_01seede2e_runawti1" // status=awaiting_input, for continue-success path

	// Audit / Observability (seeded in 0014_e2e_extras.sql)
	SeedAuditEventID = "adev_01seedauditevent02" // event #2 has metadata populated
	// adev_01seedsearchtgt01 carries a distinctive resource_id + request_id for
	// the search ('q') tests. Searching either value must return that event.
	SeedAuditEventSearchResourceID = "it_01seedauditsrchtgt"
	SeedAuditEventSearchRequestID  = "rqlog_01seedauditsrchrq"
	// actor-or-target scope cohort (filter to it via the resource_ids below).
	// Caller is the seed account; each event covers one actor/target quadrant.
	SeedAuditScopeActorID       = "adev_01seedscopeactor"   // account_id=seed, target=customer
	SeedAuditScopeActorRes      = "it_01seedauditscopeac"   // its resource_id
	SeedAuditScopeTargetID      = "adev_01seedscopetarget"  // account_id=customer, target=seed
	SeedAuditScopeTargetRes     = "it_01seedauditscopetg"   // its resource_id
	SeedAuditScopeBothID        = "adev_01seedscopeboth00"  // account_id=seed, target=seed
	SeedAuditScopeBothRes       = "it_01seedauditscopebt"   // its resource_id
	SeedAuditScopeNeitherID     = "adev_01seedscopeneither" // account_id=child, target=customer (out of scope)
	SeedAuditScopeNeitherRes    = "it_01seedauditscopenn"   // its resource_id
	SeedInventoryChangeLogID    = "ivcl_01seedwss000000000" // seeded in 0007_items.sql, enriched in 0014_e2e_extras.sql
	SeedRequestLogErrorID       = "rqlog_01seedreqlog4_000" // has error_code=validation_failed for filter tests
	SeedRequestLogQueryParamsID = "rqlog_01seedreqlog5_000" // has query_json populated for include=query_params tests
	// referrer set on SeedReqLogInfraUserID (rqlog_01infrauser00) — only seed row
	// that populates the otherwise-always-null referrer field.
	SeedRequestLogReferrerValue = "https://dashboard.augno.com/inbox"
	// Audit event that populates source_ip + idempotency_key_id (NULL on all other
	// seed rows). The joined idempotency_key surfaces as the key string.
	SeedAuditEventWithSourceIPID = "adev_01seedsrcipkey0"
	SeedAuditEventSourceIP       = "198.51.100.42"
	SeedAuditEventIdempotencyKey = "e2e-seed-idempotency-key-01"

	// Tenant B (seeded in 0015_tenant_b_e2e.sql) — used for tenant isolation tests
	SeedTenantBAccountID = "ac_tenant2_e2e_isolati"
	SeedTenantBAPIKey    = "aug_sk_prod_TenantBKeyForE2eTests1_TenantBSecretForE2eIsolationTestingPurpose12didR71"
)

// pathParamSeeds maps path parameter names to seed IDs.
// Used to substitute path parameters in OpenAPI paths like /v1/catalog/properties/{property_id}/attributes.
var pathParamSeeds = map[string]string{
	"property_id":        SeedPropertyID,
	"product_line_id":    SeedProductLineID,
	"category_id":        SeedItemCategoryID, // PUT change-item-category (/items/{id}/category/{category_id})
	"attribute_id":       SeedAttributeID,    // PUT add-item-attribute (/items/{id}/attributes/{attribute_id})
	"item_category_id":   SeedItemCategoryID,
	"unit_group_id":      SeedUnitGroupID,
	"department_id":      SeedDepartmentID,
	"customer_id":        SeedCustomerAccountID,
	"sales_order_id":     SeedSalesOrderID,
	"shipment_id":        SeedShipmentID,
	"pick_id":            SeedPickID,
	"invoice_id":         SeedInvoiceID,
	"account_group_id":   SeedCustomerGroupID,
	"location_id":        SeedLocationID,
	"account_id":         SeedCustomerAccountID,
	"carrier_id":         SeedCarrierID,
	"item_id":            SeedLsnItemID,
	"production_step_id": SeedProductionStepID,
	"supplier_id":        SeedSupplierAccountID,
	"vendor_account_id":  SeedAccountID, // selling company's account for customer tenancy
	"agent_id":           SeedAgentConfigID,
	"line_id":            SeedSalesOrderLineID,
	"receiving_order_id": SeedReceivingOrderID,
	"target_id":          SeedSalesTargetID,
	"review_id":          SeedHubspotCompanyReviewID,
	// session_id is excluded from test discovery (excludedPaths in spec.go)
}

// pathSpecificIDSeeds resolves the generic {id} param based on the path prefix.
// The generic "id" param means different things depending on the endpoint.
var pathSpecificParamSeeds = map[string]map[string]string{
	"/v1/operations/production-schedules/": {
		"line_id": SeedProductionScheduleLineID,
	},
}

// pathSpecificParamSeed returns the longest-prefix match for a named path param that
// means different things on different routes — {line_id} is a sales-order line on one
// route and a schedule line on another.
func pathSpecificParamSeed(path, param string) (string, bool) {
	best := ""
	bestLen := 0
	for prefix, params := range pathSpecificParamSeeds {
		if !strings.HasPrefix(path, prefix) || len(prefix) <= bestLen {
			continue
		}
		if val, ok := params[param]; ok {
			best = val
			bestLen = len(prefix)
		}
	}
	return best, bestLen > 0
}

var pathSpecificIDSeeds = map[string]string{
	"/v1/catalog/catalog/product-lines/":      SeedProductLineID,
	"/v1/operations/production-runs/":         SeedProductionRunID,
	"/v1/operations/scanning-stations/":       SeedScanningStationID,
	"/v1/operations/machine-downtime-events/": SeedMachineDowntimeEventID,
	"/v1/operations/production-schedules/":    SeedProductionScheduleID,
	"/v1/operations/demand-overrides/":        SeedDemandOverrideID,
	"/v1/sales/account-users/":                SeedAccountUserID,
	"/v1/sales/priorities/":                   SeedPriorityID,

	// PATCH endpoint seeds
	"/v1/catalog/item-categories/":                                       SeedItemCategoryID,
	"/v1/catalog/items/":                                                 SeedItemID,
	"/v1/catalog/product-lines/":                                         SeedProductLineID,
	"/v1/catalog/product-types/":                                         SeedProductTypeID,
	"/v1/catalog/products/":                                              SeedProductID,
	"/v1/catalog/properties/{property_id}/attributes/":                   SeedAttributeID,
	"/v1/catalog/properties/":                                            SeedPropertyID,
	"/v1/catalog/unit-groups/{unit_group_id}/units/":                     SeedUnitGroupUnitID,
	"/v1/catalog/unit-groups/":                                           SeedUnitGroupID,
	"/v1/catalog/units/":                                                 SeedUnitID,
	"/v1/settings/properties/":                                           SeedSysPropertyID,
	"/v1/finance/invoices/":                                              SeedInvoiceID,
	"/v1/finance/payment-terms/":                                         SeedPaymentTermID,
	"/v1/finance/settlements/":                                           SeedSettlementID,
	"/v1/finance/transaction-allocations/":                               SeedTransactionAllocationID,
	"/v1/finance/transactions/":                                          SeedTransactionID,
	"/v1/identity/account-users/":                                        SeedAccountUserID,
	"/v1/identity/accounts/":                                             SeedAccountID,
	"/v1/settings/integrations/hubspot/sync/":                            SeedHubspotSyncJobID, // longer prefix wins over /v1/settings/integrations/
	"/v1/settings/integrations/":                                         SeedAccountIntegrationID,
	"/v1/identity/roles/":                                                SeedSalesRepRoleID, // account-owned so owner.account include populates
	"/v1/identity/users/":                                                SeedUserID,
	"/v1/operations/carriers/{carrier_id}/service-levels/":               SeedServiceLevelID,
	"/v1/operations/batches/":                                            SeedBatchID,
	"/v1/operations/carriers/":                                           SeedCarrierID,
	"/v1/operations/dc-locations/":                                       SeedDCLocationID,
	"/v1/operations/departments/":                                        SeedDepartmentID,
	"/v1/operations/machines/":                                           SeedMachineID,
	"/v1/catalog/materials/":                                             SeedMaterialID,
	"/v1/catalog/parts/":                                                 SeedPartID,
	"/v1/operations/picks/{pick_id}/lines/":                              SeedPickLineID,
	"/v1/operations/picks/":                                              SeedPickID,
	"/v1/operations/production-steps/{production_step_id}/consumptions/": SeedConsumptionID,
	"/v1/operations/production-steps/{production_step_id}/productions/":  SeedProductionID,
	"/v1/operations/production-steps/{id}":                               SeedSewLargeProductionStepID,
	"/v1/operations/production-steps/":                                   SeedProductionStepID,
	"/v1/operations/quantities/":                                         SeedQuantityID,
	"/v1/operations/rates/":                                              SeedRateID,
	"/v1/operations/shipments/{shipment_id}/lines/":                      SeedShipmentLineID,
	"/v1/operations/shipments/":                                          SeedShipmentID,
	"/v1/operations/deliveries/":                                         SeedDeliveryID,
	"/v1/operations/shipping-cases/":                                     SeedShippingCaseID,
	"/v1/operations/shipping-terms/":                                     SeedCustomShippingTermID, // account-owned so owner.account include populates
	"/v1/operations/locations/":                                          SeedLocationID,
	"/v1/sales/account-groups/":                                          SeedCustomerGroupID,
	"/v1/sales/account-prices/":                                          SeedAccountPriceID,
	"/v1/sales/addresses/":                                               SeedAddressID,
	"/v1/sales/order-discounts/":                                         SeedOrderDiscountID,
	"/v1/sales/product-line-access/":                                     SeedProductLineAccessID,
	"/v1/sales/registration-flows/":                                      SeedRegistrationFlowID,
	"/v1/sales/sales-orders/":                                            SeedSalesOrderID,
	"/v1/sales/volume-discounts/":                                        SeedVolumeDiscountID,

	"/v1/ai/agents/":                        SeedCustomAgentDefinitionID,
	"/v1/ai/memories/":                      SeedAgentMemoryID,
	"/v1/ai/runs/":                          SeedAgentRunID,
	"/v1/auth/api-keys/":                    SeedAPIKeyID,
	"/v1/core/audit-events/":                SeedAuditEventID,
	"/v1/core/email-logs/":                  SeedEmailLogID1,
	"/v1/core/jobs/":                        SeedJobID,
	"/v1/core/sandboxes/":                   SeedSandboxID,
	"/v1/operations/inventory-change-logs/": SeedInventoryChangeLogID,
	"/v1/operations/receiving-orders/{receiving_order_id}/lines/": SeedReceivingOrderLineID,
	"/v1/operations/receiving-orders/":                            SeedReceivingOrderID,
	"/v1/operations/suppliers/{supplier_id}/materials/":           SeedMaterialID,
	"/v1/operations/suppliers/":                                   SeedSupplierAccountID,
	"/v1/messaging/email-inboxes/":                                SeedEmailInboxID,
	"/v1/messaging/email-domains/":                                SeedEmailDomainID,
	"/v1/operations/location-types/":                              "building", // static type row; {id} accepts the code
	"/v1/sales/account-statuses/":                                 SeedAccountStatusID,
	"/v1/sales/accounts/{account_id}/territories/":                SeedTerritoryID,
	"/v1/sales/accounts/":                                         SeedCustomerAccountID,
	"/v1/sales/customers/":                                        SeedCustomerAccountID,

	// NOTE: purchase-orders, registration-sessions, and customer-accounts/tenancy
	// are excluded from test discovery in spec.go (excludedPaths).
}

// nullableFieldSeeds maps nullable-clear field names to test values that can be
// safely written and then cleared. Fields not in this map are skipped in the
// data-driven nullable test (covered by per-resource tests instead).
var nullableFieldSeeds = map[string]string{
	// Customer defaults
	"default_carrier_id":       SeedCarrierID,
	"default_payment_term_id":  SeedPaymentTermID,
	"default_shipping_term_id": SeedShippingTermID,
	"customer_type_group_id":   SeedCustomerGroupID,

	// Common reference IDs used across multiple endpoints
	"carrier_id":       SeedCarrierID,
	"payment_term_id":  SeedPaymentTermID,
	"shipping_term_id": SeedShippingTermID,
	// order_discount_id is intentionally excluded: the generic nullable-clear
	// test clears it on the shared seed sales order (SeedSalesOrderID), which
	// races with TestIncludes_PopulateNestedResources/retrieve-sales-order/order_discount
	// running in parallel and reading that same order. The include test is the
	// primary coverage for order_discount being populated on a sales order.
	//
	// Fields that surface as expandable sub-resources in the GET response
	// (e.g. role_id → role, default_sales_rep_id → defaults.sales_rep,
	// parent_id → parent) are excluded: the generic restore reads nil from
	// the response (the flat ID field doesn't exist) and leaves the field
	// cleared, racing with include tests that expect the seed value. These
	// are covered by per-resource CRUD tests.
	"scanning_station_id": SeedScanningStationID,
	"product_line_id":     SeedProductLineID,

	// String fields
	"email": "e2e-nullable@test.augno.com",
	"phone": "555-000-9999",
	"url":   "https://e2e-nullable.test.augno.com",
	"note":  "e2e nullable test note",

	// Clearable text fields (generic PATCH test)
	"street_line_2": "Suite 100",
	// description and notes are intentionally excluded: the product PATCH
	// cascades both to the item table via ProductUpdateItem, but the product
	// GET response surfaces them on the expandable item sub-resource (not at
	// the top level), so the generic restore reads nil and leaves the item's
	// values dirty. Per-resource CRUD tests cover nullable-clear for these.

	// bill_to_address_id and ship_to_address_id are intentionally excluded: the
	// generic nullable-clear test clears them on shared seed customers, suppliers,
	// and orders while TestIncludes_PopulateNestedResources reads those same
	// resources in parallel. Per-resource CRUD tests cover clearing these fields.
}
