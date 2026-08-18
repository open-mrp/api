package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"  // register GIF decoder for image.Decode
	_ "image/jpeg" // register JPEG decoder for image.Decode
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/textutil"
	"github.com/shopspring/decimal"
	_ "golang.org/x/image/webp" // register WEBP decoder (account logos are often .webp)
)

// ackAddress is a rendered address block for the acknowledgement email/PDF.
type ackAddress struct {
	Name         string
	Line1        string
	Line2        string
	CityStateZip string
	Phone        string
	Email        string
}

// Empty reports whether the address has no displayable content.
func (a ackAddress) Empty() bool {
	return a.Name == "" && a.Line1 == "" && a.Line2 == "" && a.CityStateZip == "" && a.Phone == "" && a.Email == ""
}

// ackLine is a single rendered order line for the acknowledgement email/PDF.
type ackLine struct {
	LineItem    string
	SKU         string
	Description string
	Price       string
	Qty         string
	Total       string
}

// ackData is the shared view model for the order-acknowledgement email and PDF.
// It carries pre-formatted, presentation-ready strings so the email template and
// the PDF renderer stay in lockstep with the legacy Dashboard layout.
type ackData struct {
	// Seller letterhead / branding.
	AccountName    string
	LogoURL        string
	AccountAddress ackAddress
	AccountEmail   string
	AccountPhone   string
	AccountWebsite string

	// LogoImage / LogoImageType hold the fetched logo bytes for embedding in the
	// PDF (the email references LogoURL directly). Populated best-effort by the
	// caller; empty when unavailable.
	LogoImage     []byte
	LogoImageType string

	// Order identity.
	OrderNumber    string
	CustomerPO     string
	CustomerNumber string
	CustomerName   string
	OrderDateShort string // e.g. 7/14/2026 (email)
	OrderDateLong  string // e.g. 07/14/2026 (PDF)

	// OrderOnlineLink is the customer-portal registration URL, set only for
	// accounts with a portal configured; empty otherwise (no CTA is rendered).
	OrderOnlineLink string

	// Parties.
	BillTo    ackAddress
	ShipTo    ackAddress
	HasShipTo bool
	// ContactEmails are the order's acknowledgement recipient emails, shown under
	// the Bill To block (mirrors the legacy PDF). Set by the caller.
	ContactEmails []string

	// Order terms.
	Carrier      string
	Priority     string
	PaymentTerms string
	SalesRep     string

	// Lines + total.
	Lines      []ackLine
	OrderTotal string

	Year string
}

// buildOrderAcknowledgementData assembles the shared acknowledgement view model
// from the order, its lines, the seller account (branding), and the seller's
// origin address. account and originAddr are optional; nil values degrade to an
// empty letterhead rather than failing.
func buildOrderAcknowledgementData(order *domain.SalesOrder, lines []*domain.SalesOrderLine, account *domain.Account, originAddr *domain.ShippingAddress) ackData {
	d := ackData{
		AccountName:    accountDisplayName(account, order.CustomerName),
		OrderNumber:    textutil.FormatRecordNumber(order.Number),
		CustomerPO:     ptrutil.Deref(order.CustomerPONumber),
		CustomerNumber: textutil.FormatRecordNumber(order.CustomerNumber),
		CustomerName:   order.CustomerName,
		OrderDateShort: order.CreatedAt.Format("1/2/2006"),
		OrderDateLong:  order.CreatedAt.Format("01/02/2006"),
		Carrier:        ackCarrier(order),
		Priority:       order.PriorityName,
		PaymentTerms:   ptrutil.Deref(order.PaymentTermName),
		SalesRep:       ptrutil.Deref(order.SalesRepName),
		Year:           order.CreatedAt.Format("2006"),
	}

	if account != nil && account.Branding != nil {
		d.LogoURL = ptrutil.Deref(account.Branding.LogoURL)
		d.AccountEmail = ptrutil.Deref(account.Branding.SupportEmail)
		d.AccountPhone = ptrutil.Deref(account.Branding.PhoneNumber)
		d.AccountWebsite = ptrutil.Deref(account.Branding.WebsiteURL)
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

	total := decimal.Zero
	for _, line := range lines {
		price := parseDecimalOrZero(line.UnitPriceValue)
		qty := parseDecimalOrZero(line.QuantityValue)
		lineTotal := price.Mul(qty)
		total = total.Add(lineTotal)

		d.Lines = append(d.Lines, ackLine{
			LineItem:    fmt.Sprintf("%03d", line.LineItemNumber),
			SKU:         line.ProductSKU,
			Description: ptrutil.Deref(line.ProductDescription),
			Price:       formatPrice(price, line.UnitPriceDenominatorUnitAbbr),
			Qty:         formatQty(qty, line.QuantityUnitName),
			Total:       formatMoney(lineTotal),
		})
	}
	d.OrderTotal = formatMoney(total)

	return d
}

// emailParams flattens the view model into the template params passed through the
// outbox to the notification-service email template. Nested slices/maps survive
// the JSON round-trip and are consumed by html/template's range/index.
func (d ackData) emailParams() map[string]any {
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
		"account_name":      d.AccountName,
		"logo_url":          d.LogoURL,
		"order_number":      d.OrderNumber,
		"customer_po":       d.CustomerPO,
		"order_date":        d.OrderDateShort,
		"order_total":       d.OrderTotal,
		"has_ship_to":       d.HasShipTo,
		"ship_to_name":      d.ShipTo.Name,
		"ship_to_line1":     d.ShipTo.Line1,
		"ship_to_line2":     d.ShipTo.Line2,
		"ship_to_csz":       d.ShipTo.CityStateZip,
		"lines":             lines,
		"account_email":     d.AccountEmail,
		"account_website":   d.AccountWebsite,
		"year":              d.Year,
		"customer_number":   d.CustomerNumber,
		"order_online_link": d.OrderOnlineLink,
	}
}

