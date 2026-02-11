package apiexample

import (
	"encoding/json"
)

func ValidateAndMarshalToMap(example any) map[string]any {
	b, err := json.Marshal(example)
	if err != nil {
		return make(map[string]any)
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return make(map[string]any)
	}
	return result
}
