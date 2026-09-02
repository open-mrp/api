package email

import (
	"context"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

// What the merchant-facing templates actually render, against the dashboard components they
// replaced.
//
// The param maps below are the shapes core-service produces (invoiceDoc.emailParams and
// purchaseOrderDoc.emailParams); the core-service tests assert that those builders emit exactly these
// keys, so between them a rename on either side fails a test instead of silently blanking a section
// of a customer's invoice.

func render(t *testing.T, id constants.EmailTemplate, params map[string]any) string {
	t.Helper()
	renderer, apiErr := NewTemplateRenderer()
	if apiErr != nil {
		t.Fatalf("construct renderer: %v", apiErr)
	}
	out, apiErr := renderer.RenderTemplate(context.Background(), id, params)
	if apiErr != nil {
		t.Fatalf("render %s: %v", id, apiErr)
	}
	return out
}

func assertContains(t *testing.T, html string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("rendered output missing %q", want)
		}
	}
}

func assertOmits(t *testing.T, html string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(html, u) {
			t.Errorf("rendered output should not contain %q", u)
		}
	}
}

func invoiceParams() map[string]any {
	return map[string]any{
		"account_name":        "Carolon Co",
		"logo_url":            "https://cdn.example/logo.png",
		"invoice_number":      "005821",
		"invoice_date":        "7/14/2026",
		"invoice_total":       "$11,475.00",
		"master_tracking_url": "https://track.example/1Z999",
		"has_bill_to":         true,
		"bill_to_name":        "Northwind Traders",
		"bill_to_line1":       "55 Harbour Way",
		"bill_to_line2":       "Suite 300",
		"bill_to_csz":         "Seattle, WA 98101",
		"lines": []map[string]any{
			{"sku": "SOCK-CREW-BLK", "description": "Crew sock, black, size L", "qty": "1,200 pair", "price": "$8.50 / dz", "total": "$10,200.00"},
			{"sku": "SOCK-ANK-WHT", "description": "", "qty": "300 pair", "price": "$4.25 / pr", "total": "$1,275.00"},
		},
		"account_email":     "service@carolon.com",
		"account_website":   "https://carolon.com",
		"year":              "2026",
		"customer_number":   "42",
		"order_online_link": "https://app.example/carolon/auth/register",
		"email_subject":     "Invoice 005821",
		"instagram_handle":  "carolon",
		"twitter_handle":    "",
		"facebook_handle":   "",
		"linkedin_handle":   "",
		"marketing_blurb":   "Proudly made in North Carolina since 1948.",
	}
}

func TestInvoiceEmailRendersLegacyContent(t *testing.T) {
	t.Parallel()

	html := render(t, constants.EmailTemplateInvoice, invoiceParams())

	t.Run("header copy matches the legacy EmailHeader", func(t *testing.T) {
		assertContains(t, html,
			"Your order has shipped!",
			"Thank you for your recent order. We are pleased to confirm that we have shipped your order.",
		)
	})

	t.Run("tracking button links to the carrier", func(t *testing.T) {
		assertContains(t, html, "Track Shipment", "https://track.example/1Z999")
	})

	t.Run("invoice summary and bill to both render", func(t *testing.T) {
		assertContains(t, html,
			"Invoice Summary", "Invoice Number:", "005821", "Invoice Date:", "7/14/2026",
			"Bill To", "Northwind Traders", "55 Harbour Way", "Suite 300", "Seattle, WA 98101",
		)
	})

	t.Run("line table carries the legacy three columns", func(t *testing.T) {
		assertContains(t, html,
			"Product", "Quantity", "Price",
			"SOCK-CREW-BLK", "Crew sock, black, size L", "1,200 pair", "$10,200.00",
			"SOCK-ANK-WHT", "300 pair", "$1,275.00",
			"$11,475.00",
		)
	})

	t.Run("portal call to action quotes the customer number", func(t *testing.T) {
		assertContains(t, html,
			"Want to order online? Click the button below:",
			"Order Online",
			"https://app.example/carolon/auth/register",
			"you will need your customer number 42 to link your account",
		)
	})

	t.Run("footer carries branding, socials and copyright", func(t *testing.T) {
		assertContains(t, html,
			"Proudly made in North Carolina since 1948.",
			"https://www.instagram.com/carolon/",
			"service@carolon.com",
			"https://carolon.com",
			"&copy; 2026 Carolon Co. All rights reserved.",
		)
		// Handles that are blank must not render an empty link.
		assertOmits(t, html, "https://www.x.com//", "https://www.facebook.com//", "https://www.linkedin.com//")
	})
}

