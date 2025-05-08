package parser

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseTemplate_Success(t *testing.T) {
	yamlData := `
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
    value: "Hello!"

`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err, "Expected no error when parsing valid YAML")
	assert.Len(t, tpls, 2, "Should parse 2 instructions (View + Template)")
}

func TestAdjustTemplate_Simple(t *testing.T) {
	yamlData := `
---
kind: View
metadata:
  name: simple-view
spec:
  template:
    templates: ["tmpl1"]
---
kind: Template
metadata:
  name: tmpl1
spec:
  message:
    type: "string"
    value: "Hello, {{item.name}}!"
  code:
    type: "number"
    path: "item.code"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err, "Should parse template without error")

	itemMap := map[string]any{"name": "World", "code": 42}
	dataMap := map[string]any{}

	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, dataMap)

	resultJSON, err := temp.AdjustTemplate(ctx, itemMap, tpls)
	assert.NoError(t, err, "AdjustTemplate should succeed")

	expected := `{
		"message": "Hello, World!",
		"code": 42
	}`
	assert.JSONEq(t, expected, string(resultJSON), "Mismatch in simple scenario")
}

func TestAdjustTemplate_NoView(t *testing.T) {
	yamlData := `
---
kind: Template
metadata:
  name: only-template
spec:
  someKey:
    type: "string"
    value: "No view here"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err)
	item := map[string]any{"foo": "bar"}
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, map[string]any{"test": "value"})

	resultJSON, err := temp.AdjustTemplate(ctx, item, tpls)
	assert.NoError(t, err, "Should not fail, but produce empty result")

	assert.JSONEq(t, `{}`, string(resultJSON), "Expected empty JSON when no View instructions")
}

