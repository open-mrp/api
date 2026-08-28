package field

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

// These tests exercise the real decode path: json.Unmarshal into a struct whose field is a
// value Clearable[T] / Optional[T]. This is what the HTTP layer does, and it is where the
// value-vs-pointer distinction matters — a value field is addressable so encoding/json invokes
// UnmarshalJSON even for an explicit null, whereas a pointer field would short-circuit null to
// nil without calling it.

type clearableHolder[T any] struct {
	F Clearable[T] `json:"f,omitzero"`
}

type optionalHolder[T any] struct {
	F Optional[T] `json:"f,omitzero"`
}

func decodeClearable[T any](t *testing.T, body string) (Clearable[T], error) {
	t.Helper()
	var h clearableHolder[T]
	err := json.Unmarshal([]byte(body), &h)
	return h.F, err
}

func decodeOptional[T any](t *testing.T, body string) (Optional[T], error) {
	t.Helper()
	var h optionalHolder[T]
	err := json.Unmarshal([]byte(body), &h)
	return h.F, err
}

type innerObj struct {
	A string `json:"a"`
	B int    `json:"b"`
}

var refTime = time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)

// ---- Clearable: three states across inner types ----

func TestClearable_decode_threeStates(t *testing.T) {
	t.Parallel()

	// absentBody is "{}" for every type; nullBody is the same; only the value body and
	// expected value vary per inner type. Each case is checked through json.Unmarshal.
	t.Run("string", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":"hi"}`, "hi", func(b string) (any, error) {
			f, err := decodeClearable[string](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[string])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":42}`, 42, func(b string) (any, error) {
			f, err := decodeClearable[int](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[int])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		// false is the zero value; this confirms a set-false survives and is not treated as unset.
		assertClearableStates(t, `{"f":false}`, false, func(b string) (any, error) {
			f, err := decodeClearable[bool](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[bool])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("float64", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":3.5}`, 3.5, func(b string) (any, error) {
			f, err := decodeClearable[float64](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[float64])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":["x","y"]}`, []string{"x", "y"}, func(b string) (any, error) {
			f, err := decodeClearable[[]string](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[[]string])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":{"a":"x","b":7}}`, innerObj{A: "x", B: 7}, func(b string) (any, error) {
			f, err := decodeClearable[innerObj](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[innerObj])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})

	t.Run("time", func(t *testing.T) {
		t.Parallel()
		assertClearableStates(t, `{"f":"2026-06-04T12:30:00Z"}`, refTime, func(b string) (any, error) {
			f, err := decodeClearable[time.Time](t, b)
			return f, err
		}, func(f any) (bool, bool, bool, any) {
			c := f.(Clearable[time.Time])
			v, _ := c.Value()
			return c.IsUnset(), c.IsClear(), c.IsSet(), v
		})
	})
}

// assertClearableStates runs the absent / null / value cases for one inner type.
// decode parses a body into a wrapped field; inspect reports (unset, clear, set, value).
func assertClearableStates(
	t *testing.T,
	valueBody string,
	wantValue any,
	decode func(body string) (any, error),
	inspect func(any) (unset, clear, set bool, value any),
) {
	t.Helper()

	absent, err := decode(`{}`)
	if err != nil {
		t.Fatalf("absent: unexpected error: %v", err)
	}
	if u, c, s, _ := inspect(absent); !u || c || s {
		t.Fatalf("absent: want unset, got unset=%v clear=%v set=%v", u, c, s)
	}

	cleared, err := decode(`{"f":null}`)
	if err != nil {
		t.Fatalf("null: unexpected error: %v", err)
	}
	if u, c, s, _ := inspect(cleared); u || !c || s {
		t.Fatalf("null: want clear, got unset=%v clear=%v set=%v", u, c, s)
	}

	setF, err := decode(valueBody)
	if err != nil {
		t.Fatalf("value: unexpected error: %v", err)
	}
	u, c, s, v := inspect(setF)
	if u || c || !s {
		t.Fatalf("value: want set, got unset=%v clear=%v set=%v", u, c, s)
	}
	if !reflect.DeepEqual(v, wantValue) {
		t.Fatalf("value: got %#v, want %#v", v, wantValue)
	}
}

// ---- Optional: states across inner types ----

func TestOptional_decode_states(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":"hi"}`, "hi", func(b string) (any, error) {
			f, err := decodeOptional[string](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[string])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":42}`, 42, func(b string) (any, error) {
			f, err := decodeOptional[int](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[int])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":false}`, false, func(b string) (any, error) {
			f, err := decodeOptional[bool](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[bool])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":["x","y"]}`, []string{"x", "y"}, func(b string) (any, error) {
			f, err := decodeOptional[[]string](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[[]string])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":{"a":"x","b":7}}`, innerObj{A: "x", B: 7}, func(b string) (any, error) {
			f, err := decodeOptional[innerObj](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[innerObj])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})

	t.Run("time", func(t *testing.T) {
		t.Parallel()
		assertOptionalStates(t, `{"f":"2026-06-04T12:30:00Z"}`, refTime, func(b string) (any, error) {
			f, err := decodeOptional[time.Time](t, b)
			return f, err
		}, func(f any) (bool, bool, any) {
			o := f.(Optional[time.Time])
			v, _ := o.Value()
			return o.IsUnset(), o.IsSet(), v
		})
	})
}

