package service

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/textutil"
)

// The purchase order a supplier receives: the submission email and the PDF attached to it, ported
// from the dashboard's PurchaseOrderSubmissionEmail and PurchaseOrderPdf.
//
// A purchase order is the one document here whose counterparty is the supplier rather than the
// customer, which is why it carries a supplier number instead of a customer number and a requested
// delivery date instead of order terms. Everything else — letterhead, addresses, the line table — is
// the shared acknowledgement furniture, so it reuses ackData rather than growing a parallel model.

// poUnitPriceDigits is the number of decimals the purchase order shows on a unit price. The
// dashboard passes 4 to RateUtils.abbreviate here and nowhere else: a purchased component is often
// priced in fractions of a cent, and rounding to 2 turned "$0.0125 / ea" into "$0.01 / ea" on the
// document the supplier invoices against.
const poUnitPriceDigits = 4

// purchaseOrderDoc is the purchase order's view model: the shared letterhead/addresses/table in
// Header, plus the two summary values only the email shows.
type purchaseOrderDoc struct {
	Header ackData
	// RequestedDeliveryDate is the promised date formatted for display, empty when the order has
	// none — the email's row is dropped entirely in that case, as the legacy template's is.
	RequestedDeliveryDate string
	// SubmittedOn is the order's creation date, which the email labels "Submitted On".
	SubmittedOn string
}

// gatherPurchaseOrderDoc collects everything the purchase order document renders. Every lookup
// except the order itself is best-effort: a failure blanks its own section rather than costing the
// supplier the order.
func gatherPurchaseOrderDoc(ctx context.Context, repos domain.RepoFactory, accountID, purchaseOrderID string, recipients []string) (purchaseOrderDoc, *apierror.APIError) {
	repo := repos.NewPurchaseOrderRepo()

	order, apiErr := repo.Get(ctx, accountID, purchaseOrderID)
	if apiErr != nil {
		return purchaseOrderDoc{}, apiErr
	}
	lines, _ := repo.GetLines(ctx, purchaseOrderID)

	account, _ := repos.NewAccountRepo().GetByID(ctx, accountID)
	// The buyer's own origin address heads the letterhead: the supplier is being told who is
	// ordering, so the address block is ours, not theirs.
	originAddr, _ := repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, accountID)

	return buildPurchaseOrderDoc(order, lines, account, originAddr, recipients), nil
}

// buildPurchaseOrderDoc assembles the purchase order document. Everything but the order is optional
// and degrades to a blank section.
func buildPurchaseOrderDoc(
	order *domain.PurchaseOrder,
	lines []*domain.PurchaseOrderLine,
	account *domain.Account,
	originAddr *domain.ShippingAddress,
	contactEmails []string,
) purchaseOrderDoc {
	number := textutil.FormatRecordNumber(order.Number)
	subject := "Purchase Order " + number + " Submission"

	d := ackData{
		DocumentTitle: "PURCHASE ORDER",
		NumberLabel:   "Purchase Order Number",
		AccountName:   accountDisplayName(account, order.SupplierName),
		OrderNumber:   number,
		// Rendered in the server's zone, as the dashboard's date-fns and toLocaleDateString are.
		OrderDateShort: order.CreatedAt.Local().Format("1/2/2006"),
		OrderDateLong:  order.CreatedAt.Local().Format("01/02/2006"),
		Year:           time.Now().Format("2006"),
		EmailSubject:   subject,
	}

	// The identity block names the supplier and the delivery they are being asked for, where a sales
	// order names the customer and its PO.
	requested := ""
	if order.PromisedAt != nil {
		requested = order.PromisedAt.Local().Format("01/02/2006")
	}
	d.IdentityRows = []ackIdentityField{
		{Label: "Supplier Number", Value: textutil.FormatAccountNumber(order.SupplierNumber)},
		{Label: "Date", Value: d.OrderDateLong},
		{Label: "Requested Delivery Date", Value: requested},
	}

	if account != nil && account.Branding != nil {
		d.LogoURL = ptrutil.Deref(account.Branding.LogoURL)
		d.AccountEmail = ptrutil.Deref(account.Branding.SupportEmail)
		d.AccountPhone = ptrutil.Deref(account.Branding.PhoneNumber)
		d.AccountWebsite = ptrutil.Deref(account.Branding.WebsiteURL)
		d.InstagramHandle = ptrutil.Deref(account.Branding.InstagramHandle)
		d.TwitterHandle = ptrutil.Deref(account.Branding.TwitterHandle)
		d.FacebookHandle = ptrutil.Deref(account.Branding.FacebookHandle)
		d.LinkedInHandle = ptrutil.Deref(account.Branding.LinkedInHandle)
	}
	if account != nil {
		d.MarketingBlurb = accountMarketingBlurbs[account.ID]
	}
	if originAddr != nil {
		d.AccountAddress = ackAddress{
			Line1:        originAddr.Street1,
			Line2:        ptrutil.Deref(originAddr.Street2),
			CityStateZip: joinCityStateZip(originAddr.City, originAddr.State, originAddr.Zip),
			Phone:        ptrutil.Deref(originAddr.Phone),
			Email:        ptrutil.Deref(originAddr.Email),
		}
	}

	d.BillTo = ackAddress{
		Name:         ptrutil.Deref(order.BillToName),
		Line1:        ptrutil.Deref(order.BillToStreetLine1),
		Line2:        ptrutil.Deref(order.BillToStreetLine2),
		CityStateZip: joinCityStateZip(ptrutil.Deref(order.BillToLocality), ptrutil.Deref(order.BillToState), ptrutil.Deref(order.BillToPostalCode)),
		Phone:        ptrutil.Deref(order.BillToPhone),
		Email:        ptrutil.Deref(order.BillToEmail),
	}
	d.ShipTo = ackAddress{
		Name:         ptrutil.Deref(order.ShipToName),
		Line1:        ptrutil.Deref(order.ShipToStreetLine1),
		Line2:        ptrutil.Deref(order.ShipToStreetLine2),
		CityStateZip: joinCityStateZip(ptrutil.Deref(order.ShipToLocality), ptrutil.Deref(order.ShipToState), ptrutil.Deref(order.ShipToPostalCode)),
		Phone:        ptrutil.Deref(order.ShipToPhone),
		Email:        ptrutil.Deref(order.ShipToEmail),
	}
	d.HasShipTo = !d.ShipTo.Empty()

	// The PDF lists the order's submission contacts under Bill To, lowercased as the legacy template
	// does. They are the same addresses the mail goes to, so the caller passes them rather than
	// having this re-read them.
	for _, email := range contactEmails {
		if email != "" {
			d.ContactEmails = append(d.ContactEmails, lowerASCII(email))
		}
	}

	// Ordered by line item number, matching PurchaseOrderLineUtils.sort, so the supplier reads the
	// lines in the order the buyer entered them rather than in insertion order.
	sorted := append([]*domain.PurchaseOrderLine(nil), lines...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].LineItemNumber < sorted[j].LineItemNumber })

	total := decimal.Zero
	for _, line := range sorted {
		price := parseDecimalOrZero(line.UnitPriceValue)
		qty := parseDecimalOrZero(line.QuantityValue)
		lineTotal := price.Mul(qty)
		total = total.Add(lineTotal)

		d.Lines = append(d.Lines, ackLine{
			LineItem:    fmt.Sprintf("%03d", line.LineItemNumber),
			SKU:         purchaseOrderLineSKU(line),
			Description: ptrutil.Deref(line.ProductDescription),
			Price:       formatRateAmount(price, line.UnitPriceDenominatorUnitAbbr, poUnitPriceDigits),
			// The unit's full name, on the email as well as the PDF.
			Qty:   formatMeasure(qty, line.QuantityUnitName, 0),
			Total: formatMoney(lineTotal),
		})
	}
	d.OrderTotal = formatMoney(total)

	return purchaseOrderDoc{
		Header:                d,
		RequestedDeliveryDate: requested,
		SubmittedOn:           d.OrderDateLong,
	}
}

