//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Writes one record makes to another on its behalf (a ship stamping its order, an unissue tearing
// down a pick, a settlement recording its transactions) are audited on the record written.

// expectAuditFieldChange polls until an event on the resource records a change to field, and returns that change.
func expectAuditFieldChange(t *testing.T, resourceID, resourceType, action, field string) map[string]any {
	t.Helper()

	var change map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
			"resource_ids":   {resourceID},
			"resource_types": {resourceType},
			"actions":        {action},
			"include":        {"changes"},
			"limit":          {"100"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("unexpected status %d while querying audit events", status)
		}
		for _, item := range jsonArray(parseJSON(body), "data") {
			event, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if c, ok := changeForField(jsonListData(event, "changes"), field); ok {
				change = c
				return nil
			}
		}
		return fmt.Errorf("no %s audit event yet recording %s on %s %s", action, field, resourceType, resourceID)
	})
	return change
}

func shipmentRelatedID(t *testing.T, shipmentID, related string) string {
	t.Helper()
	shipment := readShipment(t, shipmentID, "related."+related)
	record := jsonObject(jsonObject(shipment, "related"), related)
	require.NotNil(t, record, "the shipment should name its %s: %v", related, shipment)
	return jsonField(record, "id")
}

func idsOf(items []any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			ids = append(ids, jsonField(m, "id"))
		}
	}
	return ids
}

// --- Shipping ---

// Shipping stamps the order fulfilled, gives its cases an SSCC, and writes the carrier's charge to the freight line.
func TestOrderCascadeAudit_ShipAuditsOrderCasesAndFreightLine(t *testing.T) {
	t.Parallel()

	sale := shipSaleToNewCustomer(t)
	orderID := shipmentRelatedID(t, sale.shipmentID, "sales_order")

	firstShip := expectAuditFieldChange(t, orderID, "sales_order", "update", "first_ship_at")
	assertNilField(t, firstShip, "old_value")
	assert.NotEmpty(t, jsonField(firstShip, "new_value"))
	status := expectAuditFieldChange(t, orderID, "sales_order", "update", "sales_order_status_code")
	assert.Equal(t, "fulfilled", jsonField(status, "new_value"), "the whole order shipped")

	caseIDs := shipmentCaseIDs(t, sale.shipmentID)
	require.NotEmpty(t, caseIDs)
	sscc := expectAuditFieldChange(t, caseIDs[0], "shipping_case", "update", "sscc")
	assertNilField(t, sscc, "old_value")
	assert.NotEmpty(t, jsonField(sscc, "new_value"))
	expectAuditFieldChange(t, caseIDs[0], "shipping_case", "update", "shipped_at")

	sku := orderedSKU(t, orderID)
	order := getSalesOrder(t, orderID, url.Values{"include": {"lines"}})
	var freightLineID string
	for _, raw := range jsonListData(order, "lines") {
		if line := raw.(map[string]any); jsonField(line, "product_sku") != sku {
			freightLineID = jsonField(line, "id")
		}
	}
	require.NotEmpty(t, freightLineID, "shipping leaves the order a freight line to carry the cost")
	expectAuditEvent(t, freightLineID, "sales_order_line", "create")
	cost := expectAuditFieldChange(t, freightLineID, "sales_order_line", "update", "unit_cost_value")
	assertNilField(t, cost, "old_value")
}

// Voiding hands the order back to issued and clears what shipping stamped on its cases.
func TestOrderCascadeAudit_VoidAuditsOrderAndCases(t *testing.T) {
	t.Parallel()

	sale := shipSaleToNewCustomer(t)
	orderID := shipmentRelatedID(t, sale.shipmentID, "sales_order")
	caseIDs := shipmentCaseIDs(t, sale.shipmentID)
	require.NotEmpty(t, caseIDs)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sale.shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsWithChanges(t, orderID, "sales_order", "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), "first_ship_at"); ok && c["new_value"] == nil {
				return nil
			}
		}
		return fmt.Errorf("no update yet clearing the order's first_ship_at")
	})
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsWithChanges(t, caseIDs[0], "shipping_case", "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), "shipped_at"); ok && c["new_value"] == nil {
				return nil
			}
		}
		return fmt.Errorf("no update yet clearing the case's shipped_at")
	})
}