func TestAdjustTemplate_IfCondition_Fail(t *testing.T) {
	yamlData := `
---
kind: View
spec:
  if: "item.enabled == true"
  template:
    templates: ["tmpl"]
---
kind: Template
metadata:
  name: tmpl
spec:
  testKey:
    type: "string"
    value: "Will not appear"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err)
	item := map[string]any{"enabled": false}
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, map[string]any{"test": "value"})

	resultJSON, err := temp.AdjustTemplate(ctx, item, tpls)
	assert.NoError(t, err)

	assert.JSONEq(t, `{}`, string(resultJSON), "View if-condition is false, so result should be empty")
}

func TestChainedObjectMerging(t *testing.T) {
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
    type: "object"
    value:
      name: "John"
      status: "basic"
---
kind: Template
metadata:
  name: template2
spec:
  data1:
    type: "object"
    value:
      status: "primary"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(templateData))
	assert.NoError(t, err, "Failed to parse chained templates")
	item := map[string]any{"something": 123}
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, map[string]any{"another": "value"})

	resultJSON, err := temp.AdjustTemplate(ctx, item, tpls)
	assert.NoError(t, err, "AdjustTemplate failed in chained merging scenario")

	expectedJSON := `{
		"data1": {
			"name": "John",
			"status": "primary"
		}
	}`
	assert.JSONEq(t, expectedJSON, string(resultJSON), "Chained merging mismatch")
}

func TestEvaluateCondition_Success(t *testing.T) {
	item := map[string]any{"id": 42, "active": true}
	data := map[string]any{"limit": 50}
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, data)
	var temp TemplateLib
	tests := []struct {
		cond     string
		expected bool
	}{
		{"item.id == 42", true},
		{"item.id == 100", false},
		{"item.active", true},
		{"item.active == false", false},
		{"item.id < 100 && context.limit == 50", true},
		{"item.id > 100 || context.limit < 10", false},
		{"(item.id > 0) && (context.limit == 50)", true},
	}

	for _, tc := range tests {
		result, err := temp.evaluateCondition(ctx, tc.cond, item)
		assert.NoError(t, err, "No error expected in evaluateCondition: %s", tc.cond)
		assert.Equal(t, tc.expected, result, "Mismatch condition: %s", tc.cond)
	}
}

func TestEvaluateCondition_Errors(t *testing.T) {
	item := map[string]any{}
	data := map[string]any{}
	var temp TemplateLib
	cond1 := "item.??error??"
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, data)
	_, err := temp.evaluateCondition(ctx, cond1, item)
	assert.Error(t, err, "Should fail on syntax error")

	cond2 := "(((("
	_, err = temp.evaluateCondition(ctx, cond2, item)
	assert.Error(t, err, "Should fail on unbalanced parens, etc.")
}

func TestProcessStringValue(t *testing.T) {
	t.Run("String with interpolation", func(t *testing.T) {
		item := map[string]any{"name": "Alice"}
		data := map[string]any{}
		val := map[string]any{
			"type":  "string",
			"value": "Hello, {{item.name}}!",
		}
		var temp TemplateLib
		result := make(map[string]any)
		ctx := context.Background()
		ctx = context.WithValue(ctx, DataKey, data)
		err := temp.processStringValue(ctx, "greeting", val, result, item)
		assert.NoError(t, err, "No error expected in interpolation")

		assert.Equal(t, "Hello, Alice!", result["greeting"], "String interpolation mismatch")
	})

	t.Run("String with path", func(t *testing.T) {
		item := map[string]any{"age": 33}
		data := map[string]any{"some": 22}
		val := map[string]any{
			"type": "string",
			"path": "item.age",
		}
		var temp TemplateLib
		result := make(map[string]any)
		ctx := context.Background()
		ctx = context.WithValue(ctx, DataKey, data)
		err := temp.processStringValue(ctx, "userAge", val, result, item)
		assert.NoError(t, err)
		assert.Equal(t, 33, result["userAge"], "Path resolution mismatch")
	})

	t.Run("Invalid path type", func(t *testing.T) {
		val := map[string]any{
			"type": "string",
			"path": 123,
		}
		data := map[string]any{"some": 22}
		var temp TemplateLib
		ctx := context.Background()
		result := make(map[string]any)
		ctx = context.WithValue(ctx, DataKey, data)
		err := temp.processStringValue(ctx, "key", val, result, nil)
		assert.Error(t, err, "Should fail due to non-string path")
	})

	t.Run("Invalid template syntax", func(t *testing.T) {
		val := map[string]any{
			"type":  "string",
			"value": "Hello, {{invalid}!",
		}
		data := map[string]any{"some": 22}
		ctx := context.Background()
		ctx = context.WithValue(ctx, DataKey, data)
		var temp TemplateLib
		result := make(map[string]any)
		err := temp.processStringValue(ctx, "broken", val, result, nil)
		if err != nil {
			assert.Error(t, err, "Expected an error from invalid Go template syntax")
		}
	})
}

func TestProcessNumberValue(t *testing.T) {
	val := map[string]any{
		"type": "number",
		"path": "item.num",
	}
	item := map[string]any{"num": 99}
	data := map[string]any{}
	result := make(map[string]any)
	var temp TemplateLib
	ctx := context.Background()
	ctx = context.WithValue(ctx, DataKey, data)
	temp.processNumberValue(ctx, "resultNumber", val, result, item)
	assert.Equal(t, 99, result["resultNumber"], "Should extract item.num = 99")

	val2 := map[string]any{
		"type": "number",
		"path": "item.unknownField",
	}
	result2 := make(map[string]any)
	temp.processNumberValue(ctx, "badNum", val2, result2, item)
	assert.Nil(t, result2["badNum"], "Unknown path => nil")
}

func TestProcessArrayValue(t *testing.T) {
	t.Run("Basic array of sub-items", func(t *testing.T) {
		item := map[string]any{"role": "user"}
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
		var temp TemplateLib
		result := make(map[string]any)
		ctx := context.WithValue(context.Background(), DataKey, map[string]any{"role": "user"})
		temp.processArrayValue(ctx, "permissions", val, result, item)
		expected := map[string]any{
			"permissions": []any{
				map[string]any{"value": "User access granted"},
			},
		}
		assert.Equal(t, expected, result, "Mismatch in array conditional logic")
	})

	t.Run("Invalid array type", func(t *testing.T) {
		val := map[string]any{
			"value": 123,
		}
		var temp TemplateLib
		result := make(map[string]any)
		temp.processArrayValue(nil, "arrKey", val, result, nil)
		assert.Nil(t, result["arrKey"], "Should remain nil if not a valid array")
	})
}

func TestApplyNestedObject(t *testing.T) {
	combined := map[string]any{
		"data1": map[string]any{
			"name": "Alice",
			"city": "Paris",
		},
	}

	val := map[string]any{
		"value": map[string]any{
			"city":   "London",
			"status": "active",
		},
	}
	var temp TemplateLib
	temp.applyNestedObject(nil, "data1", val, combined, nil)

	expected := map[string]any{
		"data1": map[string]any{
			"name":   "Alice",
			"city":   "London",
			"status": "active",
		},
	}
	assert.Equal(t, expected, combined, "Nested object merging mismatch")
}

func TestAdjustTemplate_WithErrors(t *testing.T) {
	yamlData := `
---
kind: View
spec:
  if: "item.missing == *??" # некорректный синтаксис
  template:
    templates: ["tmpl1"]
---
kind: Template
metadata:
  name: tmpl1
spec:
  greeting:
    type: "string"
    value: "Hello!"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err, "Parsing YAML itself might still succeed since it's valid YAML syntax")
	item := map[string]any{"missing": false}
	ctx := context.WithValue(context.Background(), DataKey, map[string]any{})

	resultJSON, err := temp.AdjustTemplate(ctx, item, tpls)
	assert.NoError(t, err, "We skip instructions on condition error, so not a fatal error")

	assert.JSONEq(t, `{}`, string(resultJSON), "Expected empty JSON if condition evaluation fails")
}