func assertOptionalStates(
	t *testing.T,
	valueBody string,
	wantValue any,
	decode func(body string) (any, error),
	inspect func(any) (unset, set bool, value any),
) {
	t.Helper()

	absent, err := decode(`{}`)
	if err != nil {
		t.Fatalf("absent: unexpected error: %v", err)
	}
	if u, s, _ := inspect(absent); !u || s {
		t.Fatalf("absent: want unset, got unset=%v set=%v", u, s)
	}

	setF, err := decode(valueBody)
	if err != nil {
		t.Fatalf("value: unexpected error: %v", err)
	}
	u, s, v := inspect(setF)
	if u || !s {
		t.Fatalf("value: want set, got unset=%v set=%v", u, s)
	}
	if !reflect.DeepEqual(v, wantValue) {
		t.Fatalf("value: got %#v, want %#v", v, wantValue)
	}
}

// ---- Optional: explicit null is rejected through json.Unmarshal ----

func TestOptional_decode_nullRejected(t *testing.T) {
	t.Parallel()

	_, err := decodeOptional[string](t, `{"f":null}`)
	if !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("want ErrExplicitNull, got %v", err)
	}

	// Same for a non-string inner type.
	_, err = decodeOptional[int](t, `{"f":null}`)
	if !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("want ErrExplicitNull for int, got %v", err)
	}
}

// TestOptional_decode_blankStringSets documents that a blank string is a *set* at the decode
// layer (Optional only rejects null here); blank rejection is a separate validation pass.
func TestOptional_decode_blankStringSets(t *testing.T) {
	t.Parallel()

	f, err := decodeOptional[string](t, `{"f":""}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := f.Value()
	if !ok || v != "" {
		t.Fatalf("want set to blank, got %q ok=%v", v, ok)
	}
}

// ---- Type mismatches surface as decode errors ----

func TestClearable_decode_typeMismatch(t *testing.T) {
	t.Parallel()
	if _, err := decodeClearable[int](t, `{"f":"not-an-int"}`); err == nil {
		t.Fatal("expected error decoding string into Clearable[int]")
	}
}

func TestOptional_decode_typeMismatch(t *testing.T) {
	t.Parallel()
	if _, err := decodeOptional[int](t, `{"f":"not-an-int"}`); err == nil {
		t.Fatal("expected error decoding string into Optional[int]")
	}
}

// ---- Round trips: marshal then unmarshal preserves the state ----

func TestClearable_roundTrip(t *testing.T) {
	t.Parallel()

	cases := map[string]Clearable[string]{
		"unset": Unset[string](),
		"clear": Clear[string](),
		"set":   Set("value"),
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(clearableHolder[string]{F: want})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got clearableHolder[string]
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got.F.IsUnset() != want.IsUnset() ||
				got.F.IsClear() != want.IsClear() ||
				got.F.IsSet() != want.IsSet() {
				t.Fatalf("state mismatch after round trip via %s: got %+v want %+v", b, got.F, want)
			}
			if want.IsSet() {
				gv, _ := got.F.Value()
				wv, _ := want.Value()
				if gv != wv {
					t.Fatalf("value mismatch: got %q want %q", gv, wv)
				}
			}
		})
	}
}

func TestOptional_roundTrip(t *testing.T) {
	t.Parallel()

	// Unset omits the key (omitzero); set round-trips the value. Optional cannot represent clear.
	for name, want := range map[string]Optional[string]{
		"unset": None[string](),
		"set":   Some("value"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(optionalHolder[string]{F: want})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got optionalHolder[string]
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got.F.IsSet() != want.IsSet() {
				t.Fatalf("state mismatch after round trip via %s: got %+v want %+v", b, got.F, want)
			}
			if want.IsSet() {
				gv, _ := got.F.Value()
				wv, _ := want.Value()
				if gv != wv {
					t.Fatalf("value mismatch: got %q want %q", gv, wv)
				}
			}
		})
	}
}

// ---- Embedded structs decode correctly ----

func TestClearable_decode_embeddedStruct(t *testing.T) {
	t.Parallel()

	type base struct {
		Note Clearable[string] `json:"note,omitzero"`
	}
	type req struct {
		base
		Name Optional[string] `json:"name,omitzero"`
	}

	var r req
	if err := json.Unmarshal([]byte(`{"name":"bob","note":null}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !r.Note.IsClear() {
		t.Fatalf("embedded Note: want clear, got %+v", r.Note)
	}
	if v, ok := r.Name.Value(); !ok || v != "bob" {
		t.Fatalf("Name: got %q ok=%v", v, ok)
	}
}

// ---- Non-canonical null in the request body ----

// TestClearable_decode_paddedNullClears covers the real decode path for a body whose null
// carries surrounding whitespace: encoding/json hands UnmarshalJSON the trimmed literal, so the
// field still clears. Clearable.UnmarshalJSON's own comparison is byte-exact, so this property
// belongs to the decoder, not to the wrapper.
func TestClearable_decode_paddedNullClears(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"spaces around null":  "{\"f\":  null  }",
		"newline before null": "{\"f\":\n null\n}",
		"tab before null":     "{\"f\":\tnull}",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, err := decodeClearable[string](t, body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !f.IsClear() {
				t.Fatalf("want clear, got %+v", f)
			}
		})
	}
}

