package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/textutil"
	"github.com/open-mrp/api/shared/tracing"
)

var documentEmailTracer = tracing.GetTracer("core-service.service.document_email")

// invoiceEmail is a rendered invoice awaiting its recipients. The body and the PDF are identical
// whoever receives it, so the document is assembled once and addressed as many times as the caller
// needs — a ship mails the customer and the sales rep from one render.
type invoiceEmail struct {
	accountID  string
	subject    string
	params     map[string]any
	replyTo    string
	attachment *string
	filename   string
}

// buildInvoiceEmail assembles the invoice email — template params plus the generated PDF — for an
// invoice. The PDF is best-effort: a render failure sends the email without it rather than
// withholding the invoice.
func buildInvoiceEmail(ctx context.Context, repos domain.RepoFactory, branding BrandingAssets, accountID, invoiceID string) (*invoiceEmail, *apierror.APIError) {
	invoiceRepo := repos.NewInvoiceRepo()

	invoice, apiErr := invoiceRepo.Get(ctx, domain.GetInvoiceParams{AccountID: accountID, InvoiceID: invoiceID})
	if apiErr != nil {
		return nil, apiErr
	}

	// The document backs both the email body and its PDF, so assemble it once. Its lookups are
	// best-effort, so fall back to the account name when the letterhead came back blank.
	lines, _ := invoiceRepo.GetLines(ctx, invoiceID)
	doc := gatherInvoiceDoc(ctx, repos, accountID, invoice, lines)
	// The PDF embeds the letterhead logo, so its bytes are fetched here rather than inside the
	// transaction the caller opens: a slow logo host must not hold the invoice's row locks.
	logo := fetchAccountLogo(ctx, repos, branding, accountID)
	doc.Header.LogoImageType, doc.Header.LogoImage = logo.ImageType, logo.Image

	var shipment *domain.Shipment
	if invoice.ShipmentID != nil {
		shipment, _ = repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{AccountID: accountID, ShipmentID: *invoice.ShipmentID})
	}

	params := doc.emailParams(shipmentMasterTrackingURL(shipment))
	if params["account_name"] == "" {
		accountName, apiErr := repos.NewAccountRepo().GetName(ctx, accountID)
		if apiErr != nil {
			return nil, apiErr
		}
		params["account_name"] = accountName
	}
	if params["invoice_number"] == "" {
		params["invoice_number"] = textutil.FormatRecordNumber(invoice.Number)
	}

	out := &invoiceEmail{
		accountID: accountID,
		subject:   fmt.Sprintf("Invoice %s", textutil.FormatRecordNumber(invoice.Number)),
		params:    params,
		replyTo:   doc.Header.AccountEmail,
	}

	if pdfBytes, err := buildInvoicePDF(doc); err == nil {
		encoded := base64.StdEncoding.EncodeToString(pdfBytes)
		out.attachment = &encoded
		out.filename = fmt.Sprintf("invoice-%s.pdf", invoice.Number)
	}

	return out, nil
}

// addressedTo produces the send command for one set of recipients.
func (e *invoiceEmail) addressedTo(recipients []string) messaging.EmailSendData {
	data := messaging.EmailSendData{
		To:         recipients,
		Subject:    e.subject,
		TemplateID: constants.EmailTemplateInvoice,
		Params:     e.params,
		AccountID:  &e.accountID,
	}
	applyMerchantReplyTo(&data, e.replyTo)

	if e.attachment != nil {
		contentType := "application/pdf"
		data.AttachmentData = e.attachment
		data.AttachmentFilename = &e.filename
		data.AttachmentContentType = &contentType
	}

	return data
}

// SendInvoiceEmail mails an invoice in reaction to CoreEventInvoiceIssued.
//
// It takes the account explicitly and runs no idempotency key: a consumer has no actor whose
// permissions could be checked — the publisher authorized the action that raised the invoice — and
// the inbox already deduplicates redelivery.
func (s *utilsSvcImpl) SendInvoiceEmail(ctx context.Context, params domain.SendInvoiceEmailParams) *apierror.APIError {
	ctx, span := documentEmailTracer.Start(ctx, "service.document_email.send_invoice")
	defer span.End()

	if !params.EmailCustomer && !params.EmailSalesRep {
		return nil
	}

	invoiceRepo := s.repos.NewInvoiceRepo()

	var customers []string
	if params.EmailCustomer {
		recipients, apiErr := invoiceRepo.GetEmailRecipients(ctx, params.InvoiceID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		customers = recipients
	}

	var salesRep []string
	if params.EmailSalesRep {
		invoice, apiErr := invoiceRepo.Get(ctx, domain.GetInvoiceParams{AccountID: params.AccountID, InvoiceID: params.InvoiceID})
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		email, apiErr := s.repos.NewSalesOrderRepo().GetSalesRepEmail(ctx, params.AccountID, invoice.OrderID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if email != nil {
			salesRep = []string{*email}
		}
	}

	// An invoice whose customer wants no copy is still settled: flagging it sent is what stops the
	// nightly resend sweep offering it again, so that runs even with nobody to mail.
	if len(customers) == 0 && len(salesRep) == 0 {
		if !params.EmailCustomer {
			return nil
		}
		return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			return txSvc.repos.NewInvoiceRepo().MarkEmailSent(txCtx, params.AccountID, params.InvoiceID)
		})
	}

	built, apiErr := buildInvoiceEmail(ctx, s.repos, s.branding, params.AccountID, params.InvoiceID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if len(salesRep) > 0 {
			if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, built.addressedTo(salesRep)); apiErr != nil {
				return apiErr
			}
		}

		if !params.EmailCustomer {
			return nil
		}

		if len(customers) > 0 {
			if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, built.addressedTo(customers)); apiErr != nil {
				return apiErr
			}
		}

		// Only the customer copy flags the invoice sent — the flag records whether the customer
		// received it, and the rep copy is an internal notification.
		return txSvc.repos.NewInvoiceRepo().MarkEmailSent(txCtx, params.AccountID, params.InvoiceID)
	})
}