// buildOrderAcknowledgementEmail assembles the order-acknowledgement email — the rendered template params plus the generated PDF attachment — for an order. Returns (nil, nil) when the order has no acknowledgement recipients so callers can no-op instead of sending. Shared by the automatic send on issue and the manual resend, so both produce an identical email.
func buildOrderAcknowledgementEmail(ctx context.Context, repos domain.RepoFactory, frontendURL, accountID, salesOrderID string) (*messaging.EmailSendData, *apierror.APIError) {
	repo := repos.NewSalesOrderRepo()

	recipients, apiErr := repo.GetAcknowledgementRecipients(ctx, salesOrderID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(recipients) == 0 {
		return nil, nil
	}

	order, apiErr := repo.Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return nil, apiErr
	}
	lines, apiErr := repo.GetLines(ctx, salesOrderID)
	if apiErr != nil {
		return nil, apiErr
	}

	// The seller's branding (logo, contact) and origin address power the email
	// letterhead/footer and the PDF letterhead. Both are best-effort: the
	// acknowledgement still sends with a blank letterhead if either lookup fails.
	account, _ := repos.NewAccountRepo().GetByID(ctx, accountID)
	originAddr, _ := repo.GetAccountOriginAddress(ctx, accountID)

	data := buildOrderAcknowledgementData(order, lines, account, originAddr)
	// The acknowledgement recipients are shown as contact emails under Bill To.
	data.ContactEmails = recipients
	// Gate the "Order Online" CTA on the account having a customer portal.
	data.OrderOnlineLink = portalRegisterLink(ctx, repos, frontendURL, accountID)
	// Fetch the logo bytes for the PDF letterhead (best-effort; the email uses the URL).
	if data.LogoURL != "" {
		data.LogoImageType, data.LogoImage = fetchAckLogoImage(ctx, data.LogoURL)
	}

	emailData := &messaging.EmailSendData{
		To:         recipients,
		Subject:    fmt.Sprintf("Sales Order %s", data.OrderNumber),
		TemplateID: constants.EmailTemplateOrderAcknowledgement,
		Params:     data.emailParams(),
		AccountID:  &accountID,
	}

	// Attach the generated order-acknowledgement PDF, matching legacy (which attached a
	// rendered PDF of the order). A PDF failure degrades gracefully to an attachment-free
	// email rather than blocking the acknowledgement.
	if pdfBytes, err := buildOrderAcknowledgementPDF(data); err == nil {
		encoded := base64.StdEncoding.EncodeToString(pdfBytes)
		filename := ackAttachmentFilename(data.OrderNumber, data.CustomerPO)
		contentType := "application/pdf"
		emailData.AttachmentData = &encoded
		emailData.AttachmentFilename = &filename
		emailData.AttachmentContentType = &contentType
	}

	return emailData, nil
}

// portalRegisterLink returns the customer-portal registration URL for the account, or "" when the account has no customer portal configured. A verified custom portal domain is targeted directly; otherwise the slug-prefixed frontend URL is used. Best-effort: lookup failures yield "".
func portalRegisterLink(ctx context.Context, repos domain.RepoFactory, frontendURL, accountID string) string {
	// Path mirrors the frontend's FrontendPaths.register ("/auth/register").
	const registerPath = "/auth/register"
	if portalDomain, _ := repos.NewPortalDomainRepo().GetByAccountID(ctx, accountID); portalDomain != nil && portalDomain.Status == constants.PortalDomainStatusVerified {
		return "https://" + portalDomain.Domain + registerPath
	}
	slug, _ := repos.NewAccountRepo().GetPortalSlug(ctx, accountID)
	if slug != nil && strings.TrimSpace(*slug) != "" && frontendURL != "" {
		return fmt.Sprintf("%s/%s%s", frontendURL, *slug, registerPath)
	}
	return ""
}

