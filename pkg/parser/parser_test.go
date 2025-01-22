package parser

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestAdjustTemplate(t *testing.T) {
	templateData := `
---
kind: View
metadata:
  name: default-view
spec:
  template:
    templates: ["default-template"]
---
kind: Template
metadata:
  name: default-template
spec:
  greeting:
    type: "string"
    value: "Hello, {{item.name}}!"
  age:
    type: "number"
    path: "item.age"
`

	item := map[string]any{
		"name": "John",
		"age":  30,
	}

	context := map[string]any{
		"city": "New York",
	}

	templates, err := ParseTemplate([]byte(templateData))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	result, err := templates.AdjustTemplate(item, context)
	if err != nil {
		t.Fatalf("Failed to adjust template: %v", err)
	}

	var expected map[string]any
	expectedJSON := `{
			"greeting": "Hello, John!",
			"age": 30
	}`

	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("Failed to unmarshal expected result: %v", err)
	}

	var actual map[string]any
	if err := json.Unmarshal(result, &actual); err != nil {
		t.Fatalf("Failed to unmarshal actual result: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Result mismatch. Expected: %v, Got: %v", expected, actual)
	}
}

func TestCombineTemplates(t *testing.T) {
	templateData := `
---
kind: View
metadata:
  name: view1
spec:
  template:
    templates: ["template1", "template2"]
---
kind: Template
metadata:
  name: template1
spec:
  data1:
    type: "string"
    value: "Value from template1"
---
kind: Template
metadata:
  name: template2
spec:
  data2:
    type: "string"
    value: "Value from template2"
`

	templates, err := ParseTemplate([]byte(templateData))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	templateSet := map[string]struct{}{
		"template1": {},
		"template2": {},
	}

	item := map[string]any{}
	context := map[string]any{}

	result := combineTemplates(templates, templateSet, item, context)

	expected := map[string]any{
		"data1": "Value from template1",
		"data2": "Value from template2",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("CombineTemplates result mismatch. Expected: %v, Got: %v", expected, result)
	}
}

func TestEvaluateCondition(t *testing.T) {
	item := map[string]any{
		"id":     100,
		"name":   "John",
		"active": true,
	}

	context := map[string]any{
		"userId": 100,
	}

	tests := []struct {
		condition string
		expected  bool
		shouldErr bool
	}{
		{"item.unknown == 1", false, false},
		{"item.id == 100", true, false},
		{"item.name == \"John\"", true, false},
		{"item.active", true, false},
		{"item.id == 200 || (item.id < 200 && item.id > 50)", true, false},
		{"item.id > 50", true, false},
		{"item.id < 200 && item.id > 50", true, false},
		{"item.name == \"Doe\"", false, false},
	}

	for _, test := range tests {
		result, err := evaluateCondition(test.condition, item, context)
		if test.shouldErr {
			if err == nil {
				t.Errorf("Expected error for condition: %s, got nil", test.condition)
			}
			continue
		}
		if err != nil {
			t.Errorf("Unexpected error for condition: %s, error: %v", test.condition, err)
			continue
		}
		if result != test.expected {
			t.Errorf("Condition: %s, expected: %v, got: %v", test.condition, test.expected, result)
		}
	}
}

func TestProcessArrayValue(t *testing.T) {
	item := map[string]any{
		"role": "user",
		"age":  25,
	}

	context := map[string]any{
		"access": "read",
	}

	val := map[string]any{
		"value": []any{
			map[string]any{
				"if": "item.role == \"user\"",
				"value": map[string]any{
					"type":  "string",
					"value": "User access granted",
				},
			},
			map[string]any{
				"if": "item.role == \"admin\"",
				"value": map[string]any{
					"type":  "string",
					"value": "Admin access granted",
				},
			},
		},
	}

	result := make(map[string]any)
	processArrayValue("permissions", val, result, item, context)

	expected := map[string]any{
		"permissions": []any{
			map[string]any{
				"value": "User access granted",
			},
		},
	}

	if fmt.Sprintf("%v", result) != fmt.Sprintf("%v", expected) {
		t.Errorf("Test failed. Expected: %v, Got: %v", expected, result)
	}
}

func TestInterpolateString(t *testing.T) {
	item := map[string]any{
		"name": "John",
		"age":  30,
	}

	context := map[string]any{
		"city": "New York",
	}

	templateStr := "Hello, {{item.name}}! Welcome to {{context.city}}."
	expected := "Hello, John! Welcome to New York."

	result, err := interpolateString(templateStr, item, context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("Test failed. Expected: %v, Got: %v", expected, result)
	}
}

func TestProcessMapValue(t *testing.T) {
	item := map[string]any{"id": 123, "name": "Test Item"}
	context := map[string]any{"user": map[string]any{"id": 456, "name": "Context User"}}

	tests := []struct {
		name     string
		key      string
		val      map[string]any
		expected any
	}{
		{
			name: "String type processing",
			key:  "name",
			val: map[string]any{
				"type":  "string",
				"path":  "item.name",
				"value": "Default Name",
			},
			expected: "Test Item",
		},
		{
			name: "Number type processing",
			key:  "id",
			val: map[string]any{
				"type": "number",
				"path": "item.id",
			},
			expected: 123,
		},
		{
			name: "Bool type processing",
			key:  "isActive",
			val: map[string]any{
				"type":  "bool",
				"value": true,
			},
			expected: true,
		},
		{
			name: "Object type processing",
			key:  "user",
			val: map[string]any{
				"type": "object",
				"value": map[string]any{
					"id": map[string]any{
						"type": "number",
						"path": "context.user.id",
					},
				},
			},
			expected: map[string]any{
				"type": "object",
				"value": map[string]any{
					"id": 456,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			processMapValue(tt.key, tt.val, result, item, context)

			assert.Equal(t, tt.expected, result[tt.key])
		})
	}
}

func TestProcessStringValue(t *testing.T) {
	item := map[string]any{"id": 123, "name": "Test Item"}
	context := map[string]any{"user": map[string]any{"id": 456, "name": "Context User"}}

	tests := []struct {
		name     string
		key      string
		val      map[string]any
		expected any
	}{
		{
			name: "String interpolation",
			key:  "welcomeMessage",
			val: map[string]any{
				"type":  "string",
				"value": "Hello, {{item.name}}!",
			},
			expected: "Hello, Test Item!",
		},
		{
			name: "Path resolution",
			key:  "userName",
			val: map[string]any{
				"type": "string",
				"path": "context.user.name",
			},
			expected: "Context User",
		},
		{
			name: "Invalid interpolation",
			key:  "invalidMessage",
			val: map[string]any{
				"type":  "string",
				"value": "Hello, {{invalid.path}}!",
			},
			expected: "Hello, {{invalid.path}}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			processStringValue(tt.key, tt.val, result, item, context)

			assert.Equal(t, tt.expected, result[tt.key])
		})
	}
}
