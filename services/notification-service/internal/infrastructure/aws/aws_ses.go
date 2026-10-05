package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/smithy-go"
	"go.opentelemetry.io/otel/codes"
)

var sesEmailSenderTracer = tracing.GetTracer("notification-service.aws.ses")

const (
	EmailSenderSource     = "noreply@augno.com"
	EmailTestingRecipient = "dev@augno.com"
)

func NewSESEmailSender(ctx context.Context, platformMode constants.PlatformMode, region string) (domain.EmailSender, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to load AWS configuration.")
	}

	return &sesEmailSenderImpl{
		client:       ses.NewFromConfig(cfg),
		platformMode: platformMode,
		region:       region,
	}, nil
}

type sesEmailSenderImpl struct {
	client       *ses.Client
	platformMode constants.PlatformMode
	region       string
}

func (s *sesEmailSenderImpl) Send(ctx context.Context, data domain.EmailData) (*string, *apierror.APIError) {
	ctx, span := sesEmailSenderTracer.Start(ctx, "aws.ses.send_email")
	defer span.End()

	if len(data.To) == 0 {
		err := apierror.NewMissingFieldError("At least one recipient must be provided.", "to")
		return nil, err
	}
	if data.Body == "" {
		err := apierror.NewMissingFieldError("Body must be provided.", "body")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	recipients := data.To
	cc := data.Cc
	bcc := data.Bcc
	if s.platformMode == constants.PlatformModeDevelopment {
		recipients = []string{EmailTestingRecipient}
		cc = nil
		bcc = nil
	}

	inReplyTo, references := data.InReplyTo, data.References
	if data.InReplyToSESMessageID != nil && *data.InReplyToSESMessageID != "" {
		ids := sesMessageIDHeaders(*data.InReplyToSESMessageID, s.region)
		inReplyTo = &ids[0]
		joined := strings.Join(ids, " ")
		references = &joined
	}

	// The bridge sends as the inbox address (a DKIM-verified customer domain); everything else sends as the default noreply@ sender.
	sender := EmailSenderSource
	if data.From != nil && *data.From != "" {
		sender = *data.From
	}

	rawMessage, err := generateRawEmail(rawEmailInput{
		Subject:     data.Subject,
		Body:        data.Body,
		Recipients:  recipients,
		Cc:          cc,
		Attachment:  data.Attachment,
		Filename:    data.Filename,
		SenderEmail: sender,
		IsHtml:      !data.PlainText,
		ReplyTo:     data.SendAs,
		InReplyTo:   inReplyTo,
		References:  references,
		MessageID:   data.MessageID,
	})

	if err != nil {
		apiErr := apierror.NewInternalError(err, "Failed to generate raw email.")
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, apiErr.Error())
		return nil, apiErr
	}

	destinations := append(append(append([]string{}, recipients...), cc...), bcc...)
	input := &ses.SendRawEmailInput{
		RawMessage: &sestypes.RawMessage{
			Data: rawMessage,
		},
		Source:       aws.String(sender),
		Destinations: destinations,
	}

	response, err := s.client.SendRawEmail(ctx, input)

	if err != nil {
		apiErr := classifySESSendError(err, sender)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, apiErr.Error())
		return nil, apiErr
	}

	return response.MessageId, nil
}

// classifySESSendError turns a SendRawEmail failure into an APIError that names the real cause. SES collapses several distinct problems into MessageRejected — the most common on this bridge is the sending identity (the inbox's From address/domain) not being verified for sending in this region, or the SES account still being in the sandbox (which only lets you send to verified recipients). Left as a bare "Failed to send email." internal error, these look identical to a transient outage, so the agent and the approving teammate can't tell that the fix is a config change (verify the domain / leave the sandbox) rather than a retry. We log the full SES code+message and put an actionable summary on the error the caller surfaces.
func classifySESSendError(err error, sender string) *apierror.APIError {
	var apiSESErr smithy.APIError
	if errors.As(err, &apiSESErr) {
		code := apiSESErr.ErrorCode()
		msg := apiSESErr.ErrorMessage()
		if code == "MessageRejected" || strings.Contains(strings.ToLower(msg), "not verified") {
			return apierror.NewInternalError(err, fmt.Sprintf("Email could not be sent from %q: %s. The sending domain likely isn't verified for sending in SES, or the SES account is still in the sandbox.", sender, msg))
		}
		return apierror.NewInternalError(err, fmt.Sprintf("Email could not be sent (SES %s): %s.", code, msg))
	}
	return apierror.NewInternalError(err, "Failed to send email.")
}

