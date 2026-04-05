package ws

import (
	"log/slog"
	"net/http"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/coder/websocket"
)

// NewHandler returns an http.HandlerFunc that upgrades HTTP connections to
// WebSocket, authenticates via cookie + account ID query param, and starts
// the client read/write pumps.
func NewHandler(hub *Hub, authClient *grpcclient.AuthServiceClient) http.HandlerFunc {
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
		_, err := authClient.Client.ValidateCredential(r.Context(), &pb.Credential{
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

		// Clear the server's write deadline so the long-lived WebSocket
		// connection doesn't get killed by the default httpWriteTimeout.
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

		client := NewClient(conn, hub, accountID)

		go client.WritePump(r.Context())
		client.ReadPump(r.Context())
	}
}
