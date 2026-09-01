package service

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/textutil"
	"github.com/shopspring/decimal"
)

// One row of the invoice's line table. Ordered and Invoiced are separate columns because an invoice
// bills what shipped, which is often less than what was ordered.
type invoiceLineRow struct {
	LineItem    string
	SKU         string
	Description string
	Price       string
	Ordered     string
	Invoiced    string
	// InvoicedWithUnit is the email's quantity cell, which carries the unit's full name ("6 pair").
	// The PDF splits the same figure across its Invoiced and Unit columns.
	InvoicedWithUnit string
	Unit             string
	Total            string
}

// One row of the invoice's Cases table.
type invoiceCaseRow struct {
	Number   string
	Weight   string
	Tracking string
}

// The invoice document's view model: the shared letterhead/addresses/terms, plus the line and case
// tables that are the invoice's own.
type invoiceDoc struct {
	Header     ackData
	Lines      []invoiceLineRow
	Cases      []invoiceCaseRow
	OrderTotal string
}

// Builds the carrier deep-link for a shipment's master tracking number, or "" when either the
// tracking number or the carrier is missing.
func shipmentMasterTrackingURL(shipment *domain.Shipment) string {
	if shipment == nil || shipment.MasterTrackingNumber == nil || shipment.CarrierCode == nil {
		return ""
	}
	return constants.TrackingURL(constants.CarrierCode(*shipment.CarrierCode), *shipment.MasterTrackingNumber)
}

// Flattens the invoice document into the invoice email's template parameters. masterTrackingURL is
// empty when the shipment has no tracking, which drops the Track Shipment button.
func (d invoiceDoc) emailParams(masterTrackingURL string) map[string]any {
	lines := make([]map[string]any, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = map[string]any{
			"sku":         l.SKU,
			"description": l.Description,
			"qty":         l.InvoicedWithUnit,
			"price":       l.Price,
			"total":       l.Total,
		}
	}
	return map[string]any{
		"account_name":        d.Header.AccountName,
		"logo_url":            d.Header.LogoURL,
		"invoice_number":      d.Header.OrderNumber,
		"invoice_date":        d.Header.OrderDateShort,
		"invoice_total":       d.OrderTotal,
		"master_tracking_url": masterTrackingURL,
		"has_bill_to":         !d.Header.BillTo.Empty(),
		"bill_to_name":        d.Header.BillTo.Name,
		"bill_to_line1":       d.Header.BillTo.Line1,
		"bill_to_line2":       d.Header.BillTo.Line2,
		"bill_to_csz":         d.Header.BillTo.CityStateZip,
		"lines":               lines,
		"account_email":       d.Header.AccountEmail,
		"account_website":     d.Header.AccountWebsite,
		"year":                d.Header.Year,
		"customer_number":     d.Header.CustomerNumberRaw,
		"order_online_link":   d.Header.OrderOnlineLink,
		"email_subject":       d.Header.EmailSubject,
		"instagram_handle":    d.Header.InstagramHandle,
		"twitter_handle":      d.Header.TwitterHandle,
		"facebook_handle":     d.Header.FacebookHandle,
		"linkedin_handle":     d.Header.LinkedInHandle,
		"marketing_blurb":     d.Header.MarketingBlurb,
	}
}

// Gathers everything the invoice document renders. Every lookup is best-effort: a failure blanks its
// own section rather than costing the customer the document.
func gatherInvoiceDoc(ctx context.Context, repos domain.RepoFactory, accountID string, invoice *domain.Invoice, lines []*domain.InvoiceLine) invoiceDoc {
	account, _ := repos.NewAccountRepo().GetByID(ctx, accountID)
	originAddr, _ := repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, accountID)

	var order *domain.SalesOrder
	if invoice.OrderID != "" {
		order, _ = repos.NewSalesOrderRepo().Get(ctx, accountID, invoice.OrderID)
	}

	var cases []*domain.ShippingCase
	if invoice.ShipmentID != nil {
		cases, _ = repos.NewShippingCaseRepo().ListByShipment(ctx, *invoice.ShipmentID)
	}

	contacts, _ := repos.NewInvoiceRepo().GetEmailRecipients(ctx, invoice.ID)

	return buildInvoiceDoc(invoice, lines, order, account, originAddr, cases, contacts)
}

