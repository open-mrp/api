//go:build e2e

package api_test

// Seed data constants from shared/db/seed/ and tools/apidocs/httpie_seed_data.go.
// These are IDs and values known to exist after seeding.
const (
	SeedAPIKey    = "aug_sk_prod_u6Xh5ZpaUruMAU12EPAs4z_rSA4zJM5NbRqAtalvXMoRWOUPohFKJtX7ZUFUOp36IVwdiUCZu"
	SeedAccountID = "ac_01k0a5smf9ekb8rqg12555zjqa"
	SeedUserID    = "us_1wjfmmbwg8l7"

	// Customers
	SeedCustomerAccountID  = "ac_01k09wm2fgevdsc344gpbcj30f"
	SeedCustomerName       = "Global Manufacturing Solutions"
	SeedCustomerGroupID    = "acgp_01k0a413mjeth8pe1g70t0thax"
	SeedCustomerGroupName  = "DME"
	SeedAccountRelationID  = "acre_01seedcustomer00000"
	SeedRegistrationFlowID = "mock-registration-flow"

	// Customer portal API key (owned by customer account, targets vendor account)
	SeedCustomerAPIKey = "aug_sk_prod_CustPortalE2eTestKey1_CustomerPortalE2eTestSecretValueForAuthTestingPurpose12345efS0Og"
	SeedAddressID          = "ad_01k09wnac0e1ar211e0sy0ba4g"

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
	SeedMaterialID       = "ml_01seedyrn1mat000000"
	SeedMaterialItemID   = "it_01seedyrn1item00000"
	SeedPartID           = "pt_01seedlknpart000000"
	SeedPartItemID       = "it_01seedlknitem000000"
	SeedConsumptionID    = "cp_01seedcons_kl_yarn1"
	SeedProductionID     = "pn_01seedprod_knitlg00"

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
	SeedAPIKeyID       = "apky_pajbskcck3cabxajdh8h8"
	SeedAccountUserID  = "acus_s83fjhyfmqen"

	// Priorities
	SeedPriorityID   = "pi_01seednormal0000000000"
	SeedPriorityCode = "normal"
	SeedPriorityName = "Normal"

	// Sandboxes
	SeedSandboxAccountID = "ac_sandbox_01k0a5smf9ekb8rqg12555zjqb"

	// Payment/Shipping
	SeedPaymentTermID        = "pytm_01seednet3000000"
	SeedDefaultPaymentTermID = "pytm_01seeddefault00000"
	SeedShippingTermID       = "prepaid_billed"
	SeedCarrierID            = "delivery"

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

	// Agents (seeded in agent-service 00003_seed_e2e_data.sql)
	SeedAgentDefinitionID = "agdf_01seede2e_orderbot0"
	SeedAgentConfigID     = "agcf_01seede2e_ordercfg0"
	SeedAgentMemoryID     = "agmm_01seede2e_memory01"

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
	"unitGroupId":        SeedUnitGroupID,
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
	"production_step_id": SeedProductionStepID,
	"pickId":             SeedPickID,
	"supplier_id":        SeedSupplierAccountID,
	"vendor_account_id":  SeedAccountID, // selling company's account for customer tenancy
	"agent_id":           SeedAgentConfigID,
	"lineId":             SeedSalesOrderLineID,
	"receivingOrderId":   SeedReceivingOrderID,
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
	"/v1/catalog/item-categories/":                     SeedItemCategoryID,
	"/v1/catalog/items/":                               SeedItemID,
	"/v1/catalog/product-lines/":                       SeedProductLineID,
	"/v1/catalog/product-types/":                       SeedProductTypeID,
	"/v1/catalog/products/":                            SeedItemID,
	"/v1/catalog/properties/{property_id}/attributes/": SeedAttributeID,
	"/v1/catalog/properties/":                          SeedPropertyID,
	"/v1/catalog/unit-groups/{unitGroupId}/units/":     SeedUnitGroupUnitID,
	"/v1/catalog/unit-groups/":                         SeedUnitGroupID,
	"/v1/catalog/units/":                               SeedUnitID,
	"/v1/core/sys-properties/":                         SeedSysPropertyID,
	"/v1/finance/invoices/":                            SeedInvoiceID,
	"/v1/finance/payment-terms/":                       SeedPaymentTermID,
	"/v1/finance/settlements/":                         SeedSettlementID,
	"/v1/finance/transaction-allocations/":             SeedTransactionAllocationID,
	"/v1/finance/transactions/":                        SeedTransactionID,
	"/v1/identity/account-users/":                      SeedAccountUserID,
	"/v1/identity/accounts/":                           SeedAccountID,
	"/v1/identity/roles/":                              SeedAdminRoleID,
	"/v1/identity/users/":                              SeedUserID,
	"/v1/operations/carriers/":                         SeedCarrierID,
	"/v1/operations/dc-locations/":                     SeedDCLocationID,
	"/v1/operations/departments/":                      SeedDepartmentID,
	"/v1/operations/machines/":                         SeedMachineID,
	"/v1/operations/materials/":                        SeedMaterialItemID,
	"/v1/operations/parts/":                            SeedPartItemID,
	"/v1/operations/picks/{pickId}/lines/":             SeedPickLineID,
	"/v1/operations/picks/":                            SeedPickID,
	"/v1/operations/production-steps/{production_step_id}/consumptions/": SeedConsumptionID,
	"/v1/operations/production-steps/{production_step_id}/productions/":  SeedProductionID,
	"/v1/operations/production-steps/":                                   SeedProductionStepID,
	"/v1/operations/quantities/":                                         SeedQuantityID,
	"/v1/operations/rates/":                                              SeedRateID,
	"/v1/operations/shipments/{shipment_id}/lines/":                      SeedShipmentLineID,
	"/v1/operations/shipments/":                                          SeedShipmentID,
	"/v1/operations/shipping-cases/":                                     SeedShippingCaseID,
	"/v1/operations/shipping-terms/":                                     SeedShippingTermID,
	"/v1/operations/locations/":                                          SeedLocationID,
	"/v1/sales/account-groups/":                                          SeedCustomerGroupID,
	"/v1/sales/account-prices/":                                          SeedAccountPriceID,
	"/v1/sales/addresses/":                                               SeedAddressID,
	"/v1/sales/order-discounts/":                                         SeedOrderDiscountID,
	"/v1/sales/product-line-access/":                                     SeedProductLineAccessID,
	"/v1/sales/registration-flows/":                                      SeedRegistrationFlowID,
	"/v1/sales/sales-orders/":                                            SeedSalesOrderID,
	"/v1/sales/volume-discounts/":                                        SeedVolumeDiscountID,

	"/v1/ai/agents/":       SeedAgentDefinitionID,
	"/v1/ai/memories/":     SeedAgentMemoryID,
	"/v1/core/email-logs/": SeedEmailLogID1,
	"/v1/operations/receiving-orders/{receivingOrderId}/lines/": SeedReceivingOrderLineID,
	"/v1/operations/receiving-orders/":                          SeedReceivingOrderID,
	"/v1/operations/suppliers/{supplier_id}/materials/":         SeedMaterialItemID,
	"/v1/operations/suppliers/":                                 SeedSupplierAccountID,
	"/v1/sales/accounts/{account_id}/territories/":              SeedTerritoryID,
	"/v1/sales/accounts/":                                       SeedCustomerAccountID,
	"/v1/sales/customers/":                                      SeedCustomerAccountID,

	// NOTE: purchase-orders, registration-sessions, and customer-accounts/tenancy
	// are excluded from test discovery in spec.go (excludedPaths).
}

// nullableFieldSeeds maps nullable-clear field names to test values that can be
// safely written and then cleared. Fields not in this map are skipped in the
// data-driven nullable test (covered by per-resource tests instead).
var nullableFieldSeeds = map[string]string{
	// Customer defaults
	"default_carrier_id":        SeedCarrierID,
	"default_payment_term_id":   SeedPaymentTermID,
	"default_shipping_term_id":  SeedShippingTermID,
	"default_sales_rep_user_id": SeedUserID,
	"customer_type_group_id":    SeedCustomerGroupID,

	// Common reference IDs used across multiple endpoints
	"carrier_id":          SeedCarrierID,
	"payment_term_id":     SeedPaymentTermID,
	"shipping_term_id":    SeedShippingTermID,
	"sales_rep_id":        SeedAccountUserID,
	"order_discount_id":   SeedOrderDiscountID,
	"role_id":             SeedAdminRoleID,
	"department_id":       SeedDepartmentID,
	"scanning_station_id": SeedScanningStationID,
	"product_line_id":     SeedProductLineID,

	// String fields
	"email": "e2e-nullable@test.augno.com",
	"phone": "555-000-9999",
	"url":   "https://e2e-nullable.test.augno.com",
	"note":  "e2e nullable test note",
}