func auditEventsWithChanges(t *testing.T, resourceID, resourceType, action string) []map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"resource_ids":   {resourceID},
		"resource_types": {resourceType},
		"actions":        {action},
		"include":        {"changes"},
		"limit":          {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var events []map[string]any
	for _, item := range jsonArray(parseJSON(body), "data") {
		if event, ok := item.(map[string]any); ok {
			events = append(events, event)
		}
	}
	return events
}

func TestOrderCascadeAudit_EmailingAnInvoiceAuditsTheSentFlag(t *testing.T) {
	t.Parallel()

	sale := shipSaleToNewCustomer(t)
	invoiceID := shipmentRelatedID(t, sale.shipmentID, "invoice")

	status, body, err := apiClient.Post(invoiceEmailRecordPath, map[string]any{"type": "invoice", "id": invoiceID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	sent := expectAuditFieldChange(t, invoiceID, "invoice", "update", "has_been_sent")
	assert.Equal(t, "false", jsonField(sent, "old_value"))
	assert.Equal(t, "true", jsonField(sent, "new_value"))
}

// --- Picks ---

func TestOrderCascadeAudit_UnissueAuditsThePickTeardown(t *testing.T) {
	t.Parallel()

	orderID := createLifecycleOrder(t)
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)
	lineIDs := idsOf(pickLines(t, pickID))
	require.NotEmpty(t, lineIDs)

	status, body = salesOrderAction(t, orderID, "unissue", false)
	requireStatus(t, 200, status, body)

	expectAuditEvent(t, pickID, "pick", "delete")
	for _, lineID := range lineIDs {
		expectAuditEvent(t, lineID, "pick_line", "delete")
	}
}

// A line added to an issued order opens a pick line for it, and deleting the line removes that pick line.
func TestOrderCascadeAudit_LineEditsOnAnIssuedOrderAuditPickLines(t *testing.T) {
	t.Parallel()

	orderID := createLifecycleOrder(t)
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)
	before := idsOf(pickLines(t, pickID))

	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  SeedProductID,
		"product_sku": uniqueName("E2E-AUDIT-PICK"),
		"quantity":    map[string]any{"value": "2", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	lineID := jsonField(parseJSON(body), "id")

	var added string
	for _, id := range idsOf(pickLines(t, pickID)) {
		if !slices.Contains(before, id) {
			added = id
		}
	}
	require.NotEmpty(t, added, "the new sale line gets a pick line")
	expectAuditEvent(t, added, "pick_line", "create")

	status, body, err = apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + lineID)
	require.NoError(t, err)
	require.Less(t, status, 300, "deleting the line: %s", string(body))
	expectAuditEvent(t, added, "pick_line", "delete")
}

// --- Sales order contacts and acknowledgement ---

func TestOrderCascadeAudit_SalesOrderContactsAndAcknowledgement(t *testing.T) {
	t.Parallel()

	customerID := setupOrderCustomer(t)
	name, email := contactEmail("e2e-audit-so-contact")
	contactID := jsonField(createContact(t, apiClient.WithAccountID(customerID), name, email, nil), "id")

	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	// Contacts alone: no column on the order moves, so the event is only the contact lists.
	status, body, err = apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"acknowledgement_email_contacts": []map[string]any{{"account_user_id": contactID}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	contacts := expectAuditFieldChange(t, orderID, "sales_order", "update", "acknowledgement_email_contacts")
	assert.Empty(t, jsonArray(contacts, "old_value"))
	assert.Equal(t, []any{email}, jsonArray(contacts, "new_value"))

	// Issuing with a notification hands the send to a consumer, which flags the order acknowledged.
	status, body = salesOrderAction(t, orderID, "issue", true)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _ = salesOrderAction(t, orderID, "unissue", false) })

	sent := expectAuditFieldChange(t, orderID, "sales_order", "update", "is_acknowledgment_sent")
	assert.Equal(t, "true", jsonField(sent, "new_value"))
}

