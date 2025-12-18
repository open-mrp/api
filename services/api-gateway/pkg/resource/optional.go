package apiresource

import "encoding/json"

type Optional[T any] struct {
	set bool
	val *T
}

func (o *Optional[T]) Set(v T)    { o.set, o.val = true, &v }
func (o *Optional[T]) Null()      { o.set, o.val = true, nil }
func (o *Optional[T]) Unset()     { o.set, o.val = false, nil }
func (o Optional[T]) IsSet() bool { return o.set }
func (o Optional[T]) Ptr() *T     { return o.val }

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.set = true
	if string(b) == "null" {
		o.val = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.val = &v
	return nil
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.set {
		return []byte("null"), nil
	}
	if o.val == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.val)
}
