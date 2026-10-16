package stub

import (
	"context"
	"errors"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// IntegrationCredentialChecker answers credential checks in test mode without calling a provider: credentials that
// contain "invalid" are rejected, those that contain "unreachable" get no answer, and everything else is accepted.
type IntegrationCredentialChecker struct{}

func (IntegrationCredentialChecker) Check(_ context.Context, provider constants.IntegrationCode, credentials string) (constants.IntegrationConnectionStatus, *apierror.APIError) {
	switch {
	case strings.Contains(credentials, "unreachable"):
		return "", apierror.NewExternalServiceError(errors.New("stub provider unreachable"), "Credential check could not reach "+string(provider)+".")
	case strings.Contains(credentials, "invalid"):
		return constants.IntegrationConnectionStatusInvalidCredentials, nil
	default:
		return constants.IntegrationConnectionStatusConnected, nil
	}
}