// Each optional block is gated on its own parameter, so an account without a portal, a shipment
// without tracking, or an invoice without a billing address drops that section rather than rendering
// an empty heading or a link to nowhere.
func TestInvoiceEmailDropsOptionalSections(t *testing.T) {
	t.Parallel()

	params := invoiceParams()
	params["master_tracking_url"] = ""
	params["order_online_link"] = ""
	params["has_bill_to"] = false
	params["logo_url"] = ""
	params["marketing_blurb"] = ""
	params["instagram_handle"] = ""

	html := render(t, constants.EmailTemplateInvoice, params)

	assertOmits(t, html,
		"Track Shipment",
		"Want to order online?",
		"Order Online",
		"Bill To",
		"<img",
		"instagram.com",
	)
	// The invoice itself must survive every section being dropped.
	assertContains(t, html, "Your order has shipped!", "005821", "$11,475.00", "SOCK-CREW-BLK")
}

// A line with no description renders the SKU alone rather than an empty sub-line.
func TestInvoiceEmailLineWithoutDescription(t *testing.T) {
	t.Parallel()

	params := invoiceParams()
	params["lines"] = []map[string]any{
		{"sku": "SKU-ONLY", "description": "", "qty": "1 ea", "price": "$1.00", "total": "$1.00"},
	}
	html := render(t, constants.EmailTemplateInvoice, params)

	assertContains(t, html, "SKU-ONLY")
	assertOmits(t, html, `<div style="color: #8a94a6; font-size: 0.8rem;"></div>`)
}

func purchaseOrderParams() map[string]any {
	return map[string]any{
		"account_name":            "Augno Manufacturing",
		"logo_url":                "https://cdn.example/logo.png",
		"order_number":            "000417",
		"requested_delivery_date": "04/18/2026",
		"submitted_on":            "03/04/2026",
		"order_total":             "$10,262.50",
		"has_ship_to":             true,
		"ship_to_name":            "Augno Receiving",
		"ship_to_line1":           "240 Mill St",
		"ship_to_line2":           "",
		"ship_to_csz":             "Akron, OH 44302",
		"lines": []map[string]any{
			{"sku": "WSHR-M6", "description": "Flat washer M6", "qty": "1,200 pair", "unit_price": "$8.5000 / pr", "total": "$10,200.00"},
			{"sku": "BOLT-M6", "description": "Hex bolt M6x40, zinc", "qty": "5,000 each", "unit_price": "$0.0125 / ea", "total": "$62.50"},
		},
		"account_email":    "orders@augno.com",
		"account_website":  "https://augno.com",
		"year":             "2026",
		"email_subject":    "Purchase Order 000417 Submission",
		"instagram_handle": "",
		"twitter_handle":   "",
		"facebook_handle":  "",
		"linkedin_handle":  "",
		"marketing_blurb":  "",
	}
}

func TestPurchaseOrderSubmissionRendersLegacyContent(t *testing.T) {
	t.Parallel()

	html := render(t, constants.EmailTemplatePurchaseOrderSubmission, purchaseOrderParams())

	t.Run("header copy matches the legacy EmailHeader", func(t *testing.T) {
		assertContains(t, html,
			"You have received a new order!",
			"Augno Manufacturing has submitted a new order to your account. Please review the order and process it accordingly.",
		)
	})

	t.Run("purchase order summary carries all four rows", func(t *testing.T) {
		assertContains(t, html,
			"Purchase Order Summary",
			"Purchase Order Number:", "000417",
			"Requested Delivery Date:", "04/18/2026",
			"Submitted On:", "03/04/2026",
			"Order Total:", "$10,262.50",
		)
	})

	t.Run("ship to renders beside the summary", func(t *testing.T) {
		assertContains(t, html, "Ship To", "Augno Receiving", "240 Mill St", "Akron, OH 44302")
	})

	t.Run("line table carries the legacy four columns at four-decimal prices", func(t *testing.T) {
		assertContains(t, html,
			"Product", "Quantity", "Unit Price", "Total",
			"WSHR-M6", "Flat washer M6", "1,200 pair", "$8.5000 / pr", "$10,200.00",
			"BOLT-M6", "Hex bolt M6x40, zinc", "5,000 each", "$0.0125 / ea", "$62.50",
		)
	})

	t.Run("footer carries the supplier's contact route back to the buyer", func(t *testing.T) {
		assertContains(t, html,
			"orders@augno.com",
			"https://augno.com",
			"&copy; 2026 Augno Manufacturing. All rights reserved.",
		)
	})
}

