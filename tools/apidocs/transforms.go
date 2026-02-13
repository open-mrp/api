package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/ohler55/ojg/jp"
)

type Transform struct {
	Command string        `json:"command"`
	Reason  string        `json:"reason"`
	Args    TransformArgs `json:"args"`
}

type TransformArgs struct {
	Target   any      `json:"target"` // string or []string
	Value    any      `json:"value"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Keys     []string `json:"keys"`
	Template bool     `json:"template"`
}

func applyTransforms(data any, transforms []Transform) any {
	for _, t := range transforms {
		log.Printf("Applying transform: %s (%s)", t.Command, t.Reason)
		var err error
		switch t.Command {
		case "update":
			err = transformUpdate(data, t.Args)
		case "append":
			err = transformAppend(data, t.Args)
		case "move":
			err = transformMove(data, t.Args)
		case "merge":
			err = transformMerge(data, t.Args)
		case "remove":
			err = transformRemove(data, t.Args)
		case "copy":
			err = transformCopy(data, t.Args)
		default:
			log.Printf("Warning: unknown transform command: %s", t.Command)
		}
		if err != nil {
			log.Printf("Error applying transform %s: %v", t.Command, err)
		}
	}
	return data
}

func getTargets(target any) []string {
	switch v := target.(type) {
	case string:
		return []string{v}
	case []any:
		var targets []string
		for _, t := range v {
			if s, ok := t.(string); ok {
				targets = append(targets, s)
			}
		}
		return targets
	case []string:
		return v
	default:
		return nil
	}
}

func transformUpdate(data any, args TransformArgs) error {
	targets := getTargets(args.Target)
	for _, target := range targets {
		expr, err := jp.ParseString(target)
		if err != nil {
			return err
		}
		if args.Template {
			templateStr, ok := args.Value.(string)
			if !ok {
				return fmt.Errorf("template requires string value")
			}
			if _, err := expr.Modify(data, func(v any) (any, bool) {
				if s, ok := v.(string); ok {
					return strings.ReplaceAll(templateStr, "{{value}}", s), true
				}
				return v, false
			}); err != nil {
				return err
			}
		} else {
			if _, err := expr.Modify(data, func(v any) (any, bool) {
				return args.Value, true
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func transformAppend(data any, args TransformArgs) error {
	targets := getTargets(args.Target)
	for _, target := range targets {
		expr, err := jp.ParseString(target)
		if err != nil {
			return err
		}

		if _, err := expr.Modify(data, func(v any) (any, bool) {
			switch targetVal := v.(type) {
			case map[string]any:
				newVals, ok := args.Value.(map[string]any)
				if !ok {
					return v, false
				}
				changed := false
				for k, val := range newVals {
					if _, exists := targetVal[k]; exists {
						log.Printf("Warning: append failed, property %s already exists", k)
						continue
					}
					targetVal[k] = val
					changed = true
				}
				return targetVal, changed
			case []any:
				// Check if value already exists in slice
				exists := false
				for _, item := range targetVal {
					if reflect.DeepEqual(item, args.Value) {
						exists = true
						break
					}
				}
				if !exists {
					return append(targetVal, args.Value), true
				}
				log.Printf("Warning: append failed, value already exists in slice")
				return v, false
			default:
				return v, false
			}
		}); err != nil {
			return err
		}
	}
	return nil
}

func transformMove(data any, args TransformArgs) error {
	if args.From == "" || args.To == "" {
		return fmt.Errorf("move requires from and to")
	}

	fromExpr, err := jp.ParseString(args.From)
	if err != nil {
		return err
	}

	// We need to find the parent and the key/index for 'from'
	// and the parent and the key/index for 'to'.
	// This is hard with general JSONPath.
	// For simple paths like $.a.b.c -> $.a.b.d it's easier.

	// Stainless 'move' seems to be used for renaming properties or moving nodes.
	// Example: "from": "$.components.schemas.user_response", "to": "$.components.schemas.UserResponse"

	val := fromExpr.Get(data)
	if len(val) == 0 {
		return nil
	}

	// Remove from old location
	if err := fromExpr.Del(data); err != nil {
		return err
	}

	// Set at new location
	toExpr, err := jp.ParseString(args.To)
	if err != nil {
		return err
	}
	return toExpr.Set(data, val[0])
}

func transformMerge(data any, args TransformArgs) error {
	targets := getTargets(args.Target)
	for _, target := range targets {
		expr, err := jp.ParseString(target)
		if err != nil {
			return err
		}

		if _, err := expr.Modify(data, func(v any) (any, bool) {
			targetMap, ok := v.(map[string]any)
			if !ok {
				return v, false
			}
			newVals, ok := args.Value.(map[string]any)
			if !ok {
				return v, false
			}
			for k, val := range newVals {
				targetMap[k] = val
			}
			return targetMap, true
		}); err != nil {
			return err
		}
	}
	return nil
}

func transformRemove(data any, args TransformArgs) error {
	targets := getTargets(args.Target)
	for _, target := range targets {
		expr, err := jp.ParseString(target)
		if err != nil {
			return err
		}

		if len(args.Keys) > 0 {
			// Remove specific keys from the target(s)
			if _, err := expr.Modify(data, func(v any) (any, bool) {
				changed := false
				if m, ok := v.(map[string]any); ok {
					for _, k := range args.Keys {
						if _, exists := m[k]; exists {
							delete(m, k)
							changed = true
						}
					}
				}
				return v, changed
			}); err != nil {
				return err
			}
		} else {
			// Remove the entire target(s)
			if err := expr.Del(data); err != nil {
				return err
			}
		}
	}
	return nil
}

func transformCopy(data any, args TransformArgs) error {
	if args.From == "" || args.To == "" {
		return fmt.Errorf("copy requires from and to")
	}

	fromExpr, err := jp.ParseString(args.From)
	if err != nil {
		return err
	}

	val := fromExpr.Get(data)
	if len(val) == 0 {
		return nil
	}

	toExpr, err := jp.ParseString(args.To)
	if err != nil {
		return err
	}
	return toExpr.Set(data, val[0])
}
