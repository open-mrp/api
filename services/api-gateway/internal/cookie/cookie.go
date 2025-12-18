package cookie

import (
	"context"
	"net/http"
	"time"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
)

const (
	// #nosec G101 - These are cookie names, not hardcoded credentials
	accessTokenCookieName = "__Secure-augno.access-token"
	// #nosec G101 - These are cookie names, not hardcoded credentials
	refreshTokenCookieName = "__Secure-augno.refresh-token"
)

type cookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
	Domain   string
}

func getCookieOptions(isProduction bool) cookieOptions {
	var sameSite http.SameSite
	if isProduction {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteNoneMode
	}

	opts := cookieOptions{
		Secure:   true,
		SameSite: sameSite,
		Path:     "/",
	}

	if isProduction {
		opts.Domain = "api.augno.com"
	}

	return opts
}

func setAccessTokenCookie(w http.ResponseWriter, token string, opts cookieOptions) {
	maxAge := 60 * 60 // 1 hour in seconds
	expires := time.Now().Add(time.Duration(maxAge) * time.Second)

	cookie := &http.Cookie{
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

	http.SetCookie(w, cookie)
}

func setRefreshTokenCookie(w http.ResponseWriter, token string, opts cookieOptions) {
	maxAge := 30 * 24 * 60 * 60 // 30 days in seconds
	expires := time.Now().Add(time.Duration(maxAge) * time.Second)

	cookie := &http.Cookie{
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

	http.SetCookie(w, cookie)
}

func clearAccessTokenCookie(w http.ResponseWriter, opts cookieOptions) {
	cookie := &http.Cookie{
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

	http.SetCookie(w, cookie)
}

func clearRefreshTokenCookie(w http.ResponseWriter, opts cookieOptions) {
	cookie := &http.Cookie{
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

	http.SetCookie(w, cookie)
}

func SetAuthCookiesFromContext(ctx context.Context, accessToken, refreshToken string) {
	w, ok := apicontext.GetResponseWriterFromContext(ctx)
	if !ok {
		panic("response writer not found in context")
	}
	platform, okp := apicontext.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	cookieOpts := getCookieOptions(isProduction)
	setAccessTokenCookie(w, accessToken, cookieOpts)
	setRefreshTokenCookie(w, refreshToken, cookieOpts)
}

func SetAccessTokenCookieFromContext(ctx context.Context, accessToken string) {
	w, ok := apicontext.GetResponseWriterFromContext(ctx)
	if !ok {
		panic("response writer not found in context")
	}
	platform, okp := apicontext.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	cookieOpts := getCookieOptions(isProduction)
	setAccessTokenCookie(w, accessToken, cookieOpts)
}

func ClearRefreshTokenCookieFromContext(ctx context.Context) {
	w, ok := apicontext.GetResponseWriterFromContext(ctx)
	if !ok {
		panic("response writer not found in context")
	}
	platform, okp := apicontext.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	cookieOpts := getCookieOptions(isProduction)
	clearRefreshTokenCookie(w, cookieOpts)
}

func ClearAccessTokenCookieFromContext(ctx context.Context) {
	w, ok := apicontext.GetResponseWriterFromContext(ctx)
	if !ok {
		panic("response writer not found in context")
	}
	platform, okp := apicontext.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	cookieOpts := getCookieOptions(isProduction)
	clearAccessTokenCookie(w, cookieOpts)
}

func GetAccessTokenFromRequest(r *http.Request) (string, *contracts.APIError) {
	cookie, err := r.Cookie(accessTokenCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return "", contracts.NewResourceNotFoundError("Access token cookie not found")
	}
	return cookie.Value, nil
}

func GetRefreshTokenFromRequest(r *http.Request) (string, *contracts.APIError) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return "", contracts.NewResourceNotFoundError("Refresh token cookie not found")
	}
	return cookie.Value, nil
}
