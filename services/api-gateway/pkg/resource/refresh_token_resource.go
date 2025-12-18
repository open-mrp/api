package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/ptrutil"
)

// The refresh token
type RefreshToken struct {
	// The refresh token
	Token string `json:"token" validate:"required"`
	// The refresh token expires at
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
}

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleRefreshTokenToken = "d7842e40d46df9033f68a761ddd866bb1eafefbff887806fd7918c82f74bc13a"

var SampleRefreshToken = &RefreshToken{
	Token:     SampleRefreshTokenToken,
	ExpiresAt: ptrutil.TimestampToTime(sampleExpiresAtTimestamp),
}

func (*RefreshToken) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRefreshToken)
}
