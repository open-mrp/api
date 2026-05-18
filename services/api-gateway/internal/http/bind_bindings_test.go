package httptransport

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func requireAPIErr(t *testing.T, err error, wantCode apierror.ErrorCode, wantParam string) {
	t.Helper()
	var ae *apierror.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apierror.APIError, got %T: %v", err, err)
	}
	if ae.Code != wantCode {
		t.Fatalf("want error code %q, got %q", wantCode, ae.Code)
	}
	if got := ae.Param; got != wantParam {
		t.Fatalf("want Param %q, got %q", wantParam, got)
	}
}

func Test_planFor_invalidDestination(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()
		var dst *testRequest // nil ptr
		_, err := planFor(dst)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("non-pointer receiver", func(t *testing.T) {
		t.Parallel()
		v := struct{ X string }{}
		_, err := planFor(v)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("pointer to scalar", func(t *testing.T) {
		t.Parallel()
		x := 3
		_, err := planFor(&x)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("RejectUnknown propagate planFor failure", func(t *testing.T) {
		t.Parallel()
		u := mustParseURL(t, "/?a=1")
		apiErr := RejectUnknownQueryParams(u, 42, false)
		if apiErr == nil {
			t.Fatal("expected error")
		}
		if apiErr.Code != apierror.ErrorCodeInternalError {
			t.Fatalf("got code %v", apiErr.Code)
		}
	})

	t.Run("BindFrom propagate planFor failure", func(t *testing.T) {
		t.Parallel()
		u := mustParseURL(t, "/?id=1")
		if err := BindFromQuery(u, "bad"); err == nil {
			t.Fatal("BindFromQuery: expected error")
		}

		hdr := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := BindFromHeaders(hdr, map[string]string{}); err == nil {
			t.Fatal("BindFromHeaders: expected error")
		}

		if err := BindFromPath(hdr, []int{}); err == nil {
			t.Fatal("BindFromPath: expected error")
		}
	})
}

func TestBindPlan_allowedQuery_keys(t *testing.T) {
	t.Parallel()

	type mixed struct {
		S string   `query:"s"`
		T []string `query:"t"`
	}
	p, err := planFor(&mixed{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.allowedQuery["s"]; !ok {
		t.Fatalf("missing s: %#v", p.allowedQuery)
	}
	if _, ok := p.allowedQuery["t"]; !ok || !mapHasKey(p.allowedQuery, "t[]") {
		t.Fatalf("want t + t[], got %#v", p.allowedQuery)
	}
}

func mapHasKey(m map[string]struct{}, k string) bool {
	_, ok := m[k]
	return ok
}

func TestBindFromQuery_happyPaths_extra(t *testing.T) {
	t.Parallel()

	t.Run("default applies when scalar missing", func(t *testing.T) {
		t.Parallel()
		type body struct {
			Limit int32 `query:"limit" default:"100"`
			Unset int32 `query:"unset"`
		}
		dst := &body{}
		if err := BindFromQuery(mustParseURL(t, "/x"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.Limit != 100 || dst.Unset != 0 {
			t.Fatalf("got limit=%d unset=%d", dst.Limit, dst.Unset)
		}
	})

	t.Run("slice default comma list when absent", func(t *testing.T) {
		t.Parallel()
		type body struct {
			Tags []string `query:"tags" default:"a,b,c"`
		}
		dst := &body{}
		if err := BindFromQuery(mustParseURL(t, "/x"), dst); err != nil {
			t.Fatal(err)
		}
		if len(dst.Tags) != 3 || dst.Tags[0] != "a" || dst.Tags[1] != "b" || dst.Tags[2] != "c" {
			t.Fatalf("got %#v", dst.Tags)
		}
	})

	t.Run("slice single query value splits on comma", func(t *testing.T) {
		t.Parallel()
		type body struct {
			Tags []string `query:"tags"`
		}
		dst := &body{}
		if err := BindFromQuery(mustParseURL(t, "/x?tags=a,b"), dst); err != nil {
			t.Fatal(err)
		}
		if len(dst.Tags) != 2 || dst.Tags[0] != "a" || dst.Tags[1] != "b" {
			t.Fatalf("got %#v", dst.Tags)
		}
	})

	t.Run("slice bracket syntax only", func(t *testing.T) {
		t.Parallel()
		type body struct {
			Tags []string `query:"tags"`
		}
		dst := &body{}
		if err := BindFromQuery(mustParseURL(t, "/x?tags[]=p&tags[]=q"), dst); err != nil {
			t.Fatal(err)
		}
		if len(dst.Tags) != 2 || dst.Tags[0] != "p" || dst.Tags[1] != "q" {
			t.Fatalf("got %#v", dst.Tags)
		}
	})

	t.Run("combine bare and bracket ids", func(t *testing.T) {
		t.Parallel()
		type body struct {
			IDs []int `query:"id"`
		}
		dst := &body{}
		if err := BindFromQuery(mustParseURL(t, "/x?id=1&id[]=3"), dst); err != nil {
			t.Fatal(err)
		}
		if len(dst.IDs) != 2 || dst.IDs[0] != 1 || dst.IDs[1] != 3 {
			t.Fatalf("got %#v", dst.IDs)
		}
	})

	t.Run("bool float duration scalars", func(t *testing.T) {
		t.Parallel()
		type body struct {
			B   bool          `query:"active"`
			Pi  float64       `query:"pi"`
			Dur time.Duration `query:"dur"`
		}
		u := mustParseURL(t, "/x?active=true&pi=2.25&dur=750ms")
		dst := &body{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatal(err)
		}
		if !dst.B || dst.Pi != 2.25 || dst.Dur != 750*time.Millisecond {
			t.Fatalf("got b=%v pi=%v dur=%v", dst.B, dst.Pi, dst.Dur)
		}
	})

	t.Run("RFC3339 time requires pointer receiver with query tag", func(t *testing.T) {
		t.Parallel()
		type body struct {
			Until *time.Time `query:"until"`
			Blank time.Time  `query:"blank"` // time.Time cannot bind as scalar (walker recurses inner struct).
		}
		until := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		q := make(url.Values)
		q.Set("until", until.Format(time.RFC3339))
		q.Set("blank", until.Format(time.RFC3339))
		u := &url.URL{Path: "/x", RawQuery: q.Encode()}
		dst := &body{}
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Until == nil || !dst.Until.Equal(until) {
			t.Fatalf("Until: %+v", dst.Until)
		}
		if !dst.Blank.IsZero() {
			t.Fatalf("non-pointer time.Time ignores query key; got %+v", dst.Blank)
		}
	})

	t.Run("nested named struct binds inner query keys", func(t *testing.T) {
		t.Parallel()
		type inner struct {
			Q string `query:"q"`
		}
		type outer struct {
			Inner inner
		}
		dst := &outer{}
		if err := BindFromQuery(mustParseURL(t, "/nested?q=z"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.Inner.Q != "z" {
			t.Fatalf("Inner.Q=%q", dst.Inner.Q)
		}
	})

	t.Run("nil pointer nested struct skips descendant query binds", func(t *testing.T) {
		t.Parallel()
		type inner struct {
			Q string `query:"child_q"`
		}
		type outer struct {
			I *inner
		}
		dst := &outer{I: nil}
		if err := BindFromQuery(mustParseURL(t, "/z?child_q=should-ignore"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.I != nil {
			t.Fatalf("unexpected allocation I=%+v", dst.I)
		}
	})

	t.Run("anonymous pointer embed nil skips promoted binds", func(t *testing.T) {
		t.Parallel()
		type wrapper struct {
			*apiresource.PaginationRequest
			Explicit string `query:"ex"`
		}
		dst := &wrapper{}
		u := mustParseURL(t, "/w?cursor=x&limit=9&ex=hit")
		if err := BindFromQuery(u, dst); err != nil {
			t.Fatal(err)
		}
		if dst.PaginationRequest != nil {
			t.Fatal("embedded pointer must stay nil")
		}
		if dst.Explicit != "hit" {
			t.Fatalf("explicit %q", dst.Explicit)
		}
	})

	t.Run("anonymous pointer embed non-nil binds promoted pagination fields", func(t *testing.T) {
		t.Parallel()
		type wrapper struct {
			*apiresource.PaginationRequest
		}
		dst := &wrapper{PaginationRequest: &apiresource.PaginationRequest{}}
		if err := BindFromQuery(mustParseURL(t, "/w?cursor=c2&q=needle"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.Cursor == nil || *dst.Cursor != "c2" {
			t.Fatalf("cursor %+v", dst.Cursor)
		}
		if dst.Query == nil || *dst.Query != "needle" {
			t.Fatalf("q %+v", dst.Query)
		}
	})

	t.Run("typed string slice enums", func(t *testing.T) {
		t.Parallel()
		type req struct {
			Statuses []constants.APIKeyStatus `query:"statuses"`
		}
		dst := &req{}
		if err := BindFromQuery(mustParseURL(t, "/k?statuses=active&statuses[]=revoked"), dst); err != nil {
			t.Fatal(err)
		}
		if len(dst.Statuses) != 2 ||
			dst.Statuses[0] != constants.APIKeyStatusActive ||
			dst.Statuses[1] != constants.APIKeyStatusRevoked {
			t.Fatalf("got %#v", dst.Statuses)
		}
	})
}

func TestBindFromQuery_uint_scalar(t *testing.T) {
	t.Parallel()

	type req struct {
		N uint64 `query:"n"`
	}
	dst := &req{}
	if err := BindFromQuery(mustParseURL(t, "/x?n=9007199254740993"), dst); err != nil {
		t.Fatal(err)
	}
	if dst.N != 9007199254740993 {
		t.Fatalf("got %d", dst.N)
	}
	err := BindFromQuery(mustParseURL(t, "/x?n=-1"), dst)
	requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "n")
}

func TestBindFromQuery_invalidTimePointerParse(t *testing.T) {
	t.Parallel()

	type req struct {
		Since *time.Time `query:"since"`
	}
	dst := &req{}
	err := BindFromQuery(mustParseURL(t, "/x?since=not-a-time"), dst)
	requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "since")
}

func TestBindFromHeaders_cookieUsedWhenAuthorizationHeaderAbsent(t *testing.T) {
	t.Parallel()

	type r struct {
		Refresh string `header:"Authorization" cookie:"rt"`
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "rt", Value: "only-cookie"})
	dst := &r{}
	if err := BindFromHeaders(req, dst); err != nil {
		t.Fatal(err)
	}
	if dst.Refresh != "only-cookie" {
		t.Fatalf("got %q", dst.Refresh)
	}
}

func TestBindFromPath_then_BindFromQuery_sameDestination(t *testing.T) {
	t.Parallel()

	type duo struct {
		PID int    `path:"pid"`
		Q   string `query:"q"`
	}
	dst := &duo{}
	pathReq := newRequestWithPathParams(http.MethodGet, "/p/9", nil, map[string]string{"pid": "9"})
	if err := BindFromPath(pathReq, dst); err != nil {
		t.Fatal(err)
	}
	if err := BindFromQuery(mustParseURL(t, "/p/9?q=needle"), dst); err != nil {
		t.Fatal(err)
	}
	if dst.PID != 9 || dst.Q != "needle" {
		t.Fatalf("%+v", dst)
	}
}

func TestNavigateBindField_rejectsNonStructRoot(t *testing.T) {
	t.Parallel()
	bf := bindField{
		steps: []walkStep{{kind: byte(stepField), idx: 0}},
	}
	if _, ok := navigateBindField(reflect.ValueOf("not-a-struct"), bf); ok {
		t.Fatal("expected navigate to fail when root is not a struct")
	}
}

func TestBindFromQuery_invalidFloatScalar(t *testing.T) {
	t.Parallel()

	type req struct {
		Pi float64 `query:"pi"`
	}
	err := BindFromQuery(mustParseURL(t, "/x?pi=notnum"), &req{})
	requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "pi")
}

func TestBindFromQuery_sliceRepeatedEmptyFirstElementParsesInts(t *testing.T) {
	t.Parallel()

	type req struct {
		IDs []int `query:"ids"`
	}
	// '?ids=&ids=7' repeats the parameter; URL parsing yields ids=["" ""]
	dst := &req{}
	err := BindFromQuery(mustParseURL(t, "/x?ids=&ids=7"), dst)
	requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "ids")
}

func TestBindFromQuery_repeatedScalarQueryKey_firstValueWins(t *testing.T) {
	t.Parallel()

	type req struct {
		B bool `query:"active"`
	}
	t.Run("first value binds when non-empty", func(t *testing.T) {
		t.Parallel()
		dst := &req{}
		if err := BindFromQuery(mustParseURL(t, "/x?active=true&active=false"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.B != true {
			t.Fatalf("got %v", dst.B)
		}
	})
	t.Run("leading empty skips bind entirely", func(t *testing.T) {
		t.Parallel()
		dst := &req{}
		if err := BindFromQuery(mustParseURL(t, "/x?active=&active=true"), dst); err != nil {
			t.Fatal(err)
		}
		if dst.B {
			t.Fatal("scalar query uses Values.Get(first); trailing true is ignored") // deliberate net/url semantics
		}
	})
}

func TestBindFromQuery_errorPaths_extended(t *testing.T) {
	t.Parallel()

	t.Run("invalid int", func(t *testing.T) {
		t.Parallel()
		dst := &testRequest{}
		err := BindFromQuery(mustParseURL(t, "/bad?id=oops"), dst)
		requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "id")
	})

	t.Run("invalid slice element", func(t *testing.T) {
		t.Parallel()
		type body struct {
			IDs []int `query:"ids"`
		}
		dst := &body{}
		err := BindFromQuery(mustParseURL(t, "/bad?ids=1&ids=nope"), dst)
		requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "ids")
	})

	t.Run("invalid bool", func(t *testing.T) {
		t.Parallel()
		type body struct {
			B bool `query:"active"`
		}
		dst := &body{}
		err := BindFromQuery(mustParseURL(t, "/bad?active=perhaps"), dst)
		requireAPIErr(t, err, apierror.ErrorCodeParameterInvalid, "active")
	})
}

func TestRejectUnknownQueryParams_extended(t *testing.T) {
	t.Parallel()

	type listReq struct {
		A string `query:"a"`
	}

	t.Run("include rejected when allowInclude false", func(t *testing.T) {
		t.Parallel()
		u := mustParseURL(t, "/x?a=1&include=role")
		dst := &listReq{}
		apiErr := RejectUnknownQueryParams(u, dst, false)
		if apiErr == nil || apiErr.Code != apierror.ErrorCodeParameterUnknown || apiErr.Param != "include" {
			t.Fatalf("got %v", apiErr)
		}
	})

	t.Run("allowInclude true ok without include keys", func(t *testing.T) {
		t.Parallel()
		u := mustParseURL(t, "/x?a=1")
		dst := &listReq{}
		if apiErr := RejectUnknownQueryParams(u, dst, true); apiErr != nil {
			t.Fatal(apiErr)
		}
	})

	t.Run("empty query string ok", func(t *testing.T) {
		t.Parallel()
		u := mustParseURL(t, "/x")
		dst := &listReq{}
		if apiErr := RejectUnknownQueryParams(u, dst, false); apiErr != nil {
			t.Fatalf("unexpected %v", apiErr)
		}
	})
}

func TestBindFromPath_extended(t *testing.T) {
	t.Parallel()

	t.Run("defaults when path param absent", func(t *testing.T) {
		t.Parallel()
		type r struct {
			ID     string `path:"id"`
			Action string `path:"action" default:"edit"`
		}
		dst := &r{}
		req := newRequestWithPathParams(http.MethodGet, "/items/xyz", nil, map[string]string{"id": "xyz"})
		if err := BindFromPath(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.ID != "xyz" || dst.Action != "edit" {
			t.Fatalf("%+v", dst)
		}
	})

	t.Run("explicit beats default", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Action string `path:"action" default:"edit"`
		}
		dst := &r{}
		req := newRequestWithPathParams(http.MethodGet, "/", nil, map[string]string{"action": "run"})
		if err := BindFromPath(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Action != "run" {
			t.Fatalf("%q", dst.Action)
		}
	})

	t.Run("missing param without default is ignored", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Optional string `path:"missing"`
			Seen     string `path:"seen"`
		}
		dst := &r{}
		req := newRequestWithPathParams(http.MethodGet, "/", nil, map[string]string{"seen": "y"})
		if err := BindFromPath(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Optional != "" || dst.Seen != "y" {
			t.Fatalf("%+v", dst)
		}
	})

	t.Run("invalid integer path segment", func(t *testing.T) {
		t.Parallel()
		type r struct {
			N int `path:"num"`
		}
		dst := &r{}
		req := newRequestWithPathParams(http.MethodGet, "/", nil, map[string]string{"num": "zzz"})
		requireAPIErr(t, BindFromPath(req, dst), apierror.ErrorCodeParameterInvalid, "num")
	})
}

func TestBindFromHeaders_extended(t *testing.T) {
	t.Parallel()

	t.Run("Authorization Bearer strips scheme", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Auth string `header:"Authorization" scheme:"Bearer"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer abc_token")
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Auth != "abc_token" {
			t.Fatalf("%q", dst.Auth)
		}
	})

	t.Run("scheme prefix on custom header strips prefix", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Token string `header:"X-Token" scheme:"Custom"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Token", "Custom spaced-value")
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Token != "spaced-value" {
			t.Fatalf("%q", dst.Token)
		}
	})

	t.Run("cookie-only field", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Sess string `cookie:"session_id"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess_val"})
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Sess != "sess_val" {
			t.Fatalf("%q", dst.Sess)
		}
	})

	t.Run("header default when header absent", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Locale string `header:"Accept-Language" default:"en-US"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Locale != "en-US" {
			t.Fatalf("%q", dst.Locale)
		}
	})

	t.Run("cookie integer parses via setFromString", func(t *testing.T) {
		t.Parallel()
		type r struct {
			N int `cookie:"counter"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "counter", Value: "42"})
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.N != 42 {
			t.Fatalf("%d", dst.N)
		}
	})

	t.Run("Authorization invalid falls back to cookie", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Refresh string `header:"Authorization" cookie:"fallback"`
		}
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		req.Header.Set("Authorization", "not-valid-scheme xxx")
		req.AddCookie(&http.Cookie{Name: "fallback", Value: "cookie_token"})
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Refresh != "cookie_token" {
			t.Fatalf("%q", dst.Refresh)
		}
	})

	t.Run("Bearer tag mismatches Basic and uses cookie fallback", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Token string `header:"Authorization" scheme:"Bearer" cookie:"fb"`
		}
		raw := base64.StdEncoding.EncodeToString([]byte("user:pw"))
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		req.Header.Set("Authorization", "Basic "+raw)
		req.AddCookie(&http.Cookie{Name: "fb", Value: "from_cookie"})
		dst := &r{}
		if err := BindFromHeaders(req, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Token != "from_cookie" {
			t.Fatalf("%q", dst.Token)
		}
	})
}

func TestBindFromHeaders_errorPaths_extended(t *testing.T) {
	t.Parallel()

	t.Run("scheme prefix mismatch on arbitrary header", func(t *testing.T) {
		t.Parallel()
		type r struct {
			X string `header:"X-Tok" scheme:"Want"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Tok", "Wrong noprefix")
		requireAPIErr(t, BindFromHeaders(req, &r{}), apierror.ErrorCodeParameterInvalid, "X-Tok")
	})

	t.Run("integer header parses failure", func(t *testing.T) {
		t.Parallel()
		type r struct {
			N int `header:"X-Num"`
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Num", "nan")
		requireAPIErr(t, BindFromHeaders(req, &r{}), apierror.ErrorCodeParameterInvalid, "X-Num")
	})

	t.Run("bad Authorization header and no cookie yields invalid credentials", func(t *testing.T) {
		t.Parallel()
		type r struct {
			Token string `header:"Authorization" cookie:"absent_cookie"`
		}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Garbage")
		requireAPIErr(t, BindFromHeaders(req, &r{}), apierror.ErrorCodeInvalidCredentials, "Authorization")
	})
}

func TestBindIncomingRequest_matchesSeparateBindings(t *testing.T) {
	t.Parallel()

	type trio struct {
		Sess string `header:"X-Sess"`
		PID  int    `path:"pid"`
		Q    string `query:"q"`
	}

	separate := &trio{}
	incoming := &trio{}

	req := newRequestWithPathParams(http.MethodGet, "/items", nil, map[string]string{"pid": "42"})
	req.Header.Set("X-Sess", "sess-val")
	req.URL.RawQuery = "q=needle"

	if err := BindFromHeaders(req, separate); err != nil {
		t.Fatalf("BindFromHeaders: %v", err)
	}
	if err := BindFromPath(req, separate); err != nil {
		t.Fatalf("BindFromPath: %v", err)
	}
	if err := BindFromQuery(req.URL, separate); err != nil {
		t.Fatalf("BindFromQuery: %v", err)
	}
	if apiErr := RejectUnknownQueryParams(req.URL, separate, false); apiErr != nil {
		t.Fatalf("RejectUnknownQueryParams: %v", apiErr)
	}

	if err := BindIncomingRequest(req, incoming, false); err != nil {
		t.Fatalf("BindIncomingRequest: %v", err)
	}

	if *separate != *incoming {
		t.Fatalf("mismatch separate=%+v incoming=%+v", separate, incoming)
	}
}

func TestBindIncomingRequest_rejectsUndeclaredQueryKeys(t *testing.T) {
	t.Parallel()

	type list struct {
		Cursor *string `query:"cursor"`
	}
	req := httptest.NewRequest(http.MethodGet, "/items?cursor=c1&junk=bad", nil)
	err := BindIncomingRequest(req, &list{}, false)
	requireAPIErr(t, err, apierror.ErrorCodeParameterUnknown, "junk")
}
