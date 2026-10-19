package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/ptrutil"
)

// maxNoteMentions caps the people one note write can notify.
const maxNoteMentions = 25

// requireMentionable checks the mentioned account users are members of the account, so a note can't notify anyone
// outside it.
func requireMentionable(ctx context.Context, repos domain.RepoFactory, accountID string, mentions []string) *apierror.APIError {
	if len(mentions) == 0 {
		return nil
	}
	if len(mentions) > maxNoteMentions {
		return apierror.NewInvalidFormatError("mentions", fmt.Sprintf("A note can mention at most %d people.", maxNoteMentions))
	}
	found, apiErr := repos.NewAccountUserRepo().GetByIDs(ctx, accountID, mentions)
	if apiErr != nil {
		return apiErr
	}
	for i, m := range mentions {
		if !slices.ContainsFunc(found, func(d *domain.AccountUserDetail) bool { return d.ID == m }) {
			return apierror.NewInvalidFormatError(fmt.Sprintf("mentions[%d]", i), "No account user with this ID is in your account.")
		}
	}
	return nil
}

// notifyNoteMentions tells the people a note mentions (except its author) and subscribes them to the note's record.
// It writes to the outbox of the caller's transaction, so the notification goes out only if the note is saved.
func notifyNoteMentions(ctx context.Context, repos domain.RepoFactory, identity *types.Identity, note *domain.Note, mentions []string) *apierror.APIError {
	if len(mentions) == 0 {
		return nil
	}
	authorName := "Someone"
	if identity.Actor != nil && ptrutil.Deref(identity.Actor.Name) != "" {
		authorName = *identity.Actor.Name
	}
	body := note.Body
	if r := []rune(body); len(r) > 280 {
		body = string(r[:279]) + "…"
	}
	data := messaging.AlertFanoutData{
		AccountID:               note.AccountID,
		Category:                string(constants.NotificationCategoryNoteMention),
		Kind:                    "alert",
		Title:                   authorName + " mentioned you in a note",
		Body:                    body,
		Priority:                string(constants.NotificationPriorityHigh),
		SenderType:              string(constants.NotificationSenderTypeSystem),
		RecipientAccountUserIDs: slices.Clone(mentions),
	}
	if identity.HasUserActor() {
		authorAcus, apiErr := repos.NewAccountUserRepo().ResolveAccountUserID(ctx, note.AccountID, identity.Actor.ID)
		if apiErr != nil && !apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			return apiErr
		}
		data.RecipientAccountUserIDs = slices.DeleteFunc(data.RecipientAccountUserIDs, func(id string) bool { return id == authorAcus })
		if len(data.RecipientAccountUserIDs) == 0 {
			return nil
		}
		if name := ptrutil.Deref(identity.Actor.Name); name != "" && authorAcus != "" {
			data.SenderType, data.SenderID, data.SenderName = string(constants.NotificationSenderTypeUser), authorAcus, name
		}
	}
	if note.RecordType != nil && note.RecordID != nil {
		data.LinkResourceType, data.LinkResourceID = string(*note.RecordType), *note.RecordID
		data.Record = &messaging.RecordRef{Type: string(*note.RecordType), ID: *note.RecordID}
		for _, m := range data.RecipientAccountUserIDs {
			data.Subscribe = append(data.Subscribe, messaging.SubscriptionIntent{AccountUserID: m, Reason: string(constants.SubscriptionReasonMentioned)})
		}
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal the note mention fan-out.")
	}
	msg := contracts.AmqpMessage{Data: payload, Identity: identity}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue the note mention fan-out.")
	}
	return nil
}
