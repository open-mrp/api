package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var opaqueTokenUtilsTracer = tracing.GetTracer("auth-service.opaque_token_utils")

// Gen generates a new opaque token.
//
// Ex: b5e7949401f102075ac805c984360b3c256590ae20f3c795d3c1fd21ccd19332
func GenOpaqueToken(ctx context.Context) (string, *apierror.APIError) {
	_, span := opaqueTokenUtilsTracer.Start(ctx, "utils.opaque_token.gen")
	defer span.End()

	// Generate a random 32 byte token
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate opaque token."))
	}

	// Encode the token to a string
	token := hex.EncodeToString(randBytes)
	return token, nil
}
