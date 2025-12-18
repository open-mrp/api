package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/services/api-gateway/internal/header"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/fuzzy"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var PathExtractor func(*http.Request) func(string) string = defaultPathExtractor

const (
	attrHTTPStatusCode = "http.status_code"
	attrErrorType      = "error.type"
)

func ConvertToHTTPHandler[TReq, TResp any](be apiendpoint.BoundEndpoint[TReq, TResp]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		span := trace.SpanFromContext(ctx)

		ctx = apicontext.WithResponseWriter(ctx, w)

		var req TReq
		req = allocIfPtr(req)

		if err := bindFromHeaders(r, any(req)); err != nil {
			apiErr, ok := err.(*contracts.APIError)
			if !ok {
				apiErr = contracts.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(attrErrorType, "header_binding"))
			}
			RespondWithAPIError(ctx, w, apiErr)
			return
		}
		if err := bindFromPath(r, any(req)); err != nil {
			apiErr, ok := err.(*contracts.APIError)
			if !ok {
				apiErr = contracts.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(attrErrorType, "path_binding"))
			}
			RespondWithAPIError(ctx, w, apiErr)
			return
		}
		if err := bindFromQuery(r.URL, any(req)); err != nil {
			apiErr, ok := err.(*contracts.APIError)
			if !ok {
				apiErr = contracts.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(attrErrorType, "query_binding"))
			}
			RespondWithAPIError(ctx, w, apiErr)
			return
		}

		if shouldDecodeBody(r) {
			if err := decodeJSONInto(any(req), r, !be.Spec.Extras.AllowUnknownJSONFields); err != nil {
				apiErr := contracts.NewValidationError(err.Error())
				tracing.RecordControllerError(span, apiErr)
				if span.IsRecording() {
					span.SetAttributes(attribute.String(attrErrorType, "json_decode"))
				}
				RespondWithAPIError(ctx, w, apiErr)
				return
			}
		}

		if apiErr := validate.Validate(any(req)); apiErr != nil {
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(attrErrorType, "validation"))
			}
			RespondWithAPIError(ctx, w, apiErr)
			return
		}

		resp, err := be.Handler(ctx, req)
		if err != nil {
			tracing.RecordControllerError(span, err)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(attrErrorType, "handler"))
			}
			RespondWithAPIError(ctx, w, err)
			return
		}

		if span.IsRecording() {
			span.SetAttributes(attribute.Int(attrHTTPStatusCode, be.Spec.SuccessStatusCode))
		}
		RespondWithJSON(ctx, w, be.Spec.SuccessStatusCode, resp)
	}
}

