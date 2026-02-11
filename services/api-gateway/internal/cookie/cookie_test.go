package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccessTokenFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		cookieVal  string
		wantToken  string
		wantErrMsg string
	}{
		{
			name:      "Allowed on non-auth route",
			path:      "/v1/users",
			cookieVal: "test-token",
			wantToken: "test-token",
		},
		{
			name:      "Allowed on auth route",
			path:      authRoutePrefix + "/login",
			cookieVal: "test-token",
			wantToken: "test-token",
		},
		{
			name:      "Allowed on update password route",
			path:      "/v1/auth/passwords",
			cookieVal: "test-token",
			wantToken: "test-token",
		},
		{
			name:       "No cookie",
			path:       "/v1/users",
			cookieVal:  "",
			wantErrMsg: "Access token cookie not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.cookieVal != "" {
				r.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: tt.cookieVal})
			}

			got, err := GetAccessTokenFromRequest(r)
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("GetAccessTokenFromRequest() error = nil, want %v", tt.wantErrMsg)
					return
				}
				if err.PublicMessage != tt.wantErrMsg {
					t.Errorf("GetAccessTokenFromRequest() error = %v, want %v", err.PublicMessage, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("GetAccessTokenFromRequest() unexpected error: %v", err)
				return
			}
			if got != tt.wantToken {
				t.Errorf("GetAccessTokenFromRequest() = %v, want %v", got, tt.wantToken)
			}
		})
	}
}

func TestGetRefreshTokenFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		cookieVal  string
		wantToken  string
		wantErrMsg string
	}{
		{
			name:      "Allowed on auth route",
			path:      authRoutePrefix + "/refresh-tokens",
			cookieVal: "refresh-token",
			wantToken: "refresh-token",
		},
		{
			name:       "Blocked on non-auth route",
			path:       "/v1/users",
			cookieVal:  "refresh-token",
			wantErrMsg: "Refresh token cookie not allowed for this route",
		},
		{
			name:       "No cookie",
			path:       authRoutePrefix + "/refresh-tokens",
			cookieVal:  "",
			wantErrMsg: "Refresh token cookie not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.cookieVal != "" {
				r.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: tt.cookieVal})
			}

			got, err := GetRefreshTokenFromRequest(r)
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("GetRefreshTokenFromRequest() error = nil, want %v", tt.wantErrMsg)
					return
				}
				if err.PublicMessage != tt.wantErrMsg {
					t.Errorf("GetRefreshTokenFromRequest() error = %v, want %v", err.PublicMessage, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("GetRefreshTokenFromRequest() unexpected error: %v", err)
				return
			}
			if got != tt.wantToken {
				t.Errorf("GetRefreshTokenFromRequest() = %v, want %v", got, tt.wantToken)
			}
		})
	}
}

func TestCookiePaths(t *testing.T) {
	w := httptest.NewRecorder()
	opts := getCookieOptions(false, "/")
	setAccessTokenCookie(w, "access", opts)

	cookies := w.Result().Cookies()
	foundAccess := false
	for _, c := range cookies {
		if c.Name == accessTokenCookieName {
			foundAccess = true
			if c.Path != "/" {
				t.Errorf("Access token cookie path = %v, want /", c.Path)
			}
		}
	}
	if !foundAccess {
		t.Error("Access token cookie not found")
	}

	w = httptest.NewRecorder()
	opts = getCookieOptions(false, authRoutePrefix)
	setRefreshTokenCookie(w, "refresh", opts)

	cookies = w.Result().Cookies()
	foundRefresh := false
	for _, c := range cookies {
		if c.Name == refreshTokenCookieName {
			foundRefresh = true
			if c.Path != authRoutePrefix {
				t.Errorf("Refresh token cookie path = %v, want %v", c.Path, authRoutePrefix)
			}
		}
	}
	if !foundRefresh {
		t.Error("Refresh token cookie not found")
	}
}

func TestCookieDomains(t *testing.T) {
	tests := []struct {
		name         string
		isProduction bool
		wantDomain   string
	}{
		{
			name:         "Production mode domain",
			isProduction: true,
			wantDomain:   "augno.com",
		},
		{
			name:         "Non-production mode domain",
			isProduction: false,
			wantDomain:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := getCookieOptions(tt.isProduction, "/")
			if opts.Domain != tt.wantDomain {
				t.Errorf("getCookieOptions() domain = %q, want %q", opts.Domain, tt.wantDomain)
			}
		})
	}
}