// --- Shipping re-sync ---

// Re-routing an order re-points its shipments out of band; each shipment that moved is audited.
func TestOrderCascadeAudit_ShipToChangeAuditsTheOrdersShipments(t *testing.T) {
	t.Parallel()

	order := issueOrderForCustomer(t, setupOrderCustomer(t), nil)
	orderID := jsonField(order, "id")
	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)
	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers)
	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	shipments := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, shipments)
	shipmentID := jsonField(shipments[0].(map[string]any), "id")

	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"shipping_address": map[string]any{"name": uniqueName("e2e-audit-reroute"), "line_1": "1 Audit Way", "country": "US"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	moved := expectAuditFieldChange(t, shipmentID, "shipment", "update", "shipping_address_id")
	assert.NotEqual(t, jsonField(moved, "old_value"), jsonField(moved, "new_value"))
}

// --- Purchase orders ---

func TestOrderCascadeAudit_PurchaseOrderIssueAndUnissueAuditTheReceivingOrder(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineIDs := idsOf(receivingOrderLines(t, receivingOrderID))
	require.NotEmpty(t, lineIDs)

	expectAuditEvent(t, receivingOrderID, "receiving_order", "create")
	for _, lineID := range lineIDs {
		expectAuditEvent(t, lineID, "receiving_order_line", "create")
	}

	status, body := changePurchaseOrderStatus(t, purchaseOrderID, "unissue")
	requireStatus(t, 200, status, body)

	expectAuditEvent(t, receivingOrderID, "receiving_order", "delete")
	for _, lineID := range lineIDs {
		expectAuditEvent(t, lineID, "receiving_order_line", "delete")
	}
}

// Closing a purchase order completes its receiving order, and re-opening clears that again.
func TestOrderCascadeAudit_PurchaseOrderCloseAndOpenAuditTheReceivingOrder(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body := changePurchaseOrderStatus(t, purchaseOrderID, "close")
	requireStatus(t, 200, status, body)
	completed := expectAuditFieldChange(t, receivingOrderID, "receiving_order", "update", "completed_at")
	assert.Nil(t, completed["old_value"])
	assert.NotEmpty(t, jsonField(completed, "new_value"))

	status, body = changePurchaseOrderStatus(t, purchaseOrderID, "open")
	requireStatus(t, 200, status, body)
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsWithChanges(t, receivingOrderID, "receiving_order", "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), "completed_at"); ok && c["new_value"] == nil {
				return nil
			}
		}
		return fmt.Errorf("no update yet clearing completed_at on receiving order %s", receivingOrderID)
	})
}

func TestOrderCascadeAudit_PurchaseOrderLineEditsAuditReceivingLines(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := issuedPurchaseOrderReceiving(t)
	before := idsOf(receivingOrderLines(t, receivingOrderID))

	status, body, err := apiClient.Post(purchaseOrdersPath+"/"+purchaseOrderID+"/lines", materialLineBody(uniqueName("E2E-PO-AUDIT")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	poLineID := jsonField(parseJSON(body), "id")

	var added string
	for _, id := range idsOf(receivingOrderLines(t, receivingOrderID)) {
		if !slices.Contains(before, id) {
			added = id
		}
	}
	require.NotEmpty(t, added, "the added line can be received against")
	expectAuditEvent(t, added, "receiving_order_line", "create")

	status, body, err = apiClient.Delete(purchaseOrdersPath + "/" + purchaseOrderID + "/lines/" + poLineID)
	require.NoError(t, err)
	require.Less(t, status, 300, "deleting the line: %s", string(body))
	expectAuditEvent(t, added, "receiving_order_line", "delete")
}

// Stocking the last line fulfils the purchase order on the receiving order's behalf.
func TestOrderCascadeAudit_StockingAuditsThePurchaseOrderFulfilled(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	// Issuing also changed the status, so look for the event that moved it to fulfilled.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		for _, event := range auditEventsWithChanges(t, purchaseOrderID, "purchase_order", "update") {
			if c, ok := changeForField(jsonListData(event, "changes"), "sales_order_status_code"); ok && c["new_value"] == "fulfilled" {
				return nil
			}
		}
		return fmt.Errorf("no update yet moving purchase order %s to fulfilled", purchaseOrderID)
	})
	completed := expectAuditFieldChange(t, purchaseOrderID, "purchase_order", "update", "completed_at")
	assertNilField(t, completed, "old_value")
}