func shouldDecodeBody(r *http.Request) bool {
	if r.ContentLength == 0 && !hasChunked(r) {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func hasChunked(r *http.Request) bool {
	for _, te := range r.TransferEncoding {
		if strings.EqualFold(te, "chunked") {
			return true
		}
	}
	return false
}

func decodeJSONInto(dst any, r *http.Request, disallowUnknown bool) error {
	dec := json.NewDecoder(r.Body)
	if disallowUnknown {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		if field, ok := extractUnknownJSONFieldName(err); ok {
			candidates := collectJSONFieldNames(dst)
			if suggestion, dist := fuzzy.FindClosestByLevenshtein(field, candidates); suggestion != "" && dist <= 3 {
				return fmt.Errorf("Invalid JSON in request body: unknown field '%s'. Did you mean '%s'?", field, suggestion)
			}
			return fmt.Errorf("Invalid JSON in request body: unknown field '%s'.", field)
		}
		return err
	}
	if dec.More() {
		var extra any
		if err := dec.Decode(&extra); err == nil {
			return errors.New("unexpected extra JSON after request body")
		}
	}
	return nil
}

// extractUnknownJSONFieldName tries to parse the offending field name from a json decoder error
// when DisallowUnknownFields is enabled. It returns the field name and true if found.
func extractUnknownJSONFieldName(err error) (string, bool) {
	// Typical error format: `json: unknown field "expires"`
	msg := err.Error()
	const marker = "unknown field \""
	idx := strings.Index(msg, marker)
	if idx == -1 {
		return "", false
	}
	start := idx + len(marker)
	rest := msg[start:]
	end := strings.Index(rest, "\"")
	if end == -1 {
		return "", false
	}
	field := rest[:end]
	if field == "" {
		return "", false
	}
	return field, true
}

// collectJSONFieldNames returns the set of acceptable JSON keys for the provided destination struct.
// It considers json tags; if absent, it falls back to the exported field name.
func collectJSONFieldNames(dst any) []string {
	t := reflect.TypeOf(dst)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	namesSet := map[string]struct{}{}
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		if rt.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < rt.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath != "" { // unexported
				continue
			}
			// Handle anonymous embedded structs (inline in JSON)
			if sf.Anonymous {
				ft := sf.Type
				for ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					walk(ft)
					continue
				}
			}

			tag := sf.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name := ""
			if tag != "" {
				comma := strings.IndexByte(tag, ',')
				if comma >= 0 {
					name = tag[:comma]
				} else {
					name = tag
				}
			}
			if name == "" { // fallback to field name if no tag name component
				name = sf.Name
				if name == "" {
					continue
				}
				// encoding/json matches field names case-insensitively, but suggestions
				// should prefer the typical JSON style; we keep the declared name.
			}
			namesSet[name] = struct{}{}
		}
	}
	walk(t)

	names := make([]string, 0, len(namesSet))
	for k := range namesSet {
		names = append(names, k)
	}
	return names
}

func bindFromHeaders(r *http.Request, dst any) error {
	return walkStruct(dst, func(f fieldInfo) error {
		h := f.tag.Get("header")
		cookieName := f.tag.Get("cookie")

		// Try header first
		var val string
		var fromHeader bool
		if h != "" {
			val = r.Header.Get(h)
			fromHeader = val != ""
		}

		// Handle Authorization header with flexible schemes (Bearer or Basic)
		// If Authorization header is present, try to validate it
		if h == "Authorization" && fromHeader {
			scheme := f.tag.Get("scheme")
			// If scheme is not specified, allow both Bearer and Basic
			authResult, err := header.ValidateAuthHeader(val)
			if err != nil {
				// If validation fails and cookie fallback is available, try cookie
				if cookieName != "" {
					if cookie, cookieErr := r.Cookie(cookieName); cookieErr == nil && cookie != nil && cookie.Value != "" {
						return setFromString(f.value, cookie.Value, f.tag)
					}
				}
				// No cookie fallback or cookie also failed, return the auth header error
				return err
			}

			// If scheme is specified, verify it matches
			if scheme != "" {
				expectedScheme := strings.ToLower(scheme)
				actualScheme := strings.ToLower(string(authResult.Scheme))
				if expectedScheme != actualScheme {
					// Try cookie fallback if scheme doesn't match
					if cookieName != "" {
						if cookie, cookieErr := r.Cookie(cookieName); cookieErr == nil && cookie != nil && cookie.Value != "" {
							return setFromString(f.value, cookie.Value, f.tag)
						}
					}
					return fmt.Errorf("invalid %s header scheme: expected %s, got %s", h, expectedScheme, actualScheme)
				}
			}
			return setFromString(f.value, authResult.TokenString, f.tag)
		}

		// If header not found and cookie tag exists, try cookie
		if !fromHeader && cookieName != "" {
			if cookie, err := r.Cookie(cookieName); err == nil && cookie != nil && cookie.Value != "" {
				val = cookie.Value
				fromHeader = false
			}
		}

		// If still no value, check for default
		if val == "" {
			if d, ok := f.tag.Lookup("default"); ok {
				val = d
			} else {
				return nil
			}
		}

		// Handle specific scheme for non-Authorization headers
		if scheme := f.tag.Get("scheme"); scheme != "" && fromHeader {
			prefix := scheme + " "
			if !strings.HasPrefix(val, prefix) {
				return fmt.Errorf("invalid %s header scheme", h)
			}
			val = strings.TrimPrefix(val, prefix)
		}
		return setFromString(f.value, val, f.tag)
	})
}

