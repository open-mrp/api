package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyTransforms(t *testing.T) {
	specJSON := `{
		"openapi": "3.0.3",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"get": {
					"summary": "Get users",
					"parameters": [
						{
							"name": "account_id",
							"in": "query",
							"schema": {
								"type": "number"
							}
						}
					]
				}
			}
		},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"age": {
							"type": "string"
						}
					},
					"required": ["name"]
				}
			}
		}
	}`

	var data any
	err := json.Unmarshal([]byte(specJSON), &data)
	assert.NoError(t, err)

	transforms := []Transform{
		{
			Command: "update",
			Reason:  "Account IDs are strings, not numbers",
			Args: TransformArgs{
				Target: "$..parameters[?(@.name == 'account_id')].schema.type",
				Value:  "string",
			},
		},
		{
			Command: "update",
			Reason:  "User age should be integer",
			Args: TransformArgs{
				Target: "$.components.schemas.User.properties.age.type",
				Value:  "integer",
			},
		},
		{
			Command: "append",
			Reason:  "Mark id as required",
			Args: TransformArgs{
				Target: "$.components.schemas.User.required",
				Value:  "id",
			},
		},
		{
			Command: "append",
			Reason:  "Add id property",
			Args: TransformArgs{
				Target: "$.components.schemas.User.properties",
				Value: map[string]any{
					"id": map[string]any{
						"type":   "string",
						"format": "uuid",
					},
				},
			},
		},
		{
			Command: "update",
			Reason:  "Add DEPRECATED to summary",
			Args: TransformArgs{
				Target:   "$.paths['/users'].get.summary",
				Value:    "{{value}} (DEPRECATED)",
				Template: true,
			},
		},
		{
			Command: "move",
			Reason:  "Rename User to UserProfile",
			Args: TransformArgs{
				From: "$.components.schemas.User",
				To:   "$.components.schemas.UserProfile",
			},
		},
		{
			Command: "copy",
			Reason:  "Copy UserProfile to Backup",
			Args: TransformArgs{
				From: "$.components.schemas.UserProfile",
				To:   "$.components.schemas.Backup",
			},
		},
	}

	result := applyTransforms(data, transforms)

	b, err := json.Marshal(result)
	assert.NoError(t, err)
	var finalMap map[string]any
	json.Unmarshal(b, &finalMap)

	// Check account_id type
	paths := finalMap["paths"].(map[string]any)
	users := paths["/users"].(map[string]any)
	get := users["get"].(map[string]any)
	params := get["parameters"].([]any)
	accId := params[0].(map[string]any)
	assert.Equal(t, "string", accId["schema"].(map[string]any)["type"])
	assert.Equal(t, "Get users (DEPRECATED)", get["summary"])

	// Check UserProfile
	components := finalMap["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	assert.Nil(t, schemas["User"])
	assert.NotNil(t, schemas["UserProfile"])
	assert.NotNil(t, schemas["Backup"])

	userProfile := schemas["UserProfile"].(map[string]any)
	properties := userProfile["properties"].(map[string]any)
	assert.Equal(t, "integer", properties["age"].(map[string]any)["type"])
	assert.NotNil(t, properties["id"])

	required := userProfile["required"].([]any)
	assert.Contains(t, required, "id")
	assert.Contains(t, required, "name")
}

func TestApplyTransformsMerge(t *testing.T) {
	specJSON := `{
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"name": { "type": "string" }
					}
				}
			}
		}
	}`

	var data any
	json.Unmarshal([]byte(specJSON), &data)

	transforms := []Transform{
		{
			Command: "merge",
			Reason:  "Add metadata",
			Args: TransformArgs{
				Target: "$.components.schemas.User.properties.name",
				Value: map[string]any{
					"x-stainless-naming": map[string]any{
						"python": "user_name",
					},
				},
			},
		},
	}

	result := applyTransforms(data, transforms)

	b, _ := json.Marshal(result)
	var finalMap map[string]any
	json.Unmarshal(b, &finalMap)

	schemas := finalMap["components"].(map[string]any)["schemas"].(map[string]any)
	user := schemas["User"].(map[string]any)
	name := user["properties"].(map[string]any)["name"].(map[string]any)

	assert.Equal(t, "string", name["type"])
	assert.NotNil(t, name["x-stainless-naming"])
}

func TestApplyTransformsRemove(t *testing.T) {
	specJSON := `{
		"paths": {
			"/users": { "get": {} }
		},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"name": { "type": "string" },
						"age": { "type": "integer" }
					}
				}
			}
		}
	}`

	var data any
	json.Unmarshal([]byte(specJSON), &data)

	transforms := []Transform{
		{
			Command: "remove",
			Reason:  "Remove age property",
			Args: TransformArgs{
				Target: "$.components.schemas.User.properties",
				Keys:   []string{"age"},
			},
		},
		{
			Command: "remove",
			Reason:  "Remove users path",
			Args: TransformArgs{
				Target: "$.paths['/users']",
			},
		},
	}

	result := applyTransforms(data, transforms)

	b, _ := json.Marshal(result)
	var finalMap map[string]any
	json.Unmarshal(b, &finalMap)

	assert.Nil(t, finalMap["paths"].(map[string]any)["/users"])
	user := finalMap["components"].(map[string]any)["schemas"].(map[string]any)["User"].(map[string]any)
	properties := user["properties"].(map[string]any)
	assert.NotNil(t, properties["name"])
	assert.Nil(t, properties["age"])
}
