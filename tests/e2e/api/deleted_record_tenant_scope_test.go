//go:build e2e

package api_test

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A deleted record answers 410 only to the account it was deleted from; to any other tenant its id reads, updates and deletes as one that never existed.

type deletedScopeTarget struct {
	path string
	// tenantBPath is where tenant B names the same id, defaulting to path; a nested record goes under tenant B's own parent so the parent check does not answer first.
	tenantBPath string
	// owner and tenantB default to apiClient and getTenantBClient().
	owner, tenantB *Client
}

type deletedScopeCase struct {
	name  string
	setup func(t *testing.T) deletedScopeTarget
	noGet bool
	// patch is a body the update endpoint accepts; nil when the resource has no PATCH.
	patch map[string]any
	// deleteStatus is the first delete's success status; 0 means 200.
	deleteStatus int
}

func runDeletedScopeCases(t *testing.T, cases []deletedScopeCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := tc.setup(t)
			owner := cmp.Or(target.owner, apiClient)
			tenantB := cmp.Or(target.tenantB, getTenantBClient())
			probe := cmp.Or(target.tenantBPath, target.path)

			status, body, err := owner.Delete(target.path)
			require.NoError(t, err)
			requireStatus(t, cmp.Or(tc.deleteStatus, 200), status, body)

			if !tc.noGet {
				status, body, err = tenantB.GetListRaw(probe, nil)
				require.NoError(t, err)
				assert.Equal(t, 404, status, "GET as another tenant: %s", body)
			}
			if tc.patch != nil {
				status, body, err = tenantB.Patch(probe, tc.patch, newIdempotencyKey())
				require.NoError(t, err)
				assert.Equal(t, 404, status, "PATCH as another tenant: %s", body)
			}
			status, body, err = tenantB.Delete(probe)
			require.NoError(t, err)
			assert.Equal(t, 404, status, "DELETE as another tenant must not reveal the id existed: %s", body)

			status, body, err = owner.Delete(target.path)
			require.NoError(t, err)
			requireStatus(t, 410, status, body)
			requireErrorResponse(t, body, "resource_gone", "invalid_request_error")
		})
	}
}

func createdPath(t *testing.T, collection string, body map[string]any) deletedScopeTarget {
	t.Helper()
	return deletedScopeTarget{path: collection + "/" + jsonField(createAndCleanup(t, collection, body), "id")}
}

// createInTenantB creates a parent record in tenant B for the duration of the test and returns its id.
func createInTenantB(t *testing.T, collection string, body map[string]any) string {
	t.Helper()
	clientB := getTenantBClient()
	status, resp, err := clientB.Post(collection, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	id := jsonField(parseJSON(resp), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = clientB.Delete(collection + "/" + id) })
	return id
}

func TestDeletedRecordScope_Catalog(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "product",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, productsPath, validProductBody(uniqueName("e2e-dscope")))
			},
			patch: map[string]any{"description": "x"},
		},
		{
			name: "product line",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, productLinesPath, map[string]any{
					"name":              uniqueName("e2e-dscope-pl"),
					"unit_group_id":     SeedUnitGroupID,
					"commission_policy": "commission_applied",
					"freight_policy":    "billed_freight",
				})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-pl")},
		},
		{
			name: "part",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, partsPath, validPartBody(uniqueName("e2e-dscope")))
			},
			patch: map[string]any{"description": "x"},
		},
		{
			name: "material",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, materialsPath, validMaterialBody(uniqueName("e2e-dscope")))
			},
			patch: map[string]any{"description": "x"},
		},
		{
			name: "item category",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, itemCategoriesPath, map[string]any{
					"name":          uniqueName("e2e-dscope-ic"),
					"type":          "material_category",
					"unit_group_id": SeedUnitGroupID,
				})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-ic")},
		},
		{
			name: "property",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, propertiesPath, map[string]any{"name": uniqueName("e2e-dscope-prop")})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-prop")},
		},
		{
			name: "attribute",
			setup: func(t *testing.T) deletedScopeTarget {
				propertyID := jsonField(createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-dscope-attr")}), "id")
				attribute := createAndCleanup(t, attributesPath(propertyID), map[string]any{"value": uniqueName("e2e-dscope-attr")})
				propertyB := createInTenantB(t, propertiesPath, map[string]any{"name": uniqueName("e2e-dscope-attr-b")})
				return deletedScopeTarget{
					path:        attributePath(propertyID, jsonField(attribute, "id")),
					tenantBPath: attributePath(propertyB, jsonField(attribute, "id")),
				}
			},
			patch: map[string]any{"value": uniqueName("e2e-dscope-attr")},
		},
		{
			name: "unit",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, unitsPath, covCatalogUnitsCreateBody(uniqueName("e2e-dscope-unit"), uniqueName("ds")))
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-unit")},
		},
		{
			name: "unit group",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, unitGroupsPath, covCatalogUnitGroupsCreateBody(uniqueName("e2e-dscope-ug")))
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-ug")},
		},
		{
			name: "unit group unit",
			setup: func(t *testing.T) deletedScopeTarget {
				groupID := jsonField(createAndCleanup(t, unitGroupsPath, covCatalogUnitGroupsCreateBody(uniqueName("e2e-dscope-ugu"))), "id")
				unit := createAndCleanup(t, unitGroupUnitsPath(groupID), map[string]any{"unit_id": "un_01seeddozen00000000"})
				groupB := createInTenantB(t, unitGroupsPath, covCatalogUnitGroupsCreateBody(uniqueName("e2e-dscope-ugu-b")))
				return deletedScopeTarget{
					path:        covCatalogUnitGroupUnitPath(groupID, jsonField(unit, "id")),
					tenantBPath: covCatalogUnitGroupUnitPath(groupB, jsonField(unit, "id")),
				}
			},
			patch: map[string]any{"discount_fixed": 1.0},
		},
	})
}