func bindFromPath(r *http.Request, dst any) error {
	get := PathExtractor(r)
	return walkStruct(dst, func(f fieldInfo) error {
		key := f.tag.Get("path")
		if key == "" {
			return nil
		}
		val := get(key)
		if val == "" {
			if d, ok := f.tag.Lookup("default"); ok {
				val = d
			} else {
				return nil
			}
		}
		return setFromString(f.value, val, f.tag)
	})
}

func bindFromQuery(u *url.URL, dst any) error {
	q := u.Query()
	return walkStruct(dst, func(f fieldInfo) error {
		key := f.tag.Get("query")
		if key == "" {
			return nil
		}
		val := q.Get(key)
		if val == "" {
			if d, ok := f.tag.Lookup("default"); ok {
				val = d
			} else {
				return nil
			}
		}
		if f.value.Kind() == reflect.Slice && f.value.Type().Elem().Kind() == reflect.String {
			values := q[key]
			if len(values) == 0 && val != "" {
				values = strings.Split(val, ",")
			}
			f.value.Set(reflect.ValueOf(values))
			return nil
		}
		return setFromString(f.value, val, f.tag)
	})
}

type fieldInfo struct {
	value reflect.Value
	tag   reflect.StructTag
}

func walkStruct(dst any, fn func(fieldInfo) error) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}
	rv = rv.Elem()
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return errors.New("destination must point to a struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		fv := rv.Field(i)

		if sf.Anonymous || (sf.Type.Kind() == reflect.Struct && fv.CanAddr()) {
			if err := walkStruct(fv.Addr().Interface(), fn); err != nil {
				return err
			}
			continue
		}
		if sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			if err := walkStruct(fv.Interface(), fn); err != nil {
				return err
			}
			continue
		}

		if err := fn(fieldInfo{value: fv, tag: sf.Tag}); err != nil {
			return err
		}
	}
	return nil
}

func setFromString(v reflect.Value, s string, tag reflect.StructTag) error {
	if !v.CanSet() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setFromString(v.Elem(), s, tag)
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Type().PkgPath() == "time" && v.Type().Name() == "Duration" {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}
			v.SetInt(int64(d))
			return nil
		}
		i, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
	case reflect.Struct:
		if v.Type().PkgPath() == "time" && v.Type().Name() == "Time" {
			layout := tag.Get("time_layout")
			if layout == "" {
				layout = time.RFC3339
			}
			t, err := time.Parse(layout, s)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(t))
			return nil
		}
		return fmt.Errorf("unsupported string->struct conversion for %s", v.Type().String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			v.Set(reflect.ValueOf(strings.Split(s, ",")))
			return nil
		}
		return fmt.Errorf("unsupported slice element type: %s", v.Type().Elem().String())
	default:
		return fmt.Errorf("unsupported kind %s for tag-based binding", v.Kind())
	}
	return nil
}

func allocIfPtr[T any](x T) T {
	rv := reflect.ValueOf(&x).Elem()
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}
	return x
}

func defaultPathExtractor(r *http.Request) func(string) string {
	if fn := reflectHTTPRouterParam(r); fn != nil {
		return func(name string) string { return fn(name) }
	}
	if pm := tryContextPathMap(r.Context()); pm != nil {
		return func(name string) string { return pm[name] }
	}
	return func(string) string { return "" }
}

func reflectHTTPRouterParam(r *http.Request) func(string) string {
	type byNamer interface{ ByName(string) string }
	if v := r.Context().Value("httprouter.params"); v != nil {
		if p, ok := v.(byNamer); ok {
			return p.ByName
		}
	}
	return nil
}

func tryContextPathMap(ctx context.Context) map[string]string {
	if v := ctx.Value(apicontext.PathParamsKey); v != nil {
		if m, ok := v.(map[string]string); ok {
			return m
		}
	}
	return nil
}
