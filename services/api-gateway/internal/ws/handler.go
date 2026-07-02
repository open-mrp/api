package ws

import (
	"context"
	"log/slog"
	"net/http"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"
	notifpb "github.com/augno/api/shared/proto/notification"
	"github.com/coder/websocket"
)

// NewHandler returns an http.HandlerFunc that upgrades HTTP connections to WebSocket, authenticates via cookie + account ID query param, and starts the client read/write pumps. notificationClient may be nil (conversation-subscribe authz is then unavailable).
func NewHandler(hub *Hub, authClient *grpcclient.AuthServiceClient, notificationClient *grpcclient.NotificationServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract access token from cookie.
		accessToken, apiErr := cookie.GetAccessTokenFromRequest(r)
		if apiErr != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract account ID from query param.
		accountID := r.URL.Query().Get("accountId")
		if accountID == "" {
			http.Error(w, "accountId query param required", http.StatusBadRequest)
			return
		}

		// Validate via gRPC auth-service (same as AuthMiddleware).
		identity, err := authClient.Client.ValidateCredential(r.Context(), &pb.Credential{
			Token:           accessToken,
			TargetAccountId: &accountID,
		})
		if err != nil {
			apiErr := contracts.ConvertGRPCError(r.Context(), err, "auth-service")
			if apiErr != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// The actor id is the user id (us_), used as the key for the per-user notification topic.
		userID := identity.GetActor().GetId()

		// Reject actors with no personal user identity (e.g. api-key actors): they have no bell feed and no account_user to authorize conversation subscriptions, so the socket would only ever subscribe to empty-keyed topics. Closing here avoids a useless, mis-keyed connection.
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Clear the server's write deadline so the long-lived WebSocket connection doesn't get killed by the default httpWriteTimeout.
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(zeroTime); err != nil {
			slog.Error("Failed to clear write deadline for WS", "error", err)
		}

		// Accept WebSocket upgrade.
		conn, wsErr := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // CORS handled at proxy layer
		})
		if wsErr != nil {
			slog.Error("WebSocket accept failed", "error", wsErr)
			return
		}

		// Build the conversation-subscribe authz gate from the connection's user + account.
		var checkParticipant ParticipantChecker
		if notificationClient != nil {
			checkParticipant = func(ctx context.Context, conversationID string) (bool, error) {
				resp, err := notificationClient.ChatClient.IsParticipant(ctx, &notifpb.IsParticipantRequest{
					ConversationId: conversationID,
					UserId:         userID,
					AccountId:      accountID,
				})
				if err != nil {
					return false, err
				}
				return resp.GetIsParticipant(), nil
			}
		}

		client := NewClient(conn, hub, accountID, userID, checkParticipant)
		// Auto-subscribe on connect: the per-user bell topic, the account broadcast topic (announcements), and the account-independent user topic (cross-account unread hints).
		client.SubscribeUserTopic()
		client.SubscribeAccountTopic()
		client.SubscribeUserGlobalTopic()

		go client.WritePump(r.Context())
		client.ReadPump(r.Context())
	}
}