// TestOptional_unmarshalJSON_trimsBeforeNullCheck pins that Optional rejects a non-canonical
// null too, so a direct caller cannot slip one past the check.
func TestOptional_unmarshalJSON_trimsBeforeNullCheck(t *testing.T) {
	t.Parallel()

	for _, data := range []string{"null", " null ", "null\n", "\tnull"} {
		var n Optional[string]
		if err := n.UnmarshalJSON([]byte(data)); !errors.Is(err, ErrExplicitNull) {
			t.Fatalf("%q: want ErrExplicitNull, got %v", data, err)
		}
		if n.IsSet() {
			t.Fatalf("%q: must stay unset after a rejected null", data)
		}
	}
}

func TestOptional_decode_paddedNullRejected(t *testing.T) {
	t.Parallel()

	if _, err := decodeOptional[string](t, "{\"f\":  null  }"); !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("want ErrExplicitNull, got %v", err)
	}
}

// ---- Set values whose JSON form is null ----

// TestClearable_marshal_nilInnerIsNull records the collision behind the job_items hazard: a set
// field holding a nil slice, map or pointer marshals to the same "null" that means clear, so the
// state cannot survive a JSON hop. Carry an empty non-nil value instead.
func TestClearable_marshal_nilInnerIsNull(t *testing.T) {
	t.Parallel()

	slice, err := json.Marshal(clearableHolder[[]string]{F: Set[[]string](nil)})
	if err != nil {
		t.Fatalf("marshal slice: %v", err)
	}
	if string(slice) != `{"f":null}` {
		t.Fatalf("nil slice: got %s, want {\"f\":null}", slice)
	}

	m, err := json.Marshal(clearableHolder[map[string]string]{F: Set[map[string]string](nil)})
	if err != nil {
		t.Fatalf("marshal map: %v", err)
	}
	if string(m) != `{"f":null}` {
		t.Fatalf("nil map: got %s, want {\"f\":null}", m)
	}

	ptr, err := json.Marshal(optionalHolder[*string]{F: Some[*string](nil)})
	if err != nil {
		t.Fatalf("marshal pointer: %v", err)
	}
	if string(ptr) != `{"f":null}` {
		t.Fatalf("nil pointer: got %s, want {\"f\":null}", ptr)
	}
}

