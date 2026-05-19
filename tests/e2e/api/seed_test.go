//go:build e2e

package api_test

// Seed data constants from shared/db/seed/ and tools/apidocs/httpie_seed_data.go.
// These are IDs and values known to exist after seeding.
const (
	SeedAPIKey    = "aug_sk_prod_u6Xh5ZpaUruMAU12EPAs4z_rSA4zJM5NbRqAtalvXMoRWOUPohFKJtX7ZUFUOp36IVwdiUCZu"
	SeedAccountID = "ac_01k0a5smf9ekb8rqg12555zjqa"
	SeedUserID    = "us_1wjfmmbwg8l7"
	SeedUser2ID   = "us_6p7460uuwibz"

	// Customers
	SeedCustomerAccountID  = "ac_01k09wm2fgevdsc344gpbcj30f"
	SeedCustomerName       = "Global Manufacturing Solutions"
	SeedCustomerGroupID    = "acgp_01k0a413mjeth8pe1g70t0thax"
	SeedCustomerGroupName  = "DME"
	SeedAccountRelationID  = "acre_01seedcustomer00000"
	SeedRegistrationFlowID = "mock-registration-flow"

	// Customer portal API key (owned by customer account, targets vendor account)
	SeedCustomerAPIKey = "aug_sk_prod_CustPortalE2eTestKey1_CustomerPortalE2eTestSecretValueForAuthTestingPurpose12345efS0Og"
	SeedAddressID      = "ad_01k09wnac0e1ar211e0sy0ba4g"

	// Catalog
	SeedProductLineID    = "pdln_01k0a735ype5e8nrhv1n5dhq1q"
	SeedProductLineName  = "Socks"
	SeedItemID           = "it_01k0a7100aeysrs9vxpeq14yxj"
	SeedItemSKU          = "SCK-001"
	SeedItemDescription  = "Small white sock"
	SeedItemCategoryID   = "itcg_01seedsocks000000"
	SeedItemCategoryName = "Socks"
	SeedProductID        = "pd_01k0a65nx2e2crfxrvryyxnmdh"
	SeedProductTypeID    = "pdtp_01seedsale00000000"
	SeedPropertyID       = "pp_01k0a7ntn1ez6aw8x850femxeh"
	SeedPropertyName     = "Color"
	SeedAttributeID      = "at_01seedbeige00000000"

	itemsPath = "/v1/catalog/items"

	// Units
	SeedUnitID          = "un_01seedpair000000000"
	SeedUnitGroupID     = "ungp_01k0a5ecy9edg9za40dnccw53n"
	SeedUnitGroupUnitID = "ungpun_01seedsocksea000"

	// Measures
	SeedRateID     = "rt_01seedwssunitval000"
	SeedQuantityID = "qu_01seediss_ln1_qty00"

	// Infrastructure
	SeedDepartmentID      = "dp_01k0a5r01yfx3sj1vy9qgv3dc0"
	SeedMachineID         = "mc_01k0a52fb6eqhtbx9hdxj3vvnh"
	SeedLocationID        = "sglc_01seedbuilding0000"
	SeedScanningStationID = "sgsn_01k0a8201zegarjfsjaw5n7yfv"

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
	SeedSalesOrderID         = "or_01k0a8bs2yejxbsvqhrx4drkq1"
	SeedSalesOrderLineID     = "orln_01seediss_ln1_0000"
	SeedShipmentID           = "sh_01k0a87w33emw8pmkz1mf86cg1"
	SeedShipmentLineID       = "shln_01seedshpln1_00000"
	SeedShippingCaseID       = "shcs_01seedshcase1_00000"
	SeedPickID               = "pk_01k0a5tsn7f7psgagr1732fxqa"
	SeedPickLineID           = "pkln_01seediss_ln1_0000"
	SeedReceivingOrderLineID = "rcln_01seedrecvln1_0000"
	SeedInvoiceID            = "iv_01k09wnac0e1ar211e0sy0ba4g"

	// Auth
	SeedAdminRoleID    = "rl_mtg88e6u6fbu"
	SeedSalesRepRoleID = "rl_hh6mrlkv08n8"
	SeedScannerRoleID  = "rl_scanner"
	SeedAPIKeyID       = "apky_pajbskcck3cabxajdh8h8"
	SeedAccountUserID  = "acus_s83fjhyfmqen"

	// Batches, sales targets, integrations (seeded in 0014_e2e_extras.sql)
	SeedBatchID              = "bt_01seedbatch1_0000000"
	SeedSalesTargetID        = "tg_01seedtarget1_000000"
	SeedAccountIntegrationID = "acin_01seedintegration1"

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

	// System
	SeedSysPropertyID = "sypp_01seedtxnumber000"

	// Finance (seeded in 0013_finance.sql)
	SeedDCLocationID            = "dclc_01seeddc_location0"
	SeedSettlementID            = "sl_01seedsettlement000"
	SeedTransactionID           = "tx_01seedtransaction00"
	SeedTransactionAllocationID = "txal_01seedtxalloc0000"

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

	// Agents (seeded in agent-service 00001_seed_e2e_data.sql)
	SeedAgentDefinitionID       = "agdf_01seede2e_orderbot0"
	SeedCustomAgentDefinitionID = "agdf_01seede2e_custom00"
	SeedAgentConfigID           = "agcf_01seede2e_ordercfg0"
	SeedAgentMemoryID           = "agmm_01seede2e_memory01"
	SeedAgentRunID              = "agrn_01seede2e_run00001" // run #1 has agent_definition + config (needed for include=definition.config)
	SeedAgentAlertID            = "agal_01seede2e_alert002" // alert #2 has agent_run_id + agent_action_id populated

	// Audit / Observability (seeded in 0014_e2e_extras.sql)
	SeedAuditEventID            = "adev_01seedauditevent02" // event #2 has metadata populated
	SeedInventoryChangeLogID    = "ivcl_01seedwss000000000" // seeded in 0007_items.sql, enriched in 0014_e2e_extras.sql
	SeedRequestLogErrorID       = "rqlog_01seedreqlog4_000" // has error_code=validation_failed for filter tests
	SeedRequestLogQueryParamsID = "rqlog_01seedreqlog5_000" // has query_json populated for include=query_params tests

	// Tenant B (seeded in 0015_tenant_b_e2e.sql) — used for tenant isolation tests
	SeedTenantBAccountID = "ac_tenant2_e2e_isolati"
	SeedTenantBAPIKey    = "aug_sk_prod_TenantBKeyForE2eTests1_TenantBSecretForE2eIsolationTestingPurpose12didR71"
)

