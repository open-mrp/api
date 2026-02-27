package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/timeutil"
)

// RefreshToken represents a token used to obtain new access tokens.
type RefreshToken struct {
	// The opaque refresh token value.
	Token string `json:"token" validate:"required"`
	// When this refresh token expires.
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
}

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleRefreshTokenToken = "d7842e40d46df9033f68a761ddd866bb1eafefbff887806fd7918c82f74bc13a"

var SampleRefreshToken = &RefreshToken{
	Token:     SampleRefreshTokenToken,
	ExpiresAt: timeutil.TimestampToTime(sampleExpiresAtTimestamp),
}

func (*RefreshToken) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRefreshToken)
}
