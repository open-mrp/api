package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/augno/api/services/notification-service/internal/domain"
	servicemock "github.com/augno/api/services/notification-service/internal/domain/mock/service"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func newTestConsumer(t *testing.T) (*NotificationConsumer, *servicemock.MockNotificationSvc) {
	t.Helper()

	ctrl := gomock.NewController(t)
	svc := servicemock.NewMockNotificationSvc(ctrl)

	return &NotificationConsumer{
		notificationSvc: svc,
		tracer:          noop.NewTracerProvider().Tracer("test"),
	}, svc
}

// unrenderableTemplateRenderer reproduces the order_checkout outage: a template with no renderer entry fails to render, and the send dies before ever reaching SES.
type unrenderableTemplateRenderer struct{}

func (unrenderableTemplateRenderer) RenderTemplate(_ context.Context, templateID constants.EmailTemplate, _ map[string]any) (string, *apierror.APIError) {
	return "", apierror.NewInternalError(nil, "Template not found: "+string(templateID))
}

func emailSentDelivery(t *testing.T, data any) amqp091.Delivery {
	t.Helper()

	dataJSON, err := json.Marshal(data)
	require.NoError(t, err)

	body, err := json.Marshal(contracts.AmqpMessage{Data: dataJSON})
	require.NoError(t, err)

	return amqp091.Delivery{
		RoutingKey: string(contracts.NotificationEventEmailSent),
		Body:       body,
	}
}

// publishEmailStatus reuses the email_sent routing key that this queue binds, so the status payload lands here too. It has no ses_message_id, and logging it wrote a blank email_log row against every successful send.
func TestHandleEventMessageIgnoresStatusPayload(t *testing.T) {
	consumer, svc := newTestConsumer(t)

	svc.EXPECT().LogEmail(gomock.Any(), gomock.Any()).Times(0)

	statusPayload := map[string]any{"success": true, "user_id": nil}
	require.NoError(t, consumer.handleEventMessage(context.Background(), emailSentDelivery(t, statusPayload)))
}

// The order_checkout outage: the template failed to render, so the send returned an error before SES and wrote no email_log row at all. The email vanished with no trace in the activity list.
func TestHandleSendEmailLogsFailureWhenTemplateMissing(t *testing.T) {
	consumer, svc := newTestConsumer(t)
	consumer.templateRenderer = unrenderableTemplateRenderer{}

	accountID := "ac_REDACTED_ACCOUNT_ID"
	var loggedMessageID string
	var logged domain.EmailSendData
	svc.EXPECT().
		LogFailedEmail(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, messageID string, data domain.EmailSendData) {
			loggedMessageID = messageID
			logged = data
		}).
		Return(nil).
		Times(1)

	payload := messaging.EmailSendData{
		To:         []string{"kurt@example.com"},
		Subject:    "Your Order Checkout - 23124",
		TemplateID: constants.EmailTemplateOrderCheckout,
		AccountID:  &accountID,
	}
	dataJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	body, err := json.Marshal(contracts.AmqpMessage{Data: dataJSON, MessageID: "mg_nyf09gcfkl4wt1cjudpyh8"})
	require.NoError(t, err)

	err = consumer.handleCommandMessage(context.Background(), amqp091.Delivery{
		RoutingKey: string(contracts.NotificationCmdSendEmail),
		Body:       body,
	})
	require.Error(t, err, "the send must still fail so the message is retried, not silently acked")

	require.Equal(t, "mg_nyf09gcfkl4wt1cjudpyh8", loggedMessageID)
	require.Equal(t, "Your Order Checkout - 23124", logged.Subject)
	require.Equal(t, []string{"kurt@example.com"}, logged.To)
	require.Equal(t, &accountID, logged.AccountID)
}

func TestHandleEventMessageLogsRealLogEvent(t *testing.T) {
	consumer, svc := newTestConsumer(t)

	var logged domain.EmailLogData
	svc.EXPECT().
		LogEmail(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, data domain.EmailLogData) { logged = data }).
		Return(nil).
		Times(1)

	accountID := "ac_REDACTED_ACCOUNT_ID"
	logPayload := messaging.EmailLogData{
		SesMessageID: "010f019f-abc",
		AccountID:    &accountID,
		Subject:      "Your Order Checkout - 23124",
	}
	require.NoError(t, consumer.handleEventMessage(context.Background(), emailSentDelivery(t, logPayload)))

	require.Equal(t, "010f019f-abc", logged.SesMessageID)
	require.Equal(t, "Your Order Checkout - 23124", logged.Subject)
}
