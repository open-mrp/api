package domain

import "github.com/open-mrp/apikit/appctx"

// RequestLog is a type alias so that *domain.RequestLog and *appctx.RequestLog are interchangeable. The canonical definition lives in shared/appctx.
type RequestLog = appctx.RequestLog