// Assembles the invoice document from the invoice, its lines, the order behind it, and the shipment's
// cases. Everything but the invoice is optional and degrades to a blank section rather than failing.
func buildInvoiceDoc(
	invoice *domain.Invoice,
	lines []*domain.InvoiceLine,
	order *domain.SalesOrder,
	account *domain.Account,
	originAddr *domain.ShippingAddress,
	cases []*domain.ShippingCase,
	contactEmails []string,
) invoiceDoc {
	// The letterhead, addresses and terms are the order's, so build them once from the shared model
	// and then stamp the invoice's own identity over the top.
	var header ackData
	if order != nil {
		header = buildOrderAcknowledgementData(order, nil, account, originAddr)
		header.Lines = nil
		header.OrderTotal = ""
	} else {
		header = ackData{AccountName: accountDisplayName(account, invoice.CustomerName)}
		header.BillTo = ackAddress{
			Name:         ptrutil.Deref(invoice.BillingAddressName),
			Line1:        ptrutil.Deref(invoice.BillingAddressLine1),
			Line2:        ptrutil.Deref(invoice.BillingAddressLine2),
			CityStateZip: joinCityStateZip(ptrutil.Deref(invoice.BillingAddressCity), ptrutil.Deref(invoice.BillingAddressState), ptrutil.Deref(invoice.BillingAddressZip)),
		}
		if account != nil && account.Branding != nil {
			header.LogoURL = ptrutil.Deref(account.Branding.LogoURL)
			header.AccountEmail = ptrutil.Deref(account.Branding.SupportEmail)
			header.AccountPhone = ptrutil.Deref(account.Branding.PhoneNumber)
			header.AccountWebsite = ptrutil.Deref(account.Branding.WebsiteURL)
		}
		if originAddr != nil {
			header.AccountAddress = ackAddress{
				Line1:        originAddr.Street1,
				Line2:        ptrutil.Deref(originAddr.Street2),
				CityStateZip: joinCityStateZip(originAddr.City, originAddr.State, originAddr.Zip),
				Phone:        ptrutil.Deref(originAddr.Phone),
				Email:        ptrutil.Deref(originAddr.Email),
			}
		}
		header.CustomerNumber = textutil.FormatAccountNumber(invoice.CustomerNumber)
		header.CustomerNumberRaw = invoice.CustomerNumber
	}

	header.DocumentTitle = "INVOICE"
	header.NumberLabel = "Invoice Number"
	header.OrderNumber = textutil.FormatRecordNumber(invoice.Number)
	// The date carries the time on an invoice, which is stamped at the moment of shipping.
	header.OrderDateShort = invoice.CreatedAt.Local().Format("1/2/2006")
	header.OrderDateLong = invoice.CreatedAt.Local().Format("01/02/2006 03:04 PM")
	// The copyright line carries the year the mail goes out, not the invoice's.
	header.Year = time.Now().Format("2006")
	header.EmailSubject = "Invoice " + textutil.FormatRecordNumber(invoice.Number)
	header.ContactEmails = contactEmails

	doc := invoiceDoc{Header: header}

	total := decimal.Zero
	for i, line := range lines {
		price := parseDecimalOrZero(line.UnitPriceValue)
		invoiced := parseDecimalOrZero(line.QuantityValue)
		ordered := parseDecimalOrZero(line.OrderLineQtyOrdered)
		lineTotal := price.Mul(invoiced)
		total = total.Add(lineTotal)

		// Fall back to the row's position when the order line carries no number, so the column is
		// never blank.
		lineItem := fmt.Sprintf("%03d", i+1)
		if line.OrderLineItemNumber != nil {
			lineItem = fmt.Sprintf("%03d", *line.OrderLineItemNumber)
		}

		doc.Lines = append(doc.Lines, invoiceLineRow{
			LineItem:    lineItem,
			SKU:         ptrutil.Deref(line.OrderLineItemSKU),
			Description: ptrutil.Deref(line.OrderLineDescription),
			// The price is labelled with the rate's own denominator unit, as the dashboard's
			// RateUtils.abbreviate does — an item priced by the dozen and stocked in pairs reads
			// "$8.50 / dz", not "$8.50 / pr".
			Price:    formatRateAmount(price, line.UnitPriceDenUnitAbbr, 2),
			Ordered:  formatCount(ordered),
			Invoiced: formatCount(invoiced),
			// Rounded to whole units, matching the zero-digit default the legacy
			// EmailInvoiceItemSummary rendered through, but naming the unit in full rather than
			// abbreviating it.
			InvoicedWithUnit: formatMeasure(invoiced, line.QuantityUnitName, 0),
			Unit:             line.QuantityUnitName,
			Total:            formatMoney(lineTotal),
		})
	}
	doc.OrderTotal = formatMoney(total)
	doc.Header.OrderTotal = doc.OrderTotal

	for _, c := range cases {
		weight := ""
		if c.FreightWeightValue != "" {
			// Printed verbatim, as the legacy PdfShipCaseTable interpolates the measure directly
			// rather than abbreviating it.
			weight = formatRawMeasure(parseDecimalOrZero(c.FreightWeightValue), c.FreightWeightUnitAbbreviation)
		}
		doc.Cases = append(doc.Cases, invoiceCaseRow{
			Number:   c.Number,
			Weight:   weight,
			Tracking: ptrutil.Deref(c.TrackingNumber),
		})
	}

	return doc
}
