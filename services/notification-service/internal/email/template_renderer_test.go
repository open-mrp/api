package email

import (
	"context"
	"strings"
	"testing"

	"github.com/augno/api/shared/constants"

	"github.com/stretchr/testify/require"
)

// legacyOwnedTemplates are template IDs that core-service can construct but that the legacy Express API still sends and logs itself, so notification-service has no template for them. Remove an entry here the moment its send path cuts over to the outbox, or the first real send will fail to render and drop the email silently.
var legacyOwnedTemplates = map[constants.EmailTemplate]bool{
	constants.EmailTemplateStatementOfAccount:      true,
	constants.EmailTemplatePurchaseOrderSubmission: true,
}

// A published EmailTemplate with no entry in the renderer map renders nothing, so the consumer drops the message and no email is ever sent — the endpoint still returns 200 because the send is asynchronous. That is how order_checkout silently stopped mailing after its cutover, so pin every template to a renderer entry.
func TestRendererCoversEveryTemplate(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	impl, ok := renderer.(*templateRendererImpl)
	require.True(t, ok)

	for _, name := range constants.EmailTemplate("").EnumValues() {
		templateID := constants.EmailTemplate(name)
		if legacyOwnedTemplates[templateID] {
			continue
		}

		_, registered := impl.templates[templateID]
		require.Truef(t, registered, "template %q has no renderer entry, so every send of it would be dropped", name)
	}
}

func TestRenderOrderCheckoutIncludesCheckoutURL(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateOrderCheckout, map[string]any{
		"account_name": "Joseph Fretta, M.D.",
		"checkout_url": "https://pay.example.com/c/pay/cs_live_abc123",
		"order_number": "23124",
	})
	require.Nil(t, apiErr)

	require.Contains(t, body, "https://pay.example.com/c/pay/cs_live_abc123")
	require.Contains(t, body, "Joseph Fretta, M.D.")
	require.Contains(t, body, "23124")
	// html/template escapes an unsafe URL to "#ZgotmplZ" rather than erroring, which would ship a dead pay button.
	require.NotContains(t, body, "ZgotmplZ")
}

// The Customer PO row only renders when the order carries a PO number, so a blank PO doesn't leave an empty labeled row.
func TestRenderOrderAcknowledgementCustomerPO(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	params := map[string]any{
		"account_name": "Comme Cardiovascular",
		"order_number": "23124",
		"order_date":   "8/13/2026",
		"order_total":  "$150.00",
		"customer_po":  "PO-4521",
	}
	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateOrderAcknowledgement, params)
	require.Nil(t, apiErr)
	require.Contains(t, body, "Customer PO")
	require.Contains(t, body, "PO-4521")

	params["customer_po"] = ""
	body, apiErr = renderer.RenderTemplate(context.Background(), constants.EmailTemplateOrderAcknowledgement, params)
	require.Nil(t, apiErr)
	require.NotContains(t, body, "Customer PO")
}

// The Stripe checkout URL carries a query string of escaped fragments; html/template must keep it intact inside href.
func TestRenderOrderCheckoutPreservesEscapedQueryString(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	checkoutURL := "https://pay.example.com/c/pay/cs_test_a1b2c3d4e5f6g7h8i9j0k1l2#fidnandhYHdWcXxpYCc%2FJ2FgY2Rw"
	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateOrderCheckout, map[string]any{
		"account_name": "Comme Cardiovascular",
		"checkout_url": checkoutURL,
		"order_number": "23110",
	})
	require.Nil(t, apiErr)

	require.NotContains(t, body, "ZgotmplZ")
	require.True(t, strings.Contains(body, "cs_test_a1b2c3d4e5f6g7h8i9j0k1l2"))
}

// The invoice email is the customer-facing document for every ship, so pin the parts that carry
// meaning: the totals, the line items, the bill-to block and a live tracking link.
func TestRenderInvoiceIncludesSummaryLinesAndTracking(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateInvoice, map[string]any{
		"account_name":        "Seller Co",
		"invoice_number":      "INV-9",
		"invoice_date":        "5/10/2026",
		"invoice_total":       "$351.00",
		"master_tracking_url": "https://www.ups.com/track?tracknum=1Z999",
		"has_bill_to":         true,
		"bill_to_name":        "Acme Bill-To",
		"bill_to_line1":       "1 Main St",
		"bill_to_csz":         "Springfield, IL 62701",
		"year":                "2026",
		"lines": []map[string]any{
			{"sku": "SKU-1", "description": "Widget, 6061-T6", "qty": "6", "total": "$51.00"},
		},
	})
	require.Nil(t, apiErr)

	require.Contains(t, body, "Your order has shipped!")
	require.Contains(t, body, "INV-9")
	require.Contains(t, body, "$351.00")
	require.Contains(t, body, "Acme Bill-To")
	require.Contains(t, body, "Springfield, IL 62701")
	require.Contains(t, body, "SKU-1")
	require.Contains(t, body, "Widget, 6061-T6")
	require.Contains(t, body, "Track Shipment")
	require.Contains(t, body, "https://www.ups.com/track?tracknum=1Z999")
	// An escaped href would ship a dead tracking button.
	require.NotContains(t, body, "ZgotmplZ")
}

// Without a tracking URL the button is dropped rather than rendered pointing nowhere.
func TestRenderInvoiceOmitsTrackingButtonWithoutURL(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateInvoice, map[string]any{
		"account_name":   "Seller Co",
		"invoice_number": "INV-9",
		"invoice_total":  "$0.00",
	})
	require.Nil(t, apiErr)

	require.NotContains(t, body, "Track Shipment")
	require.Contains(t, body, "INV-9")
}

// The customer emails carry the dashboard's footer: social links for whatever handles the account
// set, a mailto that pre-fills the subject, and the Seller Co-only American-made panel.
func TestRenderInvoiceFooterMatchesDashboard(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateInvoice, map[string]any{
		"account_name":       "Seller Co",
		"account_email":      "orders@sellerco.example.com",
		"invoice_number":     "INV-9",
		"invoice_total":      "$351.00",
		"email_subject":      "Invoice 000009",
		"instagram_handle":   "sellerco",
		"twitter_handle":     "sellerco",
		"facebook_handle":    "sellerco",
		"linkedin_handle":    "sellerco",
		"show_american_made": true,
		"year":               "2026",
	})
	require.Nil(t, apiErr)

	// Twitter points at x.com, as the dashboard's link builder does.
	require.Contains(t, body, "https://www.instagram.com/sellerco/")
	require.Contains(t, body, "https://www.x.com/sellerco/")
	require.Contains(t, body, "https://www.facebook.com/sellerco/")
	require.Contains(t, body, "https://www.linkedin.com/sellerco/")
	// The subject rides on the mailto so a reply is already titled.
	require.Contains(t, body, "mailto:orders@sellerco.example.com?subject=Invoice%20000009")
	require.Contains(t, body, "compression products in the USA since 1975")
	require.NotContains(t, body, "ZgotmplZ")
}

// An account with no handles and no marketing panel gets neither — no empty icon row.
func TestRenderInvoiceFooterOmitsUnsetSections(t *testing.T) {
	renderer, apiErr := NewTemplateRenderer()
	require.Nil(t, apiErr)

	body, apiErr := renderer.RenderTemplate(context.Background(), constants.EmailTemplateInvoice, map[string]any{
		"account_name":   "Seller Co",
		"invoice_number": "INV-9",
		"invoice_total":  "$0.00",
	})
	require.Nil(t, apiErr)

	require.NotContains(t, body, "instagram.com")
	require.NotContains(t, body, "since 1975")
}
