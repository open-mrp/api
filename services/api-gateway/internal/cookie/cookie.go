package cookie

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	// #nosec G101 - These are cookie names, not hardcoded credentials
	accessTokenCookieName = "__Secure-openmrp.access-token"
	// #nosec G101 - These are cookie names, not hardcoded credentials
	refreshTokenCookieName = "__Secure-openmrp.refresh-token"

	// Paths
	authRoutePrefix = "/v1/auth"
)

type cookieOptions struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
	Domain   string
}

func getCookieOptions(isProduction bool, path, externalHost string) cookieOptions {
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

	// First-party hosts share a wildcard domain so sessions span all our subdomains. Requests proxied from a customer's custom portal domain get host-only cookies instead: the browser scopes them to that domain, which both makes auth work there and isolates sessions per tenant domain. When the external host is unknown, production keeps the legacy wildcard behavior.
	if isProduction {
		if domain := firstPartyCookieDomain(externalHost); domain != "" {
			opts.Domain = domain
		}
	}

	return opts
}

// firstPartyDomains are the domains we own. augno.com is gone from the list: it now
// 307s to openmrp.ai, so no browser is ever sitting on it when a cookie is set, and a
// host that only redirects never needs a cookie scoped to it. Anything not listed here
// is treated as a customer's own portal domain and gets a host-only cookie, so adding a
// host we merely redirect from would hand it a wildcard it has no use for.
var firstPartyDomains = []string{"openmrp.ai"}

// firstPartyCookieDomain returns the wildcard cookie domain for a first-party host,
// or "" for a customer's custom portal domain, which gets a host-only cookie. An
// empty host (middleware not in the chain) falls back to the primary domain to
// preserve legacy behavior.
func firstPartyCookieDomain(host string) string {
	if host == "" {
		return "." + firstPartyDomains[0]
	}
	for _, domain := range firstPartyDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return "." + domain
		}
	}
	return ""
}

// externalHostFromContext reads the browser-facing host captured by ExternalHostMiddleware; empty when unset.
func externalHostFromContext(ctx context.Context) string {
	host, _ := appctx.GetExternalHost(ctx)
	return host
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
	externalHost := externalHostFromContext(ctx)

	return []*http.Cookie{
		makeAccessTokenCookie(accessToken, getCookieOptions(isProduction, "/", externalHost)),
		makeRefreshTokenCookie(refreshToken, getCookieOptions(isProduction, authRoutePrefix, externalHost)),
	}
}

func MakeAccessTokenCookie(ctx context.Context, accessToken string) *http.Cookie {
	platform, okp := appctx.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	externalHost := externalHostFromContext(ctx)
	return makeAccessTokenCookie(accessToken, getCookieOptions(isProduction, "/", externalHost))
}

func MakeClearAuthCookies(ctx context.Context) []*http.Cookie {
	platform, okp := appctx.GetPlatformFromContext(ctx)
	if !okp {
		panic("platform not found in context")
	}
	isProduction := platform == constants.PlatformModeProduction
	externalHost := externalHostFromContext(ctx)

	return []*http.Cookie{
		makeClearAccessTokenCookie(getCookieOptions(isProduction, "/", externalHost)),
		makeClearRefreshTokenCookie(getCookieOptions(isProduction, authRoutePrefix, externalHost)),
	}
}

func makeAccessTokenCookie(token string, opts cookieOptions) *http.Cookie {
	maxAge := 60 * 60 // 1 hour in seconds
	expires := time.Now().UTC().Add(time.Duration(maxAge) * time.Second)

	return &http.Cookie{ // #nosec G124 - Secure/HttpOnly/SameSite are set from opts; gosec cannot resolve them
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

	return &http.Cookie{ // #nosec G124 - Secure/HttpOnly/SameSite are set from opts; gosec cannot resolve them
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
	return &http.Cookie{ // #nosec G124 - Secure/HttpOnly/SameSite are set from opts; gosec cannot resolve them
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
	return &http.Cookie{ // #nosec G124 - Secure/HttpOnly/SameSite are set from opts; gosec cannot resolve them
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