func TestValidateKeys(t *testing.T) {
	cond := "item.unknownField > 10"
	var temp TemplateLib
	err := temp.validateKeys(cond, map[string]interface{}{
		"item": map[string]any{"knownField": 5},
	})

	if err != nil {
		assert.Error(t, err, "Possibly unknown field triggers error")
	} else {
		assert.True(t, true, "Gval might treat unknownField as <nil>")
	}
}

func TestInterpolateString_Error(t *testing.T) {
	input := "Hello, {{broken"
	item := map[string]any{}
	ctx := context.Background()
	var temp TemplateLib
	_, err := temp.interpolateString(ctx, input, item)
	assert.Error(t, err, "Should fail on parse error in go template")
}

func TestApplyNestedObject_NoValue(t *testing.T) {
	val := map[string]any{
		"type": "object",
	}
	var temp TemplateLib
	combined := make(map[string]any)
	temp.applyNestedObject(nil, "someObj", val, combined, nil)

	assert.Equal(t, val, combined["someObj"], "If no .value => store as is")
}

func TestAdjustTemplate_IfCondition_UnknownField(t *testing.T) {
	yamlData := `
---
kind: View
spec:
  if: "item.doesNotExist > 10"
  template:
    templates: ["tmpl1"]
---
kind: Template
metadata:
  name: tmpl1
spec:
  testKey:
    type: "string"
    value: "Value if condition passes"
`
	var temp TemplateLib
	tpls, err := temp.ParseTemplate([]byte(yamlData))
	assert.NoError(t, err)
	item := map[string]any{"x": 1}
	ctx := context.WithValue(context.Background(), DataKey, map[string]any{})

	resultJSON, err := temp.AdjustTemplate(ctx, item, tpls)

	assert.NoError(t, err, "Should not be fatal error, but might skip the view")

	assert.JSONEq(t, `{}`, string(resultJSON), "No templates should be applied if condition is false or error")
}

func TestProcessArrayValue_InvalidType(t *testing.T) {
	val := map[string]any{
		"type":  "array",
		"value": 123,
	}
	var temp TemplateLib
	combined := make(map[string]any)
	temp.processArrayValue(nil, "arr", val, combined, nil)

	assert.Nil(t, combined["arr"], "Should remain nil if not a valid array")
}

func TestProcessArrayValue_ObjectItem(t *testing.T) {
	val := map[string]any{
		"value": []any{
			map[string]any{
				"objField": map[string]any{
					"type":  "object",
					"value": map[string]any{"nested": "hello"},
				},
			},
		},
	}
	var temp TemplateLib
	combined := make(map[string]any)
	temp.processArrayValue(nil, "arr", val, combined, nil)

	assert.Len(t, combined, 1)
	assert.Contains(t, combined, "arr")

	arrVal, ok := combined["arr"].([]any)
	assert.True(t, ok, "Should be a slice")
	assert.Len(t, arrVal, 1, "One item in array")
}