func TestOrderCascadeAudit_PurchaseOrderContactsAndFlatAddresses(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, func(b map[string]any) {
		b["contact_account_user_ids"] = []string{SeedSupplierAccountUserID}
	})
	orderID := jsonField(created, "id")

	// The flat bill_to_* fields create a new address, which is audited like any other.
	billTo := jsonObject(retrievePurchaseOrder(t, orderID, "bill_to_address"), "bill_to_address")
	require.NotNil(t, billTo)
	expectAuditEvent(t, jsonField(billTo, "id"), "address", "create")

	status, body, err := apiClient.Patch(purchaseOrdersPath+"/"+orderID, map[string]any{"contact_account_user_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	contacts := expectAuditFieldChange(t, orderID, "purchase_order", "update", "contact_account_user_ids")
	assert.Equal(t, []any{SeedSupplierAccountUserID}, jsonArray(contacts, "old_value"))
	assert.Empty(t, jsonArray(contacts, "new_value"))
}

// Creating a purchase order audits each of its lines under the order.
func TestOrderCascadeAudit_PurchaseOrderCreateAuditsItsLines(t *testing.T) {
	t.Parallel()

	skus := []string{uniqueName("E2E-PO-AUD-A"), uniqueName("E2E-PO-AUD-B")}
	created := createPurchaseOrder(t, func(b map[string]any) {
		b["lines"] = []map[string]any{materialLineBody(skus[0]), materialLineBody(skus[1])}
	})
	orderID := jsonField(created, "id")
	lineIDs := idsOf(jsonListData(retrievePurchaseOrder(t, orderID, "lines"), "lines"))
	require.Len(t, lineIDs, 2)

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
			"root_resource_types": {"purchase_order"},
			"root_resource_ids":   {orderID},
			"resource_types":      {"purchase_order_line"},
			"actions":             {"create"},
			"limit":               {"100"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("audit events: %d %s", status, body)
		}
		var audited []string
		for _, item := range jsonArray(parseJSON(body), "data") {
			audited = append(audited, jsonField(item.(map[string]any), "resource_id"))
		}
		slices.Sort(audited)
		want := slices.Sorted(slices.Values(lineIDs))
		if !slices.Equal(audited, want) {
			return fmt.Errorf("line creates under the order: got %v, want %v", audited, want)
		}
		return nil
	})

	sku := expectAuditFieldChange(t, lineIDs[0], "purchase_order_line", "create", "product_sku")
	assert.Contains(t, skus, jsonField(sku, "new_value"))
}

// Ordering a material from a supplier it is not yet linked to links it, and the link is audited.
func TestOrderCascadeAudit_PurchaseOrderLineLinksTheSupplierMaterial(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-audit-mat")
	itemID := createItemsViaMaterials(t, sku, 1)[0]
	partNumber := uniqueName("E2E-PO-LINK")

	line := materialLineBody(partNumber)
	line["item_id"] = itemID
	createPurchaseOrder(t, func(b map[string]any) { b["lines"] = []map[string]any{line} })

	status, body, err := apiClient.GetListRaw(supplierItemsPath(SeedSupplierAccountID), url.Values{"q": {partNumber}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	links := jsonArray(parseJSON(body), "data")
	require.Len(t, links, 1, "the order linked the material to its supplier: %s", string(body))
	linkID := jsonField(links[0].(map[string]any), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierItemsPath(SeedSupplierAccountID) + "/" + linkID) })

	expectAuditEvent(t, linkID, "supplier_item", "create")
	expectSupplierItemRootedEvents(t, SeedSupplierAccountID, linkID, 1)
}

