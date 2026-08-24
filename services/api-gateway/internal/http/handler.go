package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/header"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/fuzzy"
)

var PathExtractor func(*http.Request) func(string) string = defaultPathExtractor

const (
	AttrHTTPStatusCode = "http.status_code"
	AttrErrorType      = "error.type"
)

func GetIdentity(ctx context.Context) (*types.Identity, *apierror.APIError) {
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		return identity, nil
	}
	return nil, apierror.NewAuthenticationError("Identity not found in context")
}

func ShouldDecodeBody(r *http.Request) bool {
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

func DecodeJSONInto(dst any, r *http.Request, disallowUnknown bool) error {
	dec := json.NewDecoder(r.Body)
	if disallowUnknown {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		if field, ok := extractUnknownJSONFieldName(err); ok {
			candidates := collectJSONFieldNames(dst)
			msg := fmt.Sprintf("Invalid JSON in request body: unknown field '%s'.", field)
			if suggestion, dist := fuzzy.FindClosestByLevenshtein(field, candidates); suggestion != "" && dist <= 3 {
				msg = fmt.Sprintf("Invalid JSON in request body: unknown field '%s'. Did you mean '%s'?", field, suggestion)
			}
			return apierror.NewParameterUnknownError(msg, field)
		}

		if uterr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
			msg := fmt.Sprintf("Invalid type for field '%s': expected %s, got %s", uterr.Field, uterr.Type.String(), uterr.Value)
			return apierror.NewInvalidFormatError(msg, uterr.Field)
		}

		if serr, ok := errors.AsType[*json.SyntaxError](err); ok {
			return apierror.NewValidationError(fmt.Sprintf("Invalid JSON in request body at offset %d: %v", serr.Offset, serr.Error()))
		}

		if apiErr, ok := errors.AsType[*apierror.APIError](err); ok {
			return apiErr
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

// extractUnknownJSONFieldName tries to parse the offending field name from a json decoder error when DisallowUnknownFields is enabled. It returns the field name and true if found.
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
	before, _, ok := strings.Cut(rest, "\"")
	if !ok {
		return "", false
	}
	field := before
	if field == "" {
		return "", false
	}
	return field, true
}

// collectJSONFieldNames returns the set of acceptable JSON keys for the provided destination struct. It considers json tags; if absent, it falls back to the exported field name.
func collectJSONFieldNames(dst any) []string {
	t := reflect.TypeOf(dst)
	for t.Kind() == reflect.Pointer {
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
		for sf := range rt.Fields() {
			if sf.PkgPath != "" { // unexported
				continue
			}
			// Handle anonymous embedded structs (inline in JSON)
			if sf.Anonymous {
				ft := sf.Type
				for ft.Kind() == reflect.Pointer {
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
				before, _, ok := strings.Cut(tag, ",")
				if ok {
					name = before
				} else {
					name = tag
				}
			}
			if name == "" { // fallback to field name if no tag name component
				name = sf.Name
				if name == "" {
					continue
				}
				// encoding/json matches field names case-insensitively, but suggestions should prefer the typical JSON style; we keep the declared name.
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

// bindHeadersInto binds r's headers (and optionally cookies) into fv using struct tags.
func bindHeadersInto(r *http.Request, fv reflect.Value, tag reflect.StructTag) error {
	h := tag.Get("header")
	cookieName := tag.Get("cookie")

	var val string
	var fromHeader bool
	if h != "" {
		val = r.Header.Get(h)
		fromHeader = val != ""
	}

	if h == "Authorization" && fromHeader {
		scheme := tag.Get("scheme")
		authResult, err := header.ValidateAndExtractAuthHeader(val)
		if err != nil {
			if cookieName != "" {
				if cookie, cookieErr := r.Cookie(cookieName); cookieErr == nil && cookie != nil && cookie.Value != "" {
					return setFromString(fv, cookie.Value, tag)
				}
			}
			return err.WithParam(h)
		}

		if scheme != "" {
			expectedScheme := strings.ToLower(scheme)
			actualScheme := strings.ToLower(string(authResult.Scheme))
			if expectedScheme != actualScheme {
				if cookieName != "" {
					if cookie, cookieErr := r.Cookie(cookieName); cookieErr == nil && cookie != nil && cookie.Value != "" {
						return setFromString(fv, cookie.Value, tag)
					}
				}
				return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid %s header scheme: expected %s, got %s", h, expectedScheme, actualScheme), h)
			}
		}
		return setFromString(fv, authResult.TokenString, tag)
	}

	if !fromHeader && cookieName != "" {
		if cookie, err := r.Cookie(cookieName); err == nil && cookie != nil && cookie.Value != "" {
			val = cookie.Value
			fromHeader = false
		}
	}

	if val == "" {
		if d, ok := tag.Lookup("default"); ok {
			val = d
		} else {
			return nil
		}
	}

	if scheme := tag.Get("scheme"); scheme != "" && fromHeader {
		prefix := scheme + " "
		if !strings.HasPrefix(val, prefix) {
			return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid %s header scheme", h), h)
		}
		val = strings.TrimPrefix(val, prefix)
	}
	if err := setFromString(fv, val, tag); err != nil {
		param := h
		source := "header"
		if param == "" {
			param = cookieName
			source = "cookie"
		}
		if param == "" {
			return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid value: %v", err), "")
		}
		return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid value for %s '%s': %v", source, param, err), param)
	}
	return nil
}

func bindPathInto(get func(string) string, fv reflect.Value, tag reflect.StructTag) error {
	key := tag.Get("path")
	if key == "" {
		return nil
	}
	val := get(key)
	if val == "" {
		if d, ok := tag.Lookup("default"); ok {
			val = d
		} else {
			return nil
		}
	}
	if err := setFromString(fv, val, tag); err != nil {
		return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid value for path parameter '%s': %v", key, err), key)
	}
	return nil
}

func bindQueryInto(q url.Values, fv reflect.Value, f bindField) error {
	tag := f.tag
	key := tag.Get("query")
	if key == "" {
		return nil
	}
	if f.isSlice {
		values := append([]string{}, q[key]...)
		values = append(values, q[key+"[]"]...)
		if len(values) == 0 {
			if d, ok := tag.Lookup("default"); ok && d != "" {
				values = strings.Split(d, ",")
			} else {
				return nil
			}
		} else if len(values) == 1 && strings.Contains(values[0], ",") {
			values = strings.Split(values[0], ",")
		}
		elemType := f.fieldTyp.Elem()
		slice := reflect.MakeSlice(f.fieldTyp, len(values), len(values))
		for i, v := range values {
			if elemType.Kind() == reflect.String {
				slice.Index(i).Set(reflect.ValueOf(v).Convert(elemType))
				continue
			}
			if err := setFromString(slice.Index(i), v, tag); err != nil {
				return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid value for query parameter '%s': %v", key, err), key)
			}
		}
		fv.Set(slice)
		return nil
	}
	val := q.Get(key)
	if val == "" {
		if d, ok := tag.Lookup("default"); ok {
			val = d
		} else {
			return nil
		}
	}
	if err := setFromString(fv, val, tag); err != nil {
		return apierror.NewParameterInvalidError(fmt.Sprintf("Invalid value for query parameter '%s': %v", key, err), key)
	}
	return nil
}

func rejectUnknownAgainstPlan(plan *bindPlan, u *url.URL, allowInclude bool) *apierror.APIError {
	allowed := plan.allowedQuery
	if allowInclude {
		allowed = maps.Clone(allowed)
		allowed["include"] = struct{}{}
		allowed["include[]"] = struct{}{}
	}
	for key := range u.Query() {
		if _, ok := allowed[key]; !ok {
			return apierror.NewParameterUnknownError(
				fmt.Sprintf("Unknown query parameter '%s'.", key),
				key,
			)
		}
	}
	return nil
}

// BindIncomingRequest binds header, path, and query parameters with one traversal of the cached bind plan. For each field that declares binding tags it applies headers/cookies first, path second, and query third, matching BindFromHeaders + BindFromPath + BindFromQuery on the same destination.
func BindIncomingRequest(r *http.Request, dst any, allowIncludeQueryKeys bool) error {
	plan, err := planFor(dst)
	if err != nil {
		return err
	}
	get := PathExtractor(r)
	q := r.URL.Query()
	root := reflect.ValueOf(dst).Elem()
	for _, f := range plan.fields {
		tag := f.tag
		wantsHeader := tag.Get("header") != "" || tag.Get("cookie") != ""
		wantsPath := tag.Get("path") != ""
		wantsQuery := tag.Get("query") != ""
		if !wantsHeader && !wantsPath && !wantsQuery {
			continue
		}

		fv, ok := navigateBindField(root, f)
		if !ok || !fv.CanSet() {
			continue
		}

		if wantsHeader {
			if err := bindHeadersInto(r, fv, tag); err != nil {
				return err
			}
		}
		if wantsPath {
			if err := bindPathInto(get, fv, tag); err != nil {
				return err
			}
		}
		if wantsQuery {
			if err := bindQueryInto(q, fv, f); err != nil {
				return err
			}
		}
	}

	if apiErr := rejectUnknownAgainstPlan(plan, r.URL, allowIncludeQueryKeys); apiErr != nil {
		return apiErr
	}
	return nil
}

func BindFromHeaders(r *http.Request, dst any) error {
	plan, err := planFor(dst)
	if err != nil {
		return err
	}
	root := reflect.ValueOf(dst).Elem()
	for _, f := range plan.fields {
		tag := f.tag
		if tag.Get("header") == "" && tag.Get("cookie") == "" {
			continue
		}
		fv, ok := navigateBindField(root, f)
		if !ok || !fv.CanSet() {
			continue
		}
		if err := bindHeadersInto(r, fv, tag); err != nil {
			return err
		}
	}
	return nil
}

func BindFromPath(r *http.Request, dst any) error {
	plan, err := planFor(dst)
	if err != nil {
		return err
	}
	get := PathExtractor(r)
	root := reflect.ValueOf(dst).Elem()
	for _, f := range plan.fields {
		tag := f.tag
		if tag.Get("path") == "" {
			continue
		}
		fv, ok := navigateBindField(root, f)
		if !ok || !fv.CanSet() {
			continue
		}
		if err := bindPathInto(get, fv, tag); err != nil {
			return err
		}
	}
	return nil
}

const maxRawBodySize = 1 << 20 // 1MB limit for raw body

func BindRawBody(r *http.Request, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}
	rv = rv.Elem()
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return errors.New("destination must point to a struct")
	}

	var bodyRead bool
	var bodyData []byte

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		tag := sf.Tag.Get("rawbody")
		if tag == "" {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		if fv.Type() != reflect.TypeFor[[]byte]() {
			return fmt.Errorf("rawbody tag can only be used on []byte fields, got %s", fv.Type().String())
		}

		if !bodyRead {
			var err error
			bodyData, err = io.ReadAll(io.LimitReader(r.Body, maxRawBodySize))
			if err != nil {
				return fmt.Errorf("failed to read request body: %w", err)
			}
			bodyRead = true
		}

		fv.SetBytes(bodyData)
	}

	return nil
}

// RejectUnknownQueryParams returns an error when the URL contains query keys that are not declared on the request struct (via `query` tags). Slice parameters accept either ?key= or ?key[]= shapes; both key forms are treated as allowed. When allowInclude is true, include and include[] are permitted (validated separately by the endpoint).
func RejectUnknownQueryParams(u *url.URL, dst any, allowInclude bool) *apierror.APIError {
	plan, err := planFor(dst)
	if err != nil {
		return apierror.NewInvariantViolationError(err.Error())
	}
	return rejectUnknownAgainstPlan(plan, u, allowInclude)
}

func BindFromQuery(u *url.URL, dst any) error {
	plan, err := planFor(dst)
	if err != nil {
		return err
	}
	q := u.Query()
	root := reflect.ValueOf(dst).Elem()
	for _, f := range plan.fields {
		tag := f.tag
		if tag.Get("query") == "" {
			continue
		}
		fv, ok := navigateBindField(root, f)
		if !ok || !fv.CanSet() {
			continue
		}
		if err := bindQueryInto(q, fv, f); err != nil {
			return err
		}
	}
	return nil
}

func setFromString(v reflect.Value, s string, tag reflect.StructTag) error {
	if !v.CanSet() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
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
			parts := strings.Split(s, ",")
			elemType := v.Type().Elem()
			slice := reflect.MakeSlice(v.Type(), len(parts), len(parts))
			for i, p := range parts {
				slice.Index(i).Set(reflect.ValueOf(p).Convert(elemType))
			}
			v.Set(slice)
			return nil
		}
		return fmt.Errorf("unsupported slice element type: %s", v.Type().Elem().String())
	default:
		return fmt.Errorf("unsupported kind %s for tag-based binding", v.Kind())
	}
	return nil
}

func AllocIfPtr[T any](x T) T {
	rv := reflect.ValueOf(&x).Elem()
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
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

// unwrapEnumWrapper extracts the inner value of a field.Optional[T]/field.Clearable[T] for enum validation. It returns an addressable copy of the wrapped value and true when the wrapper holds a concrete value (IsSet); unset or cleared wrappers return false so the caller skips them.
func unwrapEnumWrapper(fv reflect.Value) (reflect.Value, bool) {
	if !fv.CanAddr() {
		// Methods are value receivers, but Value() returns a copy we cannot address; make an addressable copy so MethodByName can be called.
		tmp := reflect.New(fv.Type())
		tmp.Elem().Set(fv)
		fv = tmp.Elem()
	}
	if m := fv.MethodByName("IsSet"); m.IsValid() {
		out := m.Call(nil)
		if len(out) == 1 && out[0].Kind() == reflect.Bool && !out[0].Bool() {
			return reflect.Value{}, false
		}
	}
	valM := fv.MethodByName("Value")
	if !valM.IsValid() {
		return reflect.Value{}, false
	}
	out := valM.Call(nil)
	if len(out) != 2 || out[1].Kind() != reflect.Bool || !out[1].Bool() {
		return reflect.Value{}, false
	}
	inner := out[0]
	addr := reflect.New(inner.Type())
	addr.Elem().Set(inner)
	return addr.Elem(), true
}

func tryContextPathMap(ctx context.Context) map[string]string {
	if m, ok := appctx.GetPathParams(ctx); ok {
		return m
	}
	return nil
}

// enumParamName resolves the public parameter name for an enum field, preferring the json tag (request body) then the query tag (query parameter), falling back to the exported Go field name. Query-param structs tag fields with `query:"..."`, so without this the reported param would be the Go field name (e.g. "Status" instead of "statuses").
// enumParamName is the name the caller used for a field, taken from whichever tag describes how
// they sent it. Falling through to the Go field name puts an identifier in the error that the
// caller has never seen and cannot map back to anything they wrote.
func enumParamName(sf reflect.StructField) string {
	for _, tag := range []string{"json", "query", "path", "header", "form", "cookie"} {
		if v := sf.Tag.Get(tag); v != "" && v != "-" {
			if name := strings.Split(v, ",")[0]; name != "" {
				return name
			}
		}
	}
	return sf.Name
}

// isRequiredField reports whether the field carries the `required` validate tag, so an empty enum can be told apart from an optional one left unset.
func isRequiredField(sf reflect.StructField) bool {
	for _, rule := range strings.Split(sf.Tag.Get("validate"), ",") {
		if rule == "required" {
			return true
		}
	}
	return false
}

// newMissingEnumFieldError mirrors the code that shared/validate would pick for a missing `required` field, keeping a body field a missing_field and a query or path parameter a parameter_missing.
func newMissingEnumFieldError(sf reflect.StructField) *apierror.APIError {
	name := enumParamName(sf)
	if sf.Tag.Get("json") != "" {
		return apierror.NewMissingFieldError(fmt.Sprintf("Field '%s' is required.", name), name)
	}
	return apierror.NewParameterMissingError(fmt.Sprintf("Parameter '%s' is required.", name), name)
}

func ValidateEnumFields(dst any) *apierror.APIError {
	rv := reflect.ValueOf(dst)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		fv := rv.Field(i)
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
			ft = ft.Elem()
		}

		// Unwrap field.Optional[T]/field.Clearable[T] so the wrapped enum value is validated. These are structs with unexported fields, so without unwrapping the recursion below would skip them and the enum check would never run.
		if field.IsOptionalType(ft) || field.IsClearableType(ft) {
			isClearable := field.IsClearableType(ft)
			inner, ok := unwrapEnumWrapper(fv)
			if !ok {
				continue
			}
			// An empty string on a Clearable is the clear signal, not a value to validate.
			if isClearable && inner.Kind() == reflect.String && inner.String() == "" {
				continue
			}
			fv = inner
			ft = inner.Type()
		}

		if ft.Kind() == reflect.Struct {
			if apiErr := ValidateEnumFields(fv.Addr().Interface()); apiErr != nil {
				return apiErr
			}
			continue
		}

		// Enum slice fields (e.g. list filter query params) reject unrecognized values with 400, consistent with scalar enum validation below.
		if ft.Kind() == reflect.Slice {
			et := ft.Elem()
			if et.Kind() != reflect.String || et.Name() == "" || et.Name() == "string" {
				continue
			}
			ptrType := reflect.PointerTo(et)
			method, ok := ptrType.MethodByName("EnumValues")
			if !ok {
				continue
			}
			if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
				continue
			}
			outType := method.Type.Out(0)
			if outType.Kind() != reflect.Slice || outType.Elem().Kind() != reflect.String {
				continue
			}
			for k := 0; k < fv.Len(); k++ {
				elem := fv.Index(k)
				results := method.Func.Call([]reflect.Value{elem.Addr()})
				if len(results) != 1 {
					continue
				}
				validValues := results[0]
				currentValue := elem.String()
				isValid := false
				var allowedValues []string
				for j := 0; j < validValues.Len(); j++ {
					val := validValues.Index(j).String()
					allowedValues = append(allowedValues, val)
					if val == currentValue {
						isValid = true
						break
					}
				}
				if !isValid {
					fieldName := enumParamName(sf)
					return apierror.NewParameterInvalidError(
						fmt.Sprintf("Field '%s' must be one of: %s", fieldName, strings.Join(allowedValues, ", ")),
						fieldName,
					)
				}
			}
			continue
		}

		if ft.Kind() != reflect.String || ft.Name() == "" || ft.Name() == "string" {
			continue
		}

		ptrType := reflect.PointerTo(ft)
		method, ok := ptrType.MethodByName("EnumValues")
		if !ok {
			continue
		}

		if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
			continue
		}

		outType := method.Type.Out(0)
		if outType.Kind() != reflect.Slice || outType.Elem().Kind() != reflect.String {
			continue
		}

		results := method.Func.Call([]reflect.Value{fv.Addr()})
		if len(results) != 1 {
			continue
		}

		validValues := results[0]
		currentValue := fv.String()

		// A required scalar enum is indistinguishable from an absent one when empty, and this check runs ahead of the `required` tag. Reporting it as an unrecognized value would call a missing field invalid, so it gets the same code the tag would have produced.
		if currentValue == "" && isRequiredField(sf) {
			return newMissingEnumFieldError(sf)
		}

		isValid := false
		var allowedValues []string
		for j := 0; j < validValues.Len(); j++ {
			val := validValues.Index(j).String()
			allowedValues = append(allowedValues, val)
			if val == currentValue {
				isValid = true
				break
			}
		}

		if !isValid {
			// Read every tag, not just json: a scalar enum is as often a query filter as a body
			// field, and naming it "Type" instead of "type" told the caller nothing.
			fieldName := enumParamName(sf)
			return apierror.NewParameterInvalidError(
				fmt.Sprintf("Field '%s' must be one of: %s", fieldName, strings.Join(allowedValues, ", ")),
				fieldName,
			)
		}
	}

	return nil
}
