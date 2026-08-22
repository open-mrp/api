package token

import (
	"context"

	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var opaqueTokenTracer = tracing.GetTracer("auth-service.opaque_token")

// GenOpaqueToken generates a new opaque token.
func GenOpaqueToken(ctx context.Context) (string, *apierror.APIError) {
	_, span := opaqueTokenTracer.Start(ctx, "token.opaque_token.gen")
	defer span.End()

	t, err := crypto.RandAlphanumericString(32)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate opaque token."))
	}

	return t, nil
}