// sesMessageIDHeaders returns the Message-ID values SES may have stamped on the message it accepted as sesMessageID: SES discards any Message-ID we set and uses its own, on email.amazonses.com in us-east-1 and on the region's own subdomain elsewhere. Both forms are returned (the region-correct one first) because a References entry that matches nothing is ignored, while a single wrong guess would leave the reply unthreaded.
func sesMessageIDHeaders(sesMessageID, region string) []string {
	primary := sesMessageID + "@email.amazonses.com"
	if region != "" && region != "us-east-1" {
		return []string{sesMessageID + "@" + region + ".amazonses.com", primary}
	}
	return []string{primary}
}

// bracketReferences angle-brackets each whitespace-separated message-id in a References value (the ledger stores them bare), producing a valid rfc822 References header.
func bracketReferences(refs string) string {
	parts := strings.Fields(refs)
	for i, p := range parts {
		p = strings.Trim(p, "<>")
		parts[i] = "<" + p + ">"
	}
	return strings.Join(parts, " ")
}

type rawEmailInput struct {
	Subject     string
	Body        string
	Recipients  []string
	Cc          []string
	Attachment  []byte
	Filename    *string
	SenderEmail string
	IsHtml      bool
	ReplyTo     *string
	InReplyTo   *string
	References  *string
	MessageID   *string
}

func generateRawEmail(input rawEmailInput) ([]byte, error) {
	// Base64-encode and chunk the body to stay under the RFC 5322 line-length limit.
	base64Body := base64.StdEncoding.EncodeToString([]byte(input.Body))
	chunkedBody := splitString(base64Body, 76)

	var bodyContent string
	if input.IsHtml {
		bodyContent = fmt.Sprintf("Content-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", chunkedBody)
	} else {
		bodyContent = fmt.Sprintf("Content-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", chunkedBody)
	}

	// Prepare the attachment
	var attachmentPart string
	if len(input.Attachment) > 0 && input.Filename != nil {
		base64Attachment := base64.StdEncoding.EncodeToString(input.Attachment)
		chunkedAttachment := splitString(base64Attachment, 76)
		attachmentPart = fmt.Sprintf("\r\n--NextPart\r\n"+
			"Content-Type: application/octet-stream; name=\"%s\"\r\n"+
			"Content-Disposition: attachment; filename=\"%s\"\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"\r\n"+
			"%s\r\n", *input.Filename, *input.Filename, chunkedAttachment)
	}

	// Prepare headers
	headers := []string{
		fmt.Sprintf("From: %s", input.SenderEmail),
		fmt.Sprintf("To: %s", strings.Join(input.Recipients, ",")),
	}

	if len(input.Cc) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(input.Cc, ",")))
	}
	if input.ReplyTo != nil {
		headers = append(headers, fmt.Sprintf("Reply-To: %s", *input.ReplyTo))
	}
	// rfc822 threading: angle-bracket the message-ids so mail clients group the reply into the thread.
	if input.MessageID != nil && *input.MessageID != "" {
		headers = append(headers, fmt.Sprintf("Message-ID: <%s>", *input.MessageID))
	}
	if input.InReplyTo != nil && *input.InReplyTo != "" {
		headers = append(headers, fmt.Sprintf("In-Reply-To: <%s>", *input.InReplyTo))
	}
	if input.References != nil && *input.References != "" {
		headers = append(headers, fmt.Sprintf("References: %s", bracketReferences(*input.References)))
	}

	// Encoded so a non-ASCII subject arrives intact; ASCII passes through unchanged.
	headers = append(headers, fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", input.Subject)))
	headers = append(headers, "MIME-Version: 1.0")
	headers = append(headers, "Content-Type: multipart/mixed; boundary=\"NextPart\"")

	rawEmail := strings.Join(headers, "\r\n") + "\r\n\r\n" +
		"--NextPart\r\n" +
		bodyContent + "\r\n" +
		attachmentPart +
		"--NextPart--\r\n"

	return []byte(rawEmail), nil
}

func splitString(s string, chunkSize int) string {
	var chunks []string
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	for i := 0; i < len(runes); i += chunkSize {
		end := min(i+chunkSize, len(runes))
		chunks = append(chunks, string(runes[i:end]))
	}
	return strings.Join(chunks, "\r\n")
}
