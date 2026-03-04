package apiendpoint

import (
	"testing"

	"github.com/augno/api/shared/constants"
	"github.com/stretchr/testify/assert"
)

func roleConfig() *IncludeConfig {
	return &IncludeConfig{
		Fields: []IncludeField{
			{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"role"}},
		},
	}
}

func requestLogConfig() *IncludeConfig {
	return &IncludeConfig{
		Fields: []IncludeField{
			{Key: "account", ObjectType: constants.ObjectTypeAccount, JSONPaths: []string{"account"}},
			{Key: "actor", ObjectType: constants.ObjectTypeUser, JSONPaths: []string{"actor"}},
			{Key: "actor.role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"actor.role"}},
		},
	}
}

func TestCollapseUnexpanded_SingleObject_Collapsed(t *testing.T) {
	data := map[string]any{
		"id":     "apke_xxx",
		"object": "api_key",
		"name":   "My Key",
		"role": map[string]any{
			"id":             "rl_xxx",
			"object":         "role",
			"name":           "Admin",
			"role_type_code": "admin",
		},
	}

	result := CollapseUnexpanded(data, roleConfig(), map[string]bool{})

	assert.Nil(t, result["role"])
}

func TestCollapseUnexpanded_SingleObject_Expanded(t *testing.T) {
	data := map[string]any{
		"id":     "apke_xxx",
		"object": "api_key",
		"name":   "My Key",
		"role": map[string]any{
			"id":             "rl_xxx",
			"object":         "role",
			"name":           "Admin",
			"role_type_code": "admin",
		},
	}

	result := CollapseUnexpanded(data, roleConfig(), map[string]bool{"role": true})

	role := result["role"].(map[string]any)
	assert.Equal(t, "rl_xxx", role["id"])
	assert.Equal(t, "role", role["object"])
	assert.Equal(t, "Admin", role["name"])
	assert.Equal(t, "admin", role["role_type_code"])
}

func TestCollapseUnexpanded_ListResponse(t *testing.T) {
	data := map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{
				"id": "apke_1",
				"role": map[string]any{
					"id":     "rl_1",
					"object": "role",
					"name":   "Admin",
				},
			},
			map[string]any{
				"id": "apke_2",
				"role": map[string]any{
					"id":     "rl_2",
					"object": "role",
					"name":   "Viewer",
				},
			},
		},
	}

	result := CollapseUnexpanded(data, roleConfig(), map[string]bool{})

	items := result["data"].([]any)
	for _, item := range items {
		m := item.(map[string]any)
		assert.Nil(t, m["role"])
	}
}

func TestCollapseUnexpanded_NestedPath_NoIncludes(t *testing.T) {
	data := map[string]any{
		"id":     "rl_xxx",
		"object": "request_log",
		"actor": map[string]any{
			"id":          "usr_xxx",
			"object_type": "user",
			"role": map[string]any{
				"id":     "rl_yyy",
				"object": "role",
				"name":   "Admin",
			},
		},
	}

	result := CollapseUnexpanded(data, requestLogConfig(), map[string]bool{})

	// actor is collapsed because neither actor nor actor.role is requested.
	assert.Nil(t, result["actor"])
}

func TestCollapseUnexpanded_NestedPath_ActorOnly(t *testing.T) {
	data := map[string]any{
		"id":     "rl_xxx",
		"object": "request_log",
		"actor": map[string]any{
			"id":          "usr_xxx",
			"object_type": "user",
			"role": map[string]any{
				"id":     "rl_yyy",
				"object": "role",
				"name":   "Admin",
			},
		},
	}

	// actor requested but not actor.role — actor kept, role collapsed.
	result := CollapseUnexpanded(data, requestLogConfig(), map[string]bool{"actor": true})

	actor := result["actor"].(map[string]any)
	assert.Equal(t, "usr_xxx", actor["id"])
	assert.Nil(t, actor["role"])
}