// pathParamSeeds maps path parameter names to seed IDs.
// Used to substitute path parameters in OpenAPI paths like /v1/catalog/properties/{property_id}/attributes.
var pathParamSeeds = map[string]string{
	"property_id":        SeedPropertyID,
	"product_line_id":    SeedProductLineID,
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
	// session_id is excluded from test discovery (excludedPaths in spec.go)
}

// pathSpecificIDSeeds resolves the generic {id} param based on the path prefix.
// The generic "id" param means different things depending on the endpoint.
var pathSpecificIDSeeds = map[string]string{
	"/v1/catalog/catalog/product-lines/": SeedProductLineID,
	"/v1/operations/production-runs/":    SeedProductionRunID,
	"/v1/operations/scanning-stations/":  SeedScanningStationID,
	"/v1/sales/account-users/":           SeedAccountUserID,
	"/v1/sales/priorities/":              SeedPriorityID,

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
	"/v1/core/sys-properties/":                                           SeedSysPropertyID,
	"/v1/finance/invoices/":                                              SeedInvoiceID,
	"/v1/finance/payment-terms/":                                         SeedPaymentTermID,
	"/v1/finance/settlements/":                                           SeedSettlementID,
	"/v1/finance/transaction-allocations/":                               SeedTransactionAllocationID,
	"/v1/finance/transactions/":                                          SeedTransactionID,
	"/v1/identity/account-users/":                                        SeedAccountUserID,
	"/v1/identity/accounts/":                                             SeedAccountID,
	"/v1/identity/integrations/":                                         SeedAccountIntegrationID,
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
	"/v1/ai/alerts/":                        SeedAgentAlertID,
	"/v1/ai/memories/":                      SeedAgentMemoryID,
	"/v1/ai/runs/":                          SeedAgentRunID,
	"/v1/auth/api-keys/":                    SeedAPIKeyID,
	"/v1/core/audit-events/":                SeedAuditEventID,
	"/v1/core/email-logs/":                  SeedEmailLogID1,
	"/v1/core/sandboxes/":                   SeedSandboxID,
	"/v1/operations/inventory-change-logs/": SeedInventoryChangeLogID,
	"/v1/operations/receiving-orders/{receiving_order_id}/lines/": SeedReceivingOrderLineID,
	"/v1/operations/receiving-orders/":                            SeedReceivingOrderID,
	"/v1/operations/suppliers/{supplier_id}/materials/":           SeedMaterialID,
	"/v1/operations/suppliers/":                                   SeedSupplierAccountID,
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
	"default_sales_rep_id":     SeedAccountUserID,
	"customer_type_group_id":   SeedCustomerGroupID,

	// Common reference IDs used across multiple endpoints
	"carrier_id":       SeedCarrierID,
	"payment_term_id":  SeedPaymentTermID,
	"shipping_term_id": SeedShippingTermID,
	"sales_rep_id":     SeedAccountUserID,
	// order_discount_id is intentionally excluded: the generic nullable-clear
	// test clears it on the shared seed sales order (SeedSalesOrderID), which
	// races with TestIncludes_PopulateNestedResources/retrieve-sales-order/order_discount
	// running in parallel and reading that same order. The include test is the
	// primary coverage for order_discount being populated on a sales order.
	"role_id":             SeedAdminRoleID,
	"department_id":       SeedDepartmentID,
	"scanning_station_id": SeedScanningStationID,
	"product_line_id":     SeedProductLineID,

	// String fields
	"email": "e2e-nullable@test.augno.com",
	"phone": "555-000-9999",
	"url":   "https://e2e-nullable.test.augno.com",
	"note":  "e2e nullable test note",

	// Clearable text fields (generic PATCH test)
	"description":   "e2e nullable description",
	"notes":         "e2e nullable notes",
	"street_line_2": "Suite 100",

	// Reference IDs for location/customer clear tests
	"parent_id": "sglc_01seedknitting000",
	// bill_to_address_id and ship_to_address_id are intentionally excluded: the
	// generic nullable-clear test clears them on shared seed customers, suppliers,
	// and orders while TestIncludes_PopulateNestedResources reads those same
	// resources in parallel. Per-resource CRUD tests cover clearing these fields.
}
