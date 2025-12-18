package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"
)

var opaqueTokenUtilsTracer = tracing.GetTracer("auth-service.opaque_token_utils")

type OpaqueTokenConfig struct {
}

func DefaultOpaqueTokenConfig() OpaqueTokenConfig {
	return OpaqueTokenConfig{}
}

type opaqueTokenUtilsImpl struct {
	config OpaqueTokenConfig
}

func NewOpaqueTokenUtils(config OpaqueTokenConfig) domain.OpaqueTokenUtils {
	return &opaqueTokenUtilsImpl{config: config}
}

// Gen generates a new opaque token.
//
// Ex: b5e7949401f102075ac805c984360b3c256590ae20f3c795d3c1fd21ccd19332
func (rtu *opaqueTokenUtilsImpl) Gen(ctx context.Context) (string, *contracts.APIError) {
	_, span := opaqueTokenUtilsTracer.Start(ctx, "utils.opaque_token.gen")
	defer span.End()

	// Generate a random 32 byte token
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate opaque token."))
	}

	// Encode the token to a string
	token := hex.EncodeToString(randBytes)
	return token, nil
}
