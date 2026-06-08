package cookie

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

const (
	// #nosec G101 - These are cookie names, not hardcoded credentials
	accessTokenCookieName = "__Secure-augno.access-token"
	// #nosec G101 - These are cookie names, not hardcoded credentials
	refreshTokenCookieName = "__Secure-augno.refresh-token"

	// Paths
	authRoutePrefix = "/v1/auth"
)

type cookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
	Domain   string
}

func getCookieOptions(isProduction bool, path string) cookieOptions {
	var sameSite http.SameSite
	if isProduction {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteNoneMode
	}

	opts := cookieOptions{
		Secure:   true,
		SameSite: sameSite,
		Path:     path,
	}

	// Use a wildcard domain for production to support subdomains
	if isProduction {
		opts.Domain = ".augno.com"
	}

	return opts
}

func setAccessTokenCookie(w http.ResponseWriter, token string, opts cookieOptions) {
	http.SetCookie(w, makeAccessTokenCookie(token, opts))
}

func setRefreshTokenCookie(w http.ResponseWriter, token string, opts cookieOptions) {
	http.SetCookie(w, makeRefreshTokenCookie(token, opts))
}

func GetAccessTokenFromRequest(r *http.Request) (string, *apierror.APIError) {
	cookie, err := r.Cookie(accessTokenCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return "", apierror.NewResourceNotFoundError("Access token cookie not found")
	}

	return cookie.Value, nil
}

func GetRefreshTokenFromRequest(r *http.Request) (string, *apierror.APIError) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return "", apierror.NewResourceNotFoundError("Refresh token cookie not found")
	}

	// Refresh tokens are restricted to auth routes.
	if !strings.HasPrefix(r.URL.Path, authRoutePrefix) {
		return "", apierror.NewResourceNotFoundError("Refresh token cookie not allowed for this route")
	}

	return cookie.Value, nil
}

func MakeAuthCookies(ctx context.Context, accessToken, refreshToken string) []*http.Cookie {
	platform, okp := appctx.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction

	return []*http.Cookie{
		makeAccessTokenCookie(accessToken, getCookieOptions(isProduction, "/")),
		makeRefreshTokenCookie(refreshToken, getCookieOptions(isProduction, authRoutePrefix)),
	}
}

func MakeAccessTokenCookie(ctx context.Context, accessToken string) *http.Cookie {
	platform, okp := appctx.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	return makeAccessTokenCookie(accessToken, getCookieOptions(isProduction, "/"))
}

func MakeClearAuthCookies(ctx context.Context) []*http.Cookie {
	platform, okp := appctx.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction

	return []*http.Cookie{
		makeClearAccessTokenCookie(getCookieOptions(isProduction, "/")),
		makeClearRefreshTokenCookie(getCookieOptions(isProduction, authRoutePrefix)),
	}
}

func makeAccessTokenCookie(token string, opts cookieOptions) *http.Cookie {
	maxAge := 60 * 60 // 1 hour in seconds
	expires := time.Now().UTC().Add(time.Duration(maxAge) * time.Second)

	return &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    token,
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   maxAge,
		Expires:  expires,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}
}

func makeRefreshTokenCookie(token string, opts cookieOptions) *http.Cookie {
	maxAge := 30 * 24 * 60 * 60 // 30 days in seconds
	expires := time.Now().UTC().Add(time.Duration(maxAge) * time.Second)

	return &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   maxAge,
		Expires:  expires,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}
}

func makeClearAccessTokenCookie(opts cookieOptions) *http.Cookie {
	return &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    "",
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}
}

func makeClearRefreshTokenCookie(opts cookieOptions) *http.Cookie {
	return &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: opts.SameSite,
	}
}
