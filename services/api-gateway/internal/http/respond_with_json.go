package httptransport

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
)

func RespondWithJSON(ctx context.Context, w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	rl, ok := apicontext.GetRequestLogFromContext(ctx)
	if ok && rl != nil {
		w.Header().Set("Request-ID", rl.ID)
	}
	var dat []byte
	var err error

	if ok && rl.UserAgent != nil && strings.HasPrefix(*rl.UserAgent, "curl/") {
		dat, err = json.MarshalIndent(payload, "", "  ")
	} else {
		dat, err = json.Marshal(payload)
	}

	if code == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	if _, err := w.Write(dat); err != nil {
		log.Printf("Error writing response: %s", err)
	}
}