// TestClearable_roundTrip_nilInnerLosesTheSetState completes the case above through the decode
// half, which is where the job_items hop actually bites: a set-to-nil slice comes back CLEAR, and
// a set-to-nil pointer on an Optional comes back as a decode failure. JSON has one null, so no
// wrapper can carry these; resolved rows must hold an empty non-nil value instead.
func TestClearable_roundTrip_nilInnerLosesTheSetState(t *testing.T) {
	t.Parallel()

	t.Run("clearable nil slice decodes as clear", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(clearableHolder[[]string]{F: Set[[]string](nil)})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got clearableHolder[[]string]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if !got.F.IsClear() {
			t.Fatalf("via %s: want clear, got %+v", b, got.F)
		}
	})

	t.Run("clearable nil map decodes as clear", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(clearableHolder[map[string]string]{F: Set[map[string]string](nil)})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got clearableHolder[map[string]string]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if !got.F.IsClear() {
			t.Fatalf("via %s: want clear, got %+v", b, got.F)
		}
	})

	t.Run("optional nil pointer fails to decode", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(optionalHolder[*string]{F: Some[*string](nil)})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got optionalHolder[*string]
		if err := json.Unmarshal(b, &got); !errors.Is(err, ErrExplicitNull) {
			t.Fatalf("via %s: want ErrExplicitNull, got %v", b, err)
		}
	})
}

// TestClearable_roundTrip_emptyNonNilInner is the safe form of the case above: an allocated but
// empty slice or map marshals to [] / {} and comes back set.
func TestClearable_roundTrip_emptyNonNilInner(t *testing.T) {
	t.Parallel()

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(clearableHolder[[]string]{F: Set([]string{})})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(b) != `{"f":[]}` {
			t.Fatalf("got %s, want {\"f\":[]}", b)
		}
		var got clearableHolder[[]string]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		v, ok := got.F.Value()
		if !ok || len(v) != 0 {
			t.Fatalf("want set to an empty slice, got %#v ok=%v", v, ok)
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(optionalHolder[map[string]string]{F: Some(map[string]string{})})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(b) != `{"f":{}}` {
			t.Fatalf("got %s, want {\"f\":{}}", b)
		}
		var got optionalHolder[map[string]string]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		v, ok := got.F.Value()
		if !ok || len(v) != 0 {
			t.Fatalf("want set to an empty map, got %#v ok=%v", v, ok)
		}
	})
}

// TestClearable_roundTrip_innerTypes extends the string-only round trip to the inner types that
// travel through the job_items JSON hop.
func TestClearable_roundTrip_innerTypes(t *testing.T) {
	t.Parallel()

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[[]string]{
			"unset": Unset[[]string](),
			"clear": Clear[[]string](),
			"set":   Set([]string{"a", "b"}),
		} {
			b, err := json.Marshal(clearableHolder[[]string]{F: want})
			if err != nil {
				t.Fatalf("%s: marshal: %v", name, err)
			}
			var got clearableHolder[[]string]
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("%s: unmarshal %s: %v", name, b, err)
			}
			if got.F.IsUnset() != want.IsUnset() || got.F.IsClear() != want.IsClear() || got.F.IsSet() != want.IsSet() {
				t.Fatalf("%s: state mismatch via %s: got %+v want %+v", name, b, got.F, want)
			}
			gv, _ := got.F.Value()
			wv, _ := want.Value()
			if !reflect.DeepEqual(gv, wv) {
				t.Fatalf("%s: value got %#v want %#v", name, gv, wv)
			}
		}
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		want := Set(innerObj{A: "x", B: 7})
		b, err := json.Marshal(clearableHolder[innerObj]{F: want})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got clearableHolder[innerObj]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		gv, ok := got.F.Value()
		if !ok || gv != (innerObj{A: "x", B: 7}) {
			t.Fatalf("got %#v ok=%v", gv, ok)
		}
	})

	t.Run("time", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(clearableHolder[time.Time]{F: Set(refTime)})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got clearableHolder[time.Time]
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		gv, ok := got.F.Value()
		if !ok || !gv.Equal(refTime) {
			t.Fatalf("got %v ok=%v, want %v", gv, ok, refTime)
		}
	})
}

// TestClearable_marshalUnset_withoutOmitzero shows what a missing omitzero costs: the whole
// response or job payload fails to marshal, since an unset field has no JSON form.
func TestClearable_marshalUnset_withoutOmitzero(t *testing.T) {
	t.Parallel()

	type noOmitzero struct {
		F Clearable[string] `json:"f"`
	}
	if _, err := json.Marshal(noOmitzero{}); err == nil {
		t.Fatal("expected an error marshaling an unset field declared without omitzero")
	}

	type optionalNoOmitzero struct {
		F Optional[string] `json:"f"`
	}
	if _, err := json.Marshal(optionalNoOmitzero{}); err == nil {
		t.Fatal("expected an error marshaling an unset Optional declared without omitzero")
	}
}