// fetchAckLogoImage downloads the account logo and normalizes it to PNG bytes for
// embedding in the PDF (the email references the URL directly). It decodes any
// supported source format — PNG, JPEG, GIF, or WEBP (account logos are commonly
// .webp, which fpdf cannot embed directly) — and re-encodes to PNG. Best-effort:
// any failure (network, timeout, non-200, undecodable) returns ("", nil) and the
// PDF falls back to a text-only letterhead.
func fetchAckLogoImage(ctx context.Context, url string) (imageType string, data []byte) {
	if strings.TrimSpace(url) == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil
	}
	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL from account branding logo stored server-side

	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil || len(body) == 0 {
		return "", nil
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return "", nil
	}
	// Flatten any transparency onto white before encoding: fpdf renders alpha PNGs
	// unreliably (logos come out faint/washed-out), and the PDF background is white.
	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)
	var out bytes.Buffer
	if err := png.Encode(&out, flat); err != nil {
		return "", nil
	}
	return "PNG", out.Bytes()
}

func accountDisplayName(account *domain.Account, fallback string) string {
	if account != nil && strings.TrimSpace(account.Name) != "" {
		return account.Name
	}
	return fallback
}

// ackCarrier renders the "Ship Via" value: carrier name, with the service level in
// parentheses when present (mirrors the legacy "carrier (option)" format).
func ackCarrier(order *domain.SalesOrder) string {
	carrier := ptrutil.Deref(order.CarrierName)
	if carrier == "" {
		return ""
	}
	if sl := ptrutil.Deref(order.ServiceLevelName); sl != "" {
		return carrier + " (" + sl + ")"
	}
	return carrier
}

func joinCityStateZip(city, state, zip string) string {
	cityState := strings.TrimSpace(strings.Join(nonEmpty(city, state), ", "))
	if zip != "" {
		cityState = strings.TrimSpace(cityState + " " + zip)
	}
	return cityState
}

// formatMoney renders a decimal as USD with a thousands separator and two
// decimals, e.g. 1234.5 -> "$1,234.50".
func formatMoney(d decimal.Decimal) string {
	neg := d.IsNegative()
	s := d.Abs().StringFixed(2)
	intPart, frac, _ := strings.Cut(s, ".")
	out := "$" + addThousandsSep(intPart) + "." + frac
	if neg {
		out = "-" + out
	}
	return out
}

// formatPrice renders a unit price with its pricing unit appended, e.g.
// (8.5, "pr") -> "$8.50 / pr". The price unit (the rate's denominator) can differ
// from the quantity unit, so it is shown explicitly. The unit is omitted when blank.
func formatPrice(price decimal.Decimal, unitAbbr string) string {
	s := formatMoney(price)
	if strings.TrimSpace(unitAbbr) != "" {
		s += " / " + unitAbbr
	}
	return s
}

// formatQty renders a quantity with a thousands separator, trailing zeros trimmed,
// followed by the unit abbreviation, e.g. (1200, "EA") -> "1,200 EA".
func formatQty(qty decimal.Decimal, unitAbbr string) string {
	s := qty.String()
	intPart, frac, hasFrac := strings.Cut(s, ".")
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")
	out := addThousandsSep(intPart)
	if hasFrac {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	if unitAbbr != "" {
		out += " " + unitAbbr
	}
	return out
}

// ackAttachmentFilename names the emailed acknowledgement PDF, tagging the customer's
// PO number when the order has one so the customer can file it against their PO.
func ackAttachmentFilename(orderNumber, customerPO string) string {
	name := "order-acknowledgement-" + orderNumber
	if po := filenameSafe(customerPO); po != "" {
		name += "-PO-" + po
	}
	return name + ".pdf"
}

// filenameSafe reduces free text (e.g. a customer PO number) to a token safe in an
// attachment filename; runs of unsafe characters collapse to a single "-".
func filenameSafe(s string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range s {
		safe := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_'
		if !safe {
			pendingDash = b.Len() > 0
			continue
		}
		if pendingDash {
			b.WriteByte('-')
			pendingDash = false
		}
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), ".-")
}

func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseDecimalOrZero(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func addThousandsSep(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		b.WriteByte(',')
	}
	for i := pre; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}
