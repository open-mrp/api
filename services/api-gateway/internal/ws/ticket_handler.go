package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"
)

// NewTicketHandler returns an http.HandlerFunc that mints short-lived WebSocket tickets for cookie-authenticated callers. Custom portal domains reach it through the frontend's same-origin API proxy (so the auth cookie is present), then open the WebSocket cross-origin to the API host with the ticket instead of the cookie. When ticketSecret is empty the endpoint is disabled.
func NewTicketHandler(authClient *grpcclient.AuthServiceClient, ticketSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(ticketSecret) == 0 {
			http.Error(w, "ws tickets are not configured", http.StatusNotImplemented)
			return
		}

		accessToken, apiErr := cookie.GetAccessTokenFromRequest(r)
		if apiErr != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		accountID := r.URL.Query().Get("accountId")
		if accountID == "" {
			http.Error(w, "accountId query param required", http.StatusBadRequest)
			return
		}

		identity, err := authClient.Client.ValidateCredential(r.Context(), &pb.Credential{
			Token:           accessToken,
			TargetAccountId: &accountID,
		})
		if err != nil {
			if apiErr := contracts.ConvertGRPCError(r.Context(), err, "auth-service"); apiErr != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		userID := identity.GetActor().GetId()
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ticket, expiresAt, err := MintTicket(ticketSecret, userID, accountID, time.Now())
		if err != nil {
			slog.Error("Failed to mint WS ticket", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ticket":     ticket,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		})
	}
}