// purchaseOrderLineSKU prefers the linked item's SKU, as the legacy document reads item.sku, and
// falls back to the line's own product SKU for a line that names no item.
func purchaseOrderLineSKU(line *domain.PurchaseOrderLine) string {
	if sku := ptrutil.Deref(line.ItemSKU); sku != "" {
		return sku
	}
	return line.ProductSKU
}

// lowerASCII lowercases without locale surprises, matching JavaScript's toLowerCase on the ASCII
// addresses these are.
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// emailParams flattens the purchase order into the submission email's template parameters.
func (d purchaseOrderDoc) emailParams() map[string]any {
	h := d.Header
	lines := make([]map[string]any, len(h.Lines))
	for i, l := range h.Lines {
		lines[i] = map[string]any{
			"sku":         l.SKU,
			"description": l.Description,
			"qty":         l.Qty,
			"unit_price":  l.Price,
			"total":       l.Total,
		}
	}
	return map[string]any{
		"account_name":            h.AccountName,
		"logo_url":                h.LogoURL,
		"order_number":            h.OrderNumber,
		"requested_delivery_date": d.RequestedDeliveryDate,
		"submitted_on":            d.SubmittedOn,
		"order_total":             h.OrderTotal,
		"has_ship_to":             h.HasShipTo,
		"ship_to_name":            h.ShipTo.Name,
		"ship_to_line1":           h.ShipTo.Line1,
		"ship_to_line2":           h.ShipTo.Line2,
		"ship_to_csz":             h.ShipTo.CityStateZip,
		"lines":                   lines,
		"account_email":           h.AccountEmail,
		"account_website":         h.AccountWebsite,
		"year":                    h.Year,
		"email_subject":           h.EmailSubject,
		"instagram_handle":        h.InstagramHandle,
		"twitter_handle":          h.TwitterHandle,
		"facebook_handle":         h.FacebookHandle,
		"linkedin_handle":         h.LinkedInHandle,
		"marketing_blurb":         h.MarketingBlurb,
	}
}

// buildPurchaseOrderPDF renders the purchase order PDF a supplier receives, mirroring the legacy
// PurchaseOrderPdf layout: letterhead with the document title block, bill-to / ship-to addresses,
// then the order summary table.
//
// It has no order-terms band. A purchase order carries no carrier, priority or sales rep for the
// supplier to act on — those belong to the sales order this may be fulfilling, not to the purchase.
func buildPurchaseOrderPDF(data ackData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(ackPageLeft, ackPageTop, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	ackHeader(pdf, data)
	ackHR(pdf)
	ackCustomerAddresses(pdf, data)
	ackHR(pdf)
	ackOrderSummary(pdf, data)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// purchaseOrderAttachmentFilename names the emailed purchase order PDF.
func purchaseOrderAttachmentFilename(orderNumber string) string {
	return "purchase-order-" + filenameSafe(orderNumber) + ".pdf"
}
