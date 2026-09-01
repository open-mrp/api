package event

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
)

// The wire contract between the Express API and these consumers.
//
// Express writes rows into message_outbox that the Go enqueuer publishes unchanged, so the two sides
// agree on a JSON shape that neither language checks for the other. Two parts of it carry no type
// safety across the boundary and fail silently when they drift:
//
//   - `data` is a Go []byte, which encoding/json reads as base64. A payload embedded as a nested
//     object publishes cleanly and is undecodable here.
//   - types.Identity and IdentityTarget have no json tags, so Go marshals and unmarshals them under
//     their Go field names. Lowercase keys leave the consumer with no account, and the message is
//     dropped as malformed rather than failing loudly.
//
// The literals below are the same shapes asserted in apps/api/src/messaging/outbox.test.ts. Changing
// either side breaks a test rather than production.

// envelopeJSON builds the envelope exactly as publishToOutbox writes it.
func envelopeJSON(t *testing.T, accountID string, payload any) []byte {
	t.Helper()

	inner, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return []byte(`{
        "identity": {"Target": {"AccountID": "` + accountID + `"}},
        "data": "` + base64.StdEncoding.EncodeToString(inner) + `",
        "message_id": "3f1a0c2e-0000-4000-8000-000000000000"
    }`)
}

func TestEnvelopeDecodesIdentityWrittenByExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if msg.Identity == nil || msg.Identity.Target == nil {
		t.Fatal("identity did not decode; PascalCase field names are load-bearing")
	}
	if msg.Identity.Target.AccountID != "ac_test" {
		t.Fatalf("account = %q, want ac_test", msg.Identity.Target.AccountID)
	}
	if msg.MessageID != "3f1a0c2e-0000-4000-8000-000000000000" {
		t.Fatalf("message id = %q", msg.MessageID)
	}
}

// A payload written as a nested object rather than base64 must not decode — this is the shape the
// contract most plausibly drifts into, and it has to fail here rather than quietly.
func TestEnvelopeRejectsUnencodedPayload(t *testing.T) {
	t.Parallel()

	body := []byte(`{"identity":{"Target":{"AccountID":"ac_test"}},"data":{"account_id":"ac_test"}}`)

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err == nil {
		t.Fatal("a nested-object data field should not decode into []byte")
	}
}

func TestBatchScannedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{
		"account_id":          "ac_test",
		"batch_id":            "bt_1",
		"production_step_id":  "prst_1",
		"scanning_station_id": "sgsn_1",
		"item_id":             "it_1",
		"measure":             "60",
		"unit_id":             "un_pair",
		"seconds_measure":     "5",
		"waste_measure":       "7",
		"responsible_user_id": "us_1",
		"scanned_at":          "2026-08-19T12:00:00.000Z",
	})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.BatchScannedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.AccountID != "ac_test" || evt.BatchID != "bt_1" || evt.ProductionStepID != "prst_1" {
		t.Fatalf("identifiers did not decode: %+v", evt)
	}
	if evt.Measure != "60" || evt.UnitID != "un_pair" {
		t.Fatalf("measure did not decode: %+v", evt)
	}
	// Measures travel as strings so a decimal quantity is not rounded through a float on the way.
	seconds, err := evt.SecondsDecimal()
	if err != nil || seconds.String() != "5" {
		t.Fatalf("seconds = %s (err %v)", seconds, err)
	}
	waste, err := evt.WasteDecimal()
	if err != nil || waste.String() != "7" {
		t.Fatalf("waste = %s (err %v)", waste, err)
	}
	if evt.ResponsibleUserID == nil || *evt.ResponsibleUserID != "us_1" {
		t.Fatalf("responsible user did not decode: %+v", evt.ResponsibleUserID)
	}
	if evt.ScannedAt.IsZero() {
		t.Fatal("scanned_at did not decode; Go expects RFC3339, which toISOString produces")
	}
}

