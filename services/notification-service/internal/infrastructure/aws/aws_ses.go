package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"go.opentelemetry.io/otel/codes"
)

var sesEmailSenderTracer = tracing.GetTracer("notification-service.aws.ses")

const (
	EmailSenderSource     = "noreply@augno.com"
	EmailTestingRecipient = "dev@augno.com"
)

func NewSESEmailSender(ctx context.Context, platformMode constants.PlatformMode, region string) (domain.EmailSender, *contracts.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, contracts.NewInternalError(err, "Failed to load AWS configuration.")
	}

	return &sesEmailSenderImpl{
		client:       ses.NewFromConfig(cfg),
		platformMode: platformMode,
	}, nil
}

type sesEmailSenderImpl struct {
	client       *ses.Client
	platformMode constants.PlatformMode
}

func (s *sesEmailSenderImpl) Send(ctx context.Context, to []string, subject, body string, isBodyHtml bool, sendAs *string) (*string, *contracts.APIError) {
	ctx, span := sesEmailSenderTracer.Start(ctx, "aws.ses.send_email")
	defer span.End()

	if len(to) == 0 {
		err := contracts.NewValidationError("At least one recipient must be provided.")
		return nil, err
	}
	if body == "" {
		err := contracts.NewValidationError("Body must be provided.")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	recipients := to
	if s.platformMode == constants.PlatformModeDevelopment {
		recipients = []string{EmailTestingRecipient}
	}

	rawMessage, err := generateRawEmail(subject, body, recipients, nil, "", EmailSenderSource, isBodyHtml, sendAs)
	if err != nil {
		apiErr := contracts.NewInternalError(err, "Failed to generate raw email.")
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, apiErr.Error())
		return nil, apiErr
	}

	input := &ses.SendRawEmailInput{
		RawMessage: &sestypes.RawMessage{
			Data: rawMessage,
		},
		Source:       aws.String(EmailSenderSource),
		Destinations: recipients,
	}

	response, err := s.client.SendRawEmail(ctx, input)

	if err != nil {
		apiErr := contracts.NewInternalError(err, "Failed to send email.")
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, apiErr.Error())
		return nil, apiErr
	}

	return response.MessageId, nil
}

func generateRawEmail(subject, body string, recipients []string, attachment []byte, filename, senderEmail string, isHtml bool, replyTo *string) ([]byte, error) {
	// Prepare the body content
	// Base64 encode and chunk the body to avoid line length limits (RFC 5322)
	base64Body := base64.StdEncoding.EncodeToString([]byte(body))
	chunkedBody := splitString(base64Body, 76)

	var bodyContent string
	if isHtml {
		bodyContent = fmt.Sprintf("Content-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", chunkedBody)
	} else {
		bodyContent = fmt.Sprintf("Content-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", chunkedBody)
	}

	// Prepare the attachment
	var attachmentPart string
	if len(attachment) > 0 && filename != "" {
		base64Attachment := base64.StdEncoding.EncodeToString(attachment)
		chunkedAttachment := splitString(base64Attachment, 76)
		attachmentPart = fmt.Sprintf("\r\n--NextPart\r\n"+
			"Content-Type: application/octet-stream; name=\"%s\"\r\n"+
			"Content-Disposition: attachment; filename=\"%s\"\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"\r\n"+
			"%s\r\n", filename, filename, chunkedAttachment)
	}

	// Prepare headers
	headers := []string{
		fmt.Sprintf("From: %s", senderEmail),
		fmt.Sprintf("To: %s", strings.Join(recipients, ",")),
	}

	if replyTo != nil {
		headers = append(headers, fmt.Sprintf("Reply-To: %s", *replyTo))
	}

	headers = append(headers, fmt.Sprintf("Subject: %s", subject))
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
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return strings.Join(chunks, "\r\n")
}
