package httptransport

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/version"
)

type RespondOption func(http.Header)

func WithHeader(key, value string) RespondOption {
	return func(h http.Header) { h.Set(key, value) }
}

func applyRespondOptions(w http.ResponseWriter, ctx context.Context, code int, opts []RespondOption) {
	w.Header().Set(header.ContentTypeHeader, "application/json")
	if v, ok := appctx.GetAPIVersionFromContext(ctx); ok {
		w.Header().Set(header.VersionHeader, v.String())
	} else {
		w.Header().Set(header.VersionHeader, version.Latest.String())
	}
	rl, ok := appctx.GetRequestLog(ctx)
	if ok && rl != nil {
		w.Header().Set(header.RequestIDHeader, rl.ID)
	}
	if code == http.StatusUnauthorized {
		w.Header().Set(header.WwwAuthenticateHeader, "Bearer")
	}
	for _, opt := range opts {
		opt(w.Header())
	}
}

func RespondWithJSON(ctx context.Context, w http.ResponseWriter, code int, payload any, opts ...RespondOption) {
	applyRespondOptions(w, ctx, code, opts)
	var dat []byte
	var err error
	rl, ok := appctx.GetRequestLog(ctx)
	if ok && rl != nil && rl.UserAgent != nil && strings.HasPrefix(*rl.UserAgent, "curl/") {
		dat, err = json.MarshalIndent(payload, "", "  ")
	} else {
		dat, err = json.Marshal(payload)
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

func RespondWithJSONBytes(ctx context.Context, w http.ResponseWriter, code int, body []byte, opts ...RespondOption) {
	applyRespondOptions(w, ctx, code, opts)
	w.WriteHeader(code)
	if _, err := w.Write(body); err != nil {
		log.Printf("Error writing response: %s", err)
	}
}