func TestCollapseUnexpanded_NilSubObject(t *testing.T) {
	data := map[string]any{
		"id":     "apke_xxx",
		"object": "api_key",
		"role":   nil,
	}

	result := CollapseUnexpanded(data, roleConfig(), map[string]bool{})

	assert.Nil(t, result["role"])
}

func TestCollapseUnexpanded_MissingSubObject(t *testing.T) {
	data := map[string]any{
		"id":     "apke_xxx",
		"object": "api_key",
	}

	result := CollapseUnexpanded(data, roleConfig(), map[string]bool{})

	_, exists := result["role"]
	assert.False(t, exists)
}

func TestCollapseUnexpanded_NestedPathNilParent(t *testing.T) {
	data := map[string]any{
		"id":     "rl_xxx",
		"object": "request_log",
		"actor":  nil,
	}

	result := CollapseUnexpanded(data, requestLogConfig(), map[string]bool{})

	assert.Nil(t, result["actor"])
}

func TestCollapseUnexpanded_MultiLevelNesting(t *testing.T) {
	config := &IncludeConfig{
		Fields: []IncludeField{
			{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"api_key_info.role"}},
		},
	}

	data := map[string]any{
		"api_key_secret": "aug_sk_...",
		"api_key_info": map[string]any{
			"id":     "apke_xxx",
			"object": "api_key",
			"role": map[string]any{
				"id":     "rl_xxx",
				"object": "role",
				"name":   "Admin",
			},
		},
	}

	result := CollapseUnexpanded(data, config, map[string]bool{})

	info := result["api_key_info"].(map[string]any)
	assert.Nil(t, info["role"])
}

func TestCollapseUnexpanded_AllRequested(t *testing.T) {
	data := map[string]any{
		"id":     "rl_xxx",
		"object": "request_log",
		"account": map[string]any{
			"id":     "ac_xxx",
			"object": "account",
			"name":   "Acme Inc.",
		},
		"actor": map[string]any{
			"id":          "usr_xxx",
			"object_type": "user",
			"role": map[string]any{
				"id":     "rl_yyy",
				"object": "role",
				"name":   "Admin",
			},
		},
	}

	result := CollapseUnexpanded(data, requestLogConfig(), map[string]bool{
		"account":    true,
		"actor.role": true,
	})

	account := result["account"].(map[string]any)
	assert.Equal(t, "Acme Inc.", account["name"])

	actor := result["actor"].(map[string]any)
	role := actor["role"].(map[string]any)
	assert.Equal(t, "Admin", role["name"])
}

func TestCollapseUnexpanded_ChildIncludeKeepsParent(t *testing.T) {
	data := map[string]any{
		"id":     "rl_xxx",
		"object": "request_log",
		"account": map[string]any{
			"id":     "ac_xxx",
			"object": "account",
			"name":   "Acme Inc.",
		},
		"actor": map[string]any{
			"id":     "usr_xxx",
			"object": "user",
			"name":   "John Doe",
			"role": map[string]any{
				"id":     "rl_yyy",
				"object": "role",
				"name":   "Admin",
			},
		},
	}

	// Only actor.role requested — actor should be kept (child include), account should be collapsed.
	result := CollapseUnexpanded(data, requestLogConfig(), map[string]bool{
		"actor.role": true,
	})

	assert.Nil(t, result["account"], "account should be collapsed")

	actor := result["actor"].(map[string]any)
	assert.Equal(t, "usr_xxx", actor["id"], "actor should be kept because actor.role is requested")

	role := actor["role"].(map[string]any)
	assert.Equal(t, "Admin", role["name"], "actor.role should be expanded")
}

func TestCollapseUnexpanded_NilConfig(t *testing.T) {
	data := map[string]any{
		"id":   "apke_xxx",
		"role": map[string]any{"id": "rl_xxx", "name": "Admin"},
	}

	result := CollapseUnexpanded(data, nil, map[string]bool{})

	role := result["role"].(map[string]any)
	assert.Equal(t, "Admin", role["name"])
}