// SendSalesOrderAcknowledgement mails an order acknowledgement in reaction to CoreEventSalesOrderAcknowledged.
func (s *utilsSvcImpl) SendSalesOrderAcknowledgement(ctx context.Context, params domain.SendSalesOrderAcknowledgementParams) *apierror.APIError {
	ctx, span := documentEmailTracer.Start(ctx, "service.document_email.send_sales_order_acknowledgement")
	defer span.End()

	// Built by the same assembler the automatic send-on-issue uses, so this delivers an identical
	// acknowledgement (line items, letterhead, PDF attachment).
	emailData, apiErr := buildOrderAcknowledgementEmail(ctx, s.repos, s.branding, s.frontendURL, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if emailData == nil {
		return nil
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, *emailData); apiErr != nil {
			return apiErr
		}
		return txSvc.repos.NewSalesOrderRepo().MarkAcknowledgementSent(txCtx, params.AccountID, params.SalesOrderID)
	})
}

// SendPurchaseOrderSubmission mails a purchase order to its supplier in reaction to CoreEventPurchaseOrderSubmitted.
func (s *utilsSvcImpl) SendPurchaseOrderSubmission(ctx context.Context, params domain.SendPurchaseOrderSubmissionParams) *apierror.APIError {
	ctx, span := documentEmailTracer.Start(ctx, "service.document_email.send_purchase_order_submission")
	defer span.End()

	emailData, apiErr := buildPurchaseOrderSubmissionEmail(ctx, s.repos, s.branding, params.AccountID, params.PurchaseOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if emailData == nil {
		return nil
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, *emailData); apiErr != nil {
			return apiErr
		}
		return txSvc.repos.NewPurchaseOrderRepo().MarkSubmissionSent(txCtx, params.AccountID, params.PurchaseOrderID)
	})
}

// buildPurchaseOrderSubmissionEmail assembles the supplier submission email — the rendered template
// params plus the generated purchase order PDF — for a purchase order. Returns (nil, nil) when the
// order has no submission recipients so callers can no-op instead of sending.
func buildPurchaseOrderSubmissionEmail(ctx context.Context, repos domain.RepoFactory, branding BrandingAssets, accountID, purchaseOrderID string) (*messaging.EmailSendData, *apierror.APIError) {
	recipients, apiErr := repos.NewPurchaseOrderRepo().GetSubmissionRecipients(ctx, purchaseOrderID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(recipients) == 0 {
		return nil, nil
	}

	doc, apiErr := gatherPurchaseOrderDoc(ctx, repos, accountID, purchaseOrderID, recipients)
	if apiErr != nil {
		return nil, apiErr
	}

	// The PDF embeds the letterhead logo, so its bytes are fetched here rather than inside the
	// transaction the caller opens: a slow logo host must not hold the order's row locks.
	logo := fetchAccountLogo(ctx, repos, branding, accountID)
	doc.Header.LogoImageType, doc.Header.LogoImage = logo.ImageType, logo.Image

	emailData := &messaging.EmailSendData{
		To:         recipients,
		Subject:    fmt.Sprintf("Purchase Order %s", doc.Header.OrderNumber),
		TemplateID: constants.EmailTemplatePurchaseOrderSubmission,
		Params:     doc.emailParams(),
		AccountID:  &accountID,
	}
	applyMerchantReplyTo(emailData, doc.Header.AccountEmail)

	// Attach the generated purchase order PDF, matching legacy. A render failure degrades to an
	// attachment-free email rather than withholding the order from the supplier.
	if pdfBytes, err := buildPurchaseOrderPDF(doc.Header); err == nil {
		encoded := base64.StdEncoding.EncodeToString(pdfBytes)
		filename := purchaseOrderAttachmentFilename(doc.Header.OrderNumber)
		contentType := "application/pdf"
		emailData.AttachmentData = &encoded
		emailData.AttachmentFilename = &filename
		emailData.AttachmentContentType = &contentType
	}

	return emailData, nil
}
