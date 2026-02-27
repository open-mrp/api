package types

import (
	"fmt"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func CheckIsUser(identity *Identity) *apierror.APIError {
	if err := CheckIsAuthenticated(identity); err != nil {
		return err
	}

	if !identity.IsUser() {
		return apierror.NewAuthorizationError("You must be a user to access this resource.")
	}

	return nil
}

func CheckIsAPIKey(identity *Identity) *apierror.APIError {
	if err := CheckIsAuthenticated(identity); err != nil {
		return err
	}

	if !identity.IsAPIKey() {
		return apierror.NewAuthorizationError("You must be an API key to access this resource.")
	}

	return nil
}

func CheckIsAuthenticated(identity *Identity) *apierror.APIError {
	if !identity.IsAuthenticated() {
		return apierror.NewAuthenticationError("You must be authenticated to access this resource.")
	}

	return nil
}

func CheckIsInternalActor(identity *Identity) *apierror.APIError {
	if err := CheckIsAuthenticated(identity); err != nil {
		return err
	}

	if !identity.IsInternalUser() {
		return apierror.NewAuthorizationError("You must be an internal user for this account to access this resource.")
	}

	return nil
}

func CheckIsAdmin(identity *Identity) *apierror.APIError {
	if err := CheckIsAuthenticated(identity); err != nil {
		return err
	}

	if !identity.IsAdmin() {
		return apierror.NewAuthorizationError("You must be an administrator to access this resource.")
	}

	return nil
}

func CheckNotSandboxMode(identity *Identity) *apierror.APIError {
	if err := CheckIsAuthenticated(identity); err != nil {
		return err
	}

	if identity.AccountMode == constants.AccountModeSandbox {
		return apierror.NewAuthorizationError("Sandbox management is not available in sandbox mode.")
	}

	return nil
}

func CheckHasPermission(identity *Identity, domain PermissionDomain, action Action) *apierror.APIError {
	if identity == nil || identity.Actor == nil {
		return apierror.NewAuthorizationError("You do not have permission to access this resource.")
	}

	// Admins have all permissions
	if identity.IsAdmin() {
		return nil
	}

	permissionKey := fmt.Sprintf("%s:%s", domain, action)
	if hasPerm, ok := identity.Actor.Permissions[permissionKey]; ok && hasPerm {
		return nil
	}

	return apierror.NewAuthorizationError(GetPermissionErrorMessage(identity, domain, action))
}

func GetPermissionErrorMessage(identity *Identity, domain PermissionDomain, action Action) string {
	if identity != nil && identity.Actor != nil {
		switch identity.Type {
		case IdentityTypeUser:
			return fmt.Sprintf("User %s does not have permission to %s:%s", identity.Actor.ID, domain, action)
		case IdentityTypeAPIKey:
			return fmt.Sprintf("This API Key does not have permission to %s:%s", domain, action)
		default:
			return fmt.Sprintf("You do not have permission to %s:%s", domain, action)
		}
	}
	return fmt.Sprintf("You do not have permission to %s:%s", domain, action)
}
