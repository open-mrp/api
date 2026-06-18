package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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

func NewSESEmailSender(ctx context.Context, platformMode constants.PlatformMode, region string) (domain.EmailSender, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to load AWS configuration.")
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
	if s.platformMode == constants.PlatformModeDevelopment {
		recipients = []string{EmailTestingRecipient}
	}

	rawMessage, err := generateRawEmail(rawEmailInput{
		Subject:     data.Subject,
		Body:        data.Body,
		Recipients:  recipients,
		Attachment:  data.Attachment,
		Filename:    data.Filename,
		SenderEmail: EmailSenderSource,
		IsHtml:      true,
		ReplyTo:     data.SendAs,
	})

	if err != nil {
		apiErr := apierror.NewInternalError(err, "Failed to generate raw email.")
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
		apiErr := apierror.NewInternalError(err, "Failed to send email.")
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, apiErr.Error())
		return nil, apiErr
	}

	return response.MessageId, nil
}

type rawEmailInput struct {
	Subject     string
	Body        string
	Recipients  []string
	Attachment  []byte
	Filename    *string
	SenderEmail string
	IsHtml      bool
	ReplyTo     *string
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

	if input.ReplyTo != nil {
		headers = append(headers, fmt.Sprintf("Reply-To: %s", *input.ReplyTo))
	}

	headers = append(headers, fmt.Sprintf("Subject: %s", input.Subject))
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