// --- Settlements ---

// A settlement's own transactions and allocations are created and deleted with it, and each is audited under it.
func TestOrderCascadeAudit_SettlementAuditsItsTransactionsAndAllocations(t *testing.T) {
	t.Parallel()

	inv := invoiceNewCustomer(t)
	settlement := settle(t, map[string]any{
		"new_transactions": []any{map[string]any{"key": "adj", "type": "adjustment", "adjustment_type": "discount", "customer_id": inv.customerID}},
		"allocations":      []any{map[string]any{"transaction_key": "adj", "invoice_id": inv.invoiceID, "amount": "5.00"}},
	})
	settlementID := jsonField(settlement, "id")

	status, body, err := apiClient.GetListRaw(financeSettlementsPath+"/"+settlementID, url.Values{"include": {"allocations", "allocations.transaction"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	allocations := jsonArray(jsonObject(parseJSON(body), "allocations"), "data")
	require.Len(t, allocations, 1)
	allocation := allocations[0].(map[string]any)
	allocationID := jsonField(allocation, "id")
	transactionID := jsonField(jsonObject(allocation, "transaction"), "id")

	expectAuditEvent(t, transactionID, "transaction", "create")
	expectAuditEvent(t, allocationID, "transaction_allocation", "create")

	status, body, err = apiClient.Delete(financeSettlementsPath + "/" + settlementID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	expectAuditEvent(t, transactionID, "transaction", "delete")
	expectAuditEvent(t, allocationID, "transaction_allocation", "delete")
}

// --- Customers ---

func TestOrderCascadeAudit_CustomerNotificationRecipients(t *testing.T) {
	t.Parallel()

	customerID := createContactsCustomer(t)
	name, email := contactEmail("e2e-audit-recipient")
	contact := createContact(t, apiClient.WithAccountID(customerID), name, email, map[string]bool{"order_acknowledgement": false, "invoice": false})
	contactID := jsonField(contact, "id")

	status, body, err := apiClient.Patch(notifRecipientsPath(customerID), notifRecipientBody(map[string]any{
		"account_user_id":    contactID,
		"notification_types": []string{"order_acknowledgement"},
	}), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	recipients := expectAuditFieldChange(t, customerID, "customer", "update", "notification_recipients")
	assert.Empty(t, jsonArray(recipients, "old_value"))
	assert.Equal(t, []any{map[string]any{
		"account_user_id":         contactID,
		"notification_type_codes": []any{"order_acknowledgement"},
	}}, jsonArray(recipients, "new_value"))
}

// --- Jobs ---

// A job's lifecycle marks are what its status is read from, so each is audited.
func TestOrderCascadeAudit_JobLifecycleIsAudited(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-audit-job"), uniqueName("eaj"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	if created, _ := jobResultIDs(job); len(created) > 0 {
		t.Cleanup(func() { cleanupUnitIDs(created) })
	}
	require.Equal(t, "completed", jsonField(job, "status"))

	completed := expectAuditFieldChange(t, jobID, "async_job", "update", "completed_at")
	assertNilField(t, completed, "old_value")
}

// Cancelling races the worker; whichever transition wins is the one audited.
func TestOrderCascadeAudit_JobCancelIsAudited(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-audit-cxl"), uniqueName("eac"), "quantity"))
	_, _, err := apiClient.Post(jobsPath+"/"+jobID+"/cancel", nil, "")
	require.NoError(t, err)

	job := pollJobUntilTerminal(t, jobID)
	if created, _ := jobResultIDs(job); len(created) > 0 {
		t.Cleanup(func() { cleanupUnitIDs(created) })
	}
	switch jsonField(job, "status") {
	case "cancelled":
		expectAuditFieldChange(t, jobID, "async_job", "update", "cancelled_at")
	case "completed":
		expectAuditFieldChange(t, jobID, "async_job", "update", "completed_at")
	default:
		t.Fatalf("unexpected terminal status %q", jsonField(job, "status"))
	}
}