func TestDeletedRecordScope_Sales(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "sales order",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: salesOrdersPath + "/" + createLifecycleOrder(t)}
			},
			patch: map[string]any{"note": "x"},
		},
		{
			name: "account price",
			setup: func(t *testing.T) deletedScopeTarget {
				lockPricingWrite(t)
				return deletedScopeTarget{path: accountPricesPath + "/" + jsonField(createAccountPrice(t, SeedCustomerAccountID, "32.00"), "id")}
			},
			patch: map[string]any{"rate": map[string]any{"value": "33.00", "numerator_unit_id": e2eCurrencyUnitID, "denominator_unit_id": SeedUnitID}},
		},
		{
			name: "volume discount",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: volumeDiscountsPath + "/" + jsonField(createVolumeDiscount(t, map[string]any{}), "id")}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-vd")},
		},
		{
			name: "order discount",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: orderDiscountsPath + "/" + jsonField(createOrderDiscount(t, map[string]any{}), "id")}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-od")},
		},
		{
			name: "account group",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: accountGroupsPath + "/" + newAccountGroup(t)}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-grp")},
		},
		{
			name: "account group product line access",
			setup: func(t *testing.T) deletedScopeTarget {
				groupID := newAccountGroup(t)
				grantAccess(t, accountGroupAccessPath, map[string]any{"account_group_id": groupID, "product_line_ids": []string{SeedProductLineID}})
				return deletedScopeTarget{path: accountGroupAccessPath + "/" + groupID}
			},
			patch: map[string]any{"product_line_ids": []string{SeedProductLineID}},
		},
		{
			name: "customer product line access",
			setup: func(t *testing.T) deletedScopeTarget {
				customerID := customerInGroup(t, SeedCustomerGroupID)
				grantAccess(t, customerAccessPath, map[string]any{"customer_id": customerID, "product_line_ids": []string{SeedProductLineID}})
				return deletedScopeTarget{path: customerAccessPath + "/" + customerID}
			},
			patch: map[string]any{"product_line_ids": []string{SeedProductLineID}},
		},
		{
			name: "registration flow",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: registrationFlowsPath + "/" + createRegistrationFlow(t, apiClient)}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-flow")},
		},
		{
			name: "payment term",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, paymentTermsPath, map[string]any{"name": uniqueName("e2e-dscope-pt")})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-pt")},
		},
	})
}

func TestDeletedRecordScope_Purchasing(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "purchase order",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: purchaseOrdersPath + "/" + jsonField(createPurchaseOrder(t, nil), "id")}
			},
			patch: map[string]any{"note": "x"},
		},
	})
}

func TestDeletedRecordScope_Shipping(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "carrier",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, carriersPath, map[string]any{"name": uniqueName("e2e-dscope-carrier"), "code": "will_call"})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-carrier")},
		},
		{
			name: "service level",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, serviceLevelsPath(SeedCarrierID), map[string]any{
					"name": uniqueName("e2e-dscope-sl"),
					"code": uniqueName("e2e-dscope-sl-code"),
				})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-sl")},
		},
		{
			name: "shipping term",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, shippingTermsPath, map[string]any{"name": uniqueName("e2e-dscope-st"), "type": "free_freight"})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-st")},
		},
	})
}

