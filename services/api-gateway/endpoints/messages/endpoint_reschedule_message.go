package messageep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to move a scheduled message to a new send time.
type RescheduleMessageRequest struct {
	// The id of the scheduled message.
	MessageID string `path:"id" validate:"required"`
	// When the message should now be sent. Must be in the future.
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
	// The revised message body, replacing what it said before.
	//
	// Leaving it out keeps the current body.
	Body field.Optional[string] `json:"body,omitzero"`
}

var sampleRescheduleMessageBody = "Reminder: the line goes down for maintenance at 6pm."

var sampleRescheduleMessageRequest = &RescheduleMessageRequest{
	MessageID:   apiresource.SampleMessageID,
	ScheduledAt: time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC),
	Body:        field.Some(sampleRescheduleMessageBody),
}

func (*RescheduleMessageRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRescheduleMessageRequest)
}

// Moves a message you scheduled to a new send time, optionally revising what it says, and returns it.
//
// The message keeps its id and is sent once, at the new time; the time it had before no longer applies. You can only reschedule a message you scheduled yourself, and only until its send time arrives — once it is due, sent or canceled the request fails.
type RescheduleMessageEndpoint struct{}

func (e *RescheduleMessageEndpoint) Materialize() *apiendpoint.APIEndpoint[*RescheduleMessageRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*RescheduleMessageRequest, *apiresource.Message]{
		Title:               "Reschedule Message",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/messages/{id}/actions/reschedule",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RescheduleMessageRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).RescheduleMessage
		},
	})
}
