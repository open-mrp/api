package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
)

// checkoutEmailParams renders the payment-request email from the shared order view model. It differs from emailParams only in showing Bill To rather than Ship To — the mail asks for money, so the party being billed is the relevant one — and in carrying the Stripe session URL.
func (d ackData) checkoutEmailParams(checkoutURL string) map[string]any {
	lines := make([]map[string]any, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = map[string]any{
			"sku":         l.SKU,
			"description": l.Description,
			"qty":         l.Qty,
			"price":       l.Price,
			"total":       l.Total,
		}
	}
	return map[string]any{
		"account_name":     d.AccountName,
		"customer_name":    d.CustomerName,
		"logo_url":         d.LogoURL,
		"checkout_url":     checkoutURL,
		"order_number":     d.OrderNumber,
		"customer_po":      d.CustomerPO,
		"order_date":       d.OrderDateShort,
		"order_total":      d.OrderTotal,
		"has_bill_to":      !d.BillTo.Empty(),
		"bill_to_name":     d.BillTo.Name,
		"bill_to_line1":    d.BillTo.Line1,
		"bill_to_line2":    d.BillTo.Line2,
		"bill_to_csz":      d.BillTo.CityStateZip,
		"lines":            lines,
		"account_email":    d.AccountEmail,
		"account_website":  d.AccountWebsite,
		"year":             d.Year,
		"email_subject":    d.EmailSubject,
		"instagram_handle": d.InstagramHandle,
		"twitter_handle":   d.TwitterHandle,
		"facebook_handle":  d.FacebookHandle,
		"linkedin_handle":  d.LinkedInHandle,
		"marketing_blurb":  d.MarketingBlurb,
	}
}

// buildOrderCheckoutEmail assembles the payment-request email for an order on the same branded letterhead as the acknowledgement and the invoice. The recipient is the address the checkout link was requested for, which may be a one-off contact rather than the customer's billing address, so it is passed in rather than read from the order.
//
// The letterhead lookups are best-effort, matching buildOrderAcknowledgementEmail: a missing account or origin address degrades to a blank letterhead rather than failing a checkout the buyer is waiting on.
func buildOrderCheckoutEmail(ctx context.Context, repos domain.RepoFactory, branding BrandingAssets, order *domain.SalesOrder, lines []*domain.SalesOrderLine, accountID, sellerName, recipient, checkoutURL string) *messaging.EmailSendData {
	account, _ := repos.NewAccountRepo().GetByID(ctx, accountID)
	originAddr, _ := repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, accountID)

	data := buildOrderAcknowledgementData(order, lines, account, originAddr)
	// buildOrderAcknowledgementData falls back to the buyer's name when the seller account cannot be read. That fallback is how a payment request came to greet the customer by their own name with the merchant nowhere on it, so the seller name resolved at the call site wins, and an unresolvable seller leaves the letterhead blank rather than wearing the buyer's identity.
	data.AccountName = sellerName
	if data.LogoURL != "" {
		data.LogoURL = branding.LogoURL(ctx, data.LogoURL)
	}

	subject := fmt.Sprintf("Order %s is ready to pay", data.OrderNumber)
	if data.AccountName != "" {
		subject = fmt.Sprintf("%s — %s", data.AccountName, subject)
	}

	emailData := &messaging.EmailSendData{
		To:         []string{recipient},
		Subject:    subject,
		TemplateID: constants.EmailTemplateOrderCheckout,
		Params:     data.checkoutEmailParams(checkoutURL),
		AccountID:  &accountID,
	}
	applyMerchantReplyTo(emailData, data.AccountEmail)
	return emailData
}

// checkoutSubmitMessage is the sentence Stripe renders directly above the pay button. It names the seller, the order, and when it was placed — the three things a buyer arriving from an email needs in order to recognize a charge before entering a card.
func checkoutSubmitMessage(sellerName, orderNumber, orderDate string) string {
	if sellerName == "" {
		return fmt.Sprintf("Payment for order %s, placed %s.", orderNumber, orderDate)
	}
	return fmt.Sprintf("Payment to %s for order %s, placed %s.", sellerName, orderNumber, orderDate)
}

// statementDescriptorSuffix renders the order number for the buyer's card statement. Stripe accepts at most 22 characters and rejects quotes, apostrophes, angle brackets, and asterisks, so the number is reduced to alphanumerics and truncated rather than passed through.
func statementDescriptorSuffix(orderNumber string) string {
	const maxLength = 22

	var b strings.Builder
	for _, r := range orderNumber {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		}
		if b.Len() >= maxLength {
			break
		}
	}
	return b.String()
}