// A scan with no scrap omits the two measures entirely rather than sending nulls, and they must read
// as zero rather than as a parse failure.
func TestBatchScannedPayloadWithoutScrap(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{
		"account_id":         "ac_test",
		"batch_id":           "bt_1",
		"production_step_id": "prst_1",
		"item_id":            "it_1",
		"measure":            "60",
		"unit_id":            "un_pair",
		"scanned_at":         "2026-08-19T12:00:00.000Z",
	})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.BatchScannedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	seconds, err := evt.SecondsDecimal()
	if err != nil || !seconds.IsZero() {
		t.Fatalf("absent seconds should read zero, got %s (err %v)", seconds, err)
	}
	if evt.ResponsibleUserID != nil {
		t.Fatal("absent responsible_user_id should stay nil")
	}
}

func TestInventoryReceivedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{
		"account_id": "ac_test",
		"item_ids":   []string{"it_a", "it_b"},
		"reason":     "production_step_executed",
	})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.InventoryReceivedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.AccountID != "ac_test" {
		t.Fatalf("account = %q", evt.AccountID)
	}
	if len(evt.ItemIDs) != 2 || evt.ItemIDs[0] != "it_a" || evt.ItemIDs[1] != "it_b" {
		t.Fatalf("item ids did not decode: %+v", evt.ItemIDs)
	}
	if evt.Reason != "production_step_executed" {
		t.Fatalf("reason = %q", evt.Reason)
	}
}

func TestItemCostBasisChangedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{
		"unit_cost_updated",
		"production_step_updated",
		"consumption_created",
		"consumption_updated",
		"consumption_deleted",
	} {
		body := envelopeJSON(t, "ac_test", map[string]any{
			"account_id": "ac_test",
			"item_id":    "it_yarn",
			"reason":     reason,
		})

		var msg contracts.AmqpMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Fatalf("unmarshal envelope: %v", err)
		}

		var evt domain.ItemCostBasisChangedEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		if evt.AccountID != "ac_test" || evt.ItemID != "it_yarn" || evt.Reason != reason {
			t.Fatalf("payload did not decode for reason %q: %+v", reason, evt)
		}
	}
}

// The account is read off the payload first so a replay does not depend on the envelope surviving,
// and falls back to the identity for anything published before the field existed.
func TestAccountFallsBackToIdentity(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_from_identity", map[string]any{"item_ids": []string{"it_a"}})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.InventoryReceivedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	accountID := evt.AccountID
	if accountID == "" && msg.Identity != nil && msg.Identity.Target != nil {
		accountID = msg.Identity.Target.AccountID
	}
	if accountID != "ac_from_identity" {
		t.Fatalf("account = %q, want the identity's", accountID)
	}
}

func TestInvoiceIssuedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{
		"invoice_id":      "inv_1",
		"email_customer":  true,
		"email_sales_rep": true,
	})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.InvoiceIssuedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.InvoiceID != "inv_1" {
		t.Fatalf("invoice = %q", evt.InvoiceID)
	}
	if !evt.EmailCustomer || !evt.EmailSalesRep {
		t.Fatalf("recipient flags did not decode: %+v", evt)
	}
}

// A ship with the customer copy suppressed sends email_customer false rather than omitting it, but
// either way the rep copy must survive on its own — losing the distinction mails the customer an
// invoice the operator chose not to send.
func TestInvoiceIssuedPayloadWithoutCustomerCopy(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{
		"invoice_id":      "inv_1",
		"email_customer":  false,
		"email_sales_rep": true,
	})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.InvoiceIssuedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.EmailCustomer {
		t.Fatal("email_customer should decode false")
	}
	if !evt.EmailSalesRep {
		t.Fatal("email_sales_rep should decode true")
	}
}

func TestSalesOrderAcknowledgedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{"sales_order_id": "so_1"})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.SalesOrderAcknowledgedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.SalesOrderID != "so_1" {
		t.Fatalf("sales order = %q", evt.SalesOrderID)
	}
}

func TestPurchaseOrderSubmittedPayloadDecodesFromExpress(t *testing.T) {
	t.Parallel()

	body := envelopeJSON(t, "ac_test", map[string]any{"purchase_order_id": "po_1"})

	var msg contracts.AmqpMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var evt domain.PurchaseOrderSubmittedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if evt.PurchaseOrderID != "po_1" {
		t.Fatalf("purchase order = %q", evt.PurchaseOrderID)
	}
}
