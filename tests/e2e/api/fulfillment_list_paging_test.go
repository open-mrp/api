//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shipment and delivery lists choose each page from one of several reads (a list-order index,
// a status's rows, a set of suppliers' orders), each under the same keyset. A wrong comparison
// repeats or skips rows only when that read pages, so each is paged forward and back here.

// pagesBackToTheFirstPage reads the first one-row page, the next, and the previous of that, and
// requires the previous to be the first again.
func pagesBackToTheFirstPage(t *testing.T, path string, params url.Values) {
	t.Helper()
	merged := url.Values{"limit": {"1"}}
	for k, vs := range params {
		merged[k] = vs
	}

	first, status, err := apiClient.GetList(path, merged)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, first.Data, 1, "%s %v should hold at least two rows", path, params)
	require.True(t, first.PageInfo.HasNextPage, "%s %v should have a second page", path, params)

	second, status, err := apiClient.GetListFromPageURL(first.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, second.Data, 1)
	assert.NotEqual(t, DataItemField(first.Data[0], "id"), DataItemField(second.Data[0], "id"), "the next page repeats the first")
	require.True(t, second.PageInfo.HasPrevPage)

	back, status, err := apiClient.GetListFromPageURL(second.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, back.Data, 1)
	assert.Equal(t, DataItemField(first.Data[0], "id"), DataItemField(back.Data[0], "id"),
		"paging back from the second page must return the first")
}

// Each case pages two packed shipments of the test's own, found by a note only they carry: a search is
// residual to the key the case walks. The whole list is no fixture, since a seeded shipment dated years
// ahead heads it and every shipment a parallel test packs lands between that one and the next.
func TestShipmentsList_PagesBothWays(t *testing.T) {
	t.Parallel()

	note := uniqueName("e2e-shipment-paging")
	for range 2 {
		id := jsonField(packedShipment(t), "id")
		status, body, err := apiClient.Patch(shipmentsPath+"/"+id, map[string]any{"note": note}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	for name, params := range map[string]url.Values{
		"unfiltered": {"q": {note}},
		"status":     {"status": {"packed"}, "q": {note}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pagesBackToTheFirstPage(t, shipmentsPath, params)
		})
	}
}

// SHP-001's two seeded lines both have "seedpck" in their ids, so the searched list pages.
func TestShipmentLines_PagesBothWaysUnderASearch(t *testing.T) {
	t.Parallel()

	pagesBackToTheFirstPage(t, shipmentsPath+"/sh_01k0a87w33emw8pmkz1mf86cg1/lines", url.Values{"q": {"seedpck"}})
}

// A shipment matches several customers when any of them bought it.
func TestShipmentsList_FiltersBySeveralCustomers(t *testing.T) {
	t.Parallel()

	assert.Contains(t, shipmentIDsFiltered(t, url.Values{"customer_ids": {"ac_01nosuchcustomer000", SeedCustomerAccountID}}),
		seedPackedShipmentID)
}

// Two child-table filters both have to hold: one narrows the read, the other is checked per row.
func TestShipmentsList_ItemAndProductLineCompose(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, shipmentIDsFiltered(t, url.Values{"item_ids": {SeedItemID}, "product_line_ids": {SeedProductLineID}}),
		"the seeded item is in the seeded product line")
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"item_ids": {SeedItemID}, "product_line_ids": {"pdln_01nosuchline00000"}}),
		"an unknown product line must narrow the item's shipments to nothing")
}

// A group and a sales rep both narrow, and an unknown one of either empties the list.
func TestShipmentsList_CustomerGroupAndSalesRepCompose(t *testing.T) {
	t.Parallel()

	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {SeedCustomerGroupID}, "sales_rep_ids": {"acus_nosuchsalesrep00"}}))
	assert.Empty(t, shipmentIDsFiltered(t, url.Values{"customer_group_ids": {"acgp_01nosuchgroup0000"}, "sales_rep_ids": {SeedAccountUserID}}))
}

// The seeded deliveries: DLV-001 receives PO-001 from supplier 0 with a line for the seeded yarn,
// DLV-002 receives PO-002 from supplier 1 and has no lines.
const (
	seedDeliveryOneID = "dv_01seeddelivery1_0000"
	seedDeliveryTwoID = "dv_01seeddelivery2_0000"
	seedSupplierOneID = "ac_01seedsupplier_acct0"
	seedSupplierTwoID = "ac_01seedsupplier_acct1"
	seedDeliveredItem = "it_01seedyrn1item00000"
)

func TestDeliveries_SupplierFilterNarrowsToTheirOrders(t *testing.T) {
	t.Parallel()

	one := listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "supplier_ids": {seedSupplierOneID}})
	assert.Contains(t, one, seedDeliveryOneID)
	assert.NotContains(t, one, seedDeliveryTwoID, "another supplier's delivery must not appear")

	both := listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "supplier_ids": {seedSupplierOneID, seedSupplierTwoID}})
	assert.Contains(t, both, seedDeliveryOneID)
	assert.Contains(t, both, seedDeliveryTwoID)
}

func TestDeliveries_ItemFilterMatchesDeliveredLines(t *testing.T) {
	t.Parallel()

	ids := listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "item_ids": {seedDeliveredItem}})
	assert.Contains(t, ids, seedDeliveryOneID)
	assert.NotContains(t, ids, seedDeliveryTwoID, "a delivery with no line for the item must not appear")
}

// Search reaches the delivery's number and its purchase order's.
func TestDeliveries_SearchMatchesDeliveryAndOrderNumbers(t *testing.T) {
	t.Parallel()

	assert.Contains(t, listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "q": {"DLV-001"}}), seedDeliveryOneID)
	assert.Contains(t, listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "q": {"PO-002"}}), seedDeliveryTwoID)
	assert.Empty(t, listIDs(t, deliveriesPath, url.Values{"status": {"all"}, "q": {"zzz-no-such-delivery-zzz"}}))
}

// The two seeded deliveries are the account's only ones with these suppliers, so each case holds two.
func TestDeliveries_PagesBothWays(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]url.Values{
		"all":       {"status": {"all"}},
		"suppliers": {"status": {"all"}, "supplier_ids": {seedSupplierOneID, seedSupplierTwoID}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pagesBackToTheFirstPage(t, deliveriesPath, params)
		})
	}
}
