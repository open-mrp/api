package types

import (
	"reflect"
	"testing"
)

func TestPermissionsNotHeld(t *testing.T) {
	held := map[string]bool{"products:read": true, "orders:read": true}
	granted := map[string]bool{"products:read": true, "users:create": true, "roles:update": true, "orders:update": false}

	got := PermissionsNotHeld(held, granted)
	if want := []string{"roles:update", "users:create"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PermissionsNotHeld = %v, want %v", got, want)
	}
	if got := PermissionsNotHeld(held, map[string]bool{"products:read": true}); len(got) != 0 {
		t.Fatalf("a subset grants nothing new, got %v", got)
	}
}
