package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/versiontransforms"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	"github.com/open-mrp/api/services/api-gateway/internal/header"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/omrpctx"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/middleware"
	"github.com/open-mrp/apikit/ratelimit"
	kitrouter "github.com/open-mrp/apikit/router"
	"github.com/open-mrp/apikit/transport"
)

// router is one listener's routes and middleware, served behind apikit's request log.
type router struct {
	*kitrouter.Router
	handler http.Handler
}

func newRouter(cfg *BaseConfig) *router {
	r := &router{Router: kitrouter.NewRouter()}
	r.handler = middleware.RequestLog(middleware.RequestLogConfig{
		Saver:            requestLogSaver{publisher: cfg.RequestLogPublisher},
		Routes:           r.Router,
		Logger:           slog.New(slog.NewTextHandler(cfg.LogWriter, nil)).With("listener", cfg.LogPrefix),
		NewID:            newRequestID,
		TrustedProxyHops: cfg.TrustedProxyHops,
		SkipPaths:        []string{"/healthz"},
		MaxBodyBytes:     omrpctx.MaxLoggedBodyBytes,
	})(versiontransforms.RoutePreview9Jobs(r.Router))
	return r
}

// ServeHTTP serves a request through the request log and the router.
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func newRequestID() string {
	length := id.IDLength19
	requestID, err := id.GenID(id.RequestIDPrefix, &length)
	if err != nil {
		panic(err)
	}
	return requestID
}

// requestLogSaver publishes each finished request's log to the outbox.
type requestLogSaver struct {
	publisher domain.RequestLogPublisher
}

func (s requestLogSaver) Save(ctx context.Context, rl *appctx.RequestLog) error {
	return s.publisher.Create(ctx, rl)
}

// blockedIPs are client IPs refused outright.
var blockedIPs = []string{"49.43.184.36"}

// corsConfig names OpenMRP's own request and response headers for browsers.
var corsConfig = middleware.CORSConfig{
	AllowHeaders:  []string{header.TargetAccountIDHeader, header.ActorAccountIDHeader},
	ExposeHeaders: []string{transport.RetryAfterHeader, transport.IdempotentReplayedHeader},
}

// rateLimit limits each client IP to limit requests a second, except in development and test, where all traffic shares one IP.
func rateLimit(limit int, hops int) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.RateLimit(middleware.RateLimitConfig{
		Limiter:          ratelimit.NewMemory(limit, time.Second),
		TrustedProxyHops: hops,
		SkipPaths:        []string{"/healthz"},
		Skip:             isTestOrDevPlatform,
	})
}

func isTestOrDevPlatform(r *http.Request) bool {
	platform, ok := omrpctx.GetPlatformFromContext(r.Context())
	return ok && (platform.IsTest() || platform.IsDevelopment())
}