func TestPurchaseOrderSubmissionDropsOptionalSections(t *testing.T) {
	t.Parallel()

	params := purchaseOrderParams()
	params["requested_delivery_date"] = ""
	params["has_ship_to"] = false
	params["logo_url"] = ""

	html := render(t, constants.EmailTemplatePurchaseOrderSubmission, params)

	assertOmits(t, html, "Requested Delivery Date:", "Ship To", "<img")
	assertContains(t, html, "You have received a new order!", "000417", "Submitted On:", "$10,262.50")
}

// Template parameters carry account- and item-supplied text. html/template escapes by default, and
// these assert it stays that way: a supplier name or product description is not a license to inject
// markup into every recipient's inbox.
func TestMerchantTemplatesEscapeInterpolatedText(t *testing.T) {
	t.Parallel()

	const payload = `<script>alert(1)</script>`

	invoice := invoiceParams()
	invoice["account_name"] = payload
	invoice["lines"] = []map[string]any{
		{"sku": payload, "description": payload, "qty": "1 ea", "price": "$1.00", "total": "$1.00"},
	}
	assertOmits(t, render(t, constants.EmailTemplateInvoice, invoice), payload)

	po := purchaseOrderParams()
	po["account_name"] = payload
	po["lines"] = []map[string]any{
		{"sku": payload, "description": payload, "qty": "1 ea", "unit_price": "$1.00", "total": "$1.00"},
	}
	assertOmits(t, render(t, constants.EmailTemplatePurchaseOrderSubmission, po), payload)
}

// Every merchant template invites a reply, so each must be one the sender override treats as the
// merchant's own correspondence. A template that renders a merchant letterhead but sends from the
// platform address asks the customer to reply to an address nobody reads.
func TestMerchantFacingTemplatesSendAsMerchant(t *testing.T) {
	t.Parallel()

	for _, id := range []constants.EmailTemplate{
		constants.EmailTemplateInvoice,
		constants.EmailTemplatePurchaseOrderSubmission,
		constants.EmailTemplateOrderAcknowledgement,
		constants.EmailTemplateOrderCheckout,
		constants.EmailTemplateStatementOfAccount,
	} {
		if !id.SendsAsMerchant() {
			t.Errorf("template %q renders merchant branding but does not send as the merchant", id)
		}
	}

	// And the converse: a credential email must never leave from a tenant's domain.
	for _, id := range []constants.EmailTemplate{
		constants.EmailTemplatePasswordReset,
		constants.EmailTemplateRegistrationVerify,
		constants.EmailTemplateNewUserWelcome,
		constants.EmailTemplateChatMessage,
	} {
		if id.SendsAsMerchant() {
			t.Errorf("template %q must not send from a tenant domain", id)
		}
	}
}

func acknowledgementParams() map[string]any {
	return map[string]any{
		"account_name":    "Carolon Co",
		"logo_url":        "https://cdn.example/logo.png",
		"order_number":    "009001",
		"order_date":      "5/10/2026",
		"order_total":     "$11,470.75",
		"customer_po":     "PO-77321",
		"customer_number": "42",
		"has_ship_to":     true,
		"ship_to_name":    "Northwind DC",
		"ship_to_line1":   "9 Dock Rd",
		"ship_to_line2":   "",
		"ship_to_csz":     "Tacoma, WA 98402",
		"lines": []map[string]any{
			{"sku": "SOCK-CREW-BLK", "description": "Crew sock, black, size L", "qty": "1,200 pair", "price": "$8.50 / dz", "total": "$10,195.75"},
		},
		"account_email":     "service@carolon.com",
		"account_website":   "https://carolon.com",
		"year":              "2026",
		"order_online_link": "",
		"email_subject":     "Sales Order 009001",
		"instagram_handle":  "",
		"twitter_handle":    "",
		"facebook_handle":   "",
		"linkedin_handle":   "",
		"marketing_blurb":   "",
	}
}

// The email names the unit in full, as its PDF does. The core-service test asserts the builder emits
// that form under "qty"; this asserts the template renders the cell, so the pair covers the seam.
func TestOrderAcknowledgementEmailRendersUnitName(t *testing.T) {
	t.Parallel()

	html := render(t, constants.EmailTemplateOrderAcknowledgement, acknowledgementParams())

	assertContains(t, html,
		"009001", "PO-77321", "SOCK-CREW-BLK", "Crew sock, black, size L",
		"1,200 pair", "$10,195.75", "$11,470.75",
		"Northwind DC", "Tacoma, WA 98402",
	)
	// Nothing abbreviates the unit any more.
	assertOmits(t, html, "1,200 pr")
}
