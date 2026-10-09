package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type echoTemplateRenderer struct{}

func (echoTemplateRenderer) RenderTemplate(_ context.Context, templateID constants.EmailTemplate, _ map[string]any) (string, *apierror.APIError) {
	return "rendered " + string(templateID), nil
}

// discardBroker accepts the email log events the consumer publishes after each send.
type discardBroker struct{ messaging.MessageBroker }

func (discardBroker) PublishMessage(context.Context, string, string, contracts.AmqpMessage) error {
	return nil
}

func followupDelivery(t *testing.T) amqp091.Delivery {
	t.Helper()

	from := "OpenMRP <dev@openmrp.ai>"
	payload := messaging.EmailSendData{
		To:         []string{"registrant@example.com"},
		Subject:    "Thanks for trying OpenMRP",
		TemplateID: constants.EmailTemplateAccountFollowup,
		Params:     map[string]any{"Body": "hi"},
		From:       &from,
		Bcc:        []string{"dev@openmrp.ai"},
		ThreadNote: &messaging.EmailThreadNote{
			To:         []string{"dev@openmrp.ai"},
			TemplateID: constants.EmailTemplateAccountFollowupContext,
		},
	}
	dataJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	body, err := json.Marshal(contracts.AmqpMessage{Data: dataJSON, MessageID: "mg_followup"})
	require.NoError(t, err)

	return amqp091.Delivery{RoutingKey: string(contracts.NotificationCmdSendEmail), Body: body}
}

// The note must reply to the message SES just accepted, so the reviewer's copy, the note, and the registrant's reply share one thread.
func TestHandleSendEmailThreadsNoteUnderSentEmail(t *testing.T) {
	consumer, svc := newTestConsumer(t)
	consumer.templateRenderer = echoTemplateRenderer{}
	consumer.rabbitmq = discardBroker{}

	var sends []domain.EmailSendData
	parentID, noteID := "ses-parent", "ses-note"
	svc.EXPECT().SendEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, data domain.EmailSendData) (*string, *apierror.APIError) {
			sends = append(sends, data)
			if len(sends) == 1 {
				return &parentID, nil
			}
			return &noteID, nil
		}).Times(2)

	require.NoError(t, consumer.handleCommandMessage(context.Background(), followupDelivery(t)))

	parent, note := sends[0], sends[1]
	require.True(t, parent.PlainText)
	require.Equal(t, []string{"dev@openmrp.ai"}, parent.Bcc)
	require.Equal(t, "OpenMRP <dev@openmrp.ai>", *parent.From)

	require.Equal(t, []string{"dev@openmrp.ai"}, note.To)
	require.Equal(t, "Re: Thanks for trying OpenMRP", note.Subject)
	require.Equal(t, parentID, *note.InReplyToSESMessageID)
	require.False(t, note.PlainText)
}

// Once the registrant's email is out, a failed note must not fail the message: a redelivery would send them the same email twice.
func TestHandleSendEmailAcksWhenThreadNoteFails(t *testing.T) {
	consumer, svc := newTestConsumer(t)
	consumer.templateRenderer = echoTemplateRenderer{}
	consumer.rabbitmq = discardBroker{}

	parentID := "ses-parent"
	gomock.InOrder(
		svc.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(&parentID, nil),
		svc.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, apierror.NewInternalError(nil, "ses down")),
	)

	require.NoError(t, consumer.handleCommandMessage(context.Background(), followupDelivery(t)))
}