func TestDeletedRecordScope_Payments(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "transaction",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, transactionsPath, map[string]any{
					"customer_id":         SeedCustomerAccountID,
					"type":                "payment",
					"amount":              "1.00",
					"responsible_user_id": SeedAccountUserID,
				})
			},
			patch: map[string]any{"note": "x"},
		},
		{
			name: "settlement",
			setup: func(t *testing.T) deletedScopeTarget {
				settlementID, _ := newSettlement(t)
				return deletedScopeTarget{path: settlementsPath + "/" + settlementID}
			},
			patch: map[string]any{"note": "x"},
		},
		{
			name: "transaction allocation",
			setup: func(t *testing.T) deletedScopeTarget {
				_, allocationID := newSettlement(t)
				return deletedScopeTarget{path: financeAllocationsPath + "/" + allocationID}
			},
			noGet: true,
			patch: map[string]any{"amount": "0.50"},
		},
	})
}

func TestDeletedRecordScope_Production(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "production run",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: productionRunPath(jsonField(createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID}), "id"))}
			},
			patch: map[string]any{"number": uniqueName("e2e-dscope-run")},
		},
		{
			name: "production step",
			setup: func(t *testing.T) deletedScopeTarget {
				stepID := createLinkedStep(t, "e2e-dscope-step", createPartItem(t, "e2e-dscope-made"), createPartItem(t, "e2e-dscope-used"))
				return deletedScopeTarget{path: productionStepsPath + "/" + stepID}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-step")},
		},
		{
			// Tenant B has no production step to name it under, so its 404 comes from the step check.
			name: "consumption",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: consumptionsPath() + "/" + newConsumption(t)}
			},
			patch: map[string]any{"quantity_value": "2"},
		},
		{
			name: "batch",
			setup: func(t *testing.T) deletedScopeTarget {
				runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
				batches := runBatches(t, runID, "run")
				require.Len(t, batches, 1)
				return deletedScopeTarget{path: batchesPath + "/" + jsonField(batches[0], "id")}
			},
			noGet: true,
		},
		{
			name: "machine",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, machinesPath, map[string]any{
					"name":          uniqueName("e2e-dscope-mch"),
					"serial_number": uniqueName("e2e-dscope-sn"),
					"department_id": SeedDepartmentID,
				})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-mch")},
		},
		{
			name: "department",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, departmentsPath, map[string]any{"name": uniqueName("e2e-dscope-dep")})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-dep")},
		},
		{
			name: "scanning station",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, scanningStationsPath, covOperationsScanningStationsCreateBody(uniqueName("e2e-dscope-ss")))
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-ss")},
		},
		{
			name: "location",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, locationsPath, map[string]any{"name": uniqueName("e2e-dscope-loc"), "type": "building"})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-loc")},
		},
	})
}

func TestDeletedRecordScope_Account(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "address",
			setup: func(t *testing.T) deletedScopeTarget {
				return deletedScopeTarget{path: addressesPath + "/" + createE2EAddress(t, uniqueName("e2e-dscope-addr"))}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-addr")},
		},
		{
			name: "role",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, rolesPath, map[string]any{"name": uniqueName("e2e-dscope-role")})
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-role")},
		},
		{
			name: "sandbox",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, sandboxesPath, map[string]any{"name": uniqueName("e2e-dscope-sb"), "mode": "blank"})
			},
			deleteStatus: 202,
		},
	})
}

func TestDeletedRecordScope_Agents(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "custom agent",
			setup: func(t *testing.T) deletedScopeTarget {
				return createdPath(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("dscope"))
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-agent")},
		},
	})
}

// Groups are a user's chat rosters, so both sides sign in as the seed user, who is a member of both tenants.
func TestDeletedRecordScope_Messaging(t *testing.T) {
	t.Parallel()
	runDeletedScopeCases(t, []deletedScopeCase{
		{
			name: "messaging group",
			setup: func(t *testing.T) deletedScopeTarget {
				owner := chatUserClient(t)
				group := createMessagingGroup(t, owner, uniqueName("e2e-dscope-mg"), nil, nil)
				return deletedScopeTarget{
					path:    messagingGroupsPath + "/" + jsonField(group, "id"),
					owner:   owner,
					tenantB: loginAsUser(t, seedUserEmail, seedUserPassword, SeedTenantBAccountID),
				}
			},
			patch: map[string]any{"name": uniqueName("e2e-dscope-mg")},
		},
	})
}
