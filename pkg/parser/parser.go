package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/url"
	"strconv"
	"strings"
	"text/template"
)

type Instruction struct {
	Kind     string         `yaml:"kind"`
	Version  string         `yaml:"version"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
	If       string         `yaml:"if,omitempty"`
}

type TemplateInstructionsImpl struct {
	instructions []Instruction
}

func ParseTemplate(templateData []byte) (*TemplateInstructionsImpl, error) {
	var instructions []Instruction
	decoder := yaml.NewDecoder(bytes.NewReader(templateData))

	for {
		var instr Instruction
		err := decoder.Decode(&instr)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error parsing YAML: %w", err)
		}
		if ifCondition, ok := instr.Spec["if"].(string); ok {
			instr.If = ifCondition
		}
		instructions = append(instructions, instr)
	}

	return &TemplateInstructionsImpl{instructions: instructions}, nil
}
func findApplicableTemplate(t *TemplateInstructionsImpl, item map[string]any, context map[string]any) map[string]struct{} {
	templateSet := make(map[string]struct{})
	for _, instr := range t.instructions {
		if instr.Kind != "View" {
			continue
		}
		if instr.If != "" {
			match, err := evaluateCondition(instr.If, item, context)
			if err != nil {
				fmt.Printf("Error evaluating condition: %v\n", err)
				continue
			}
			if !match {
				continue
			}
		}
		specTemplates, ok := instr.Spec["template"].(map[string]any)
		if !ok {
			continue
		}
		if templates, ok := specTemplates["templates"].([]any); ok {
			for _, tmpl := range templates {
				if tmplName, ok := tmpl.(string); ok {
					templateSet[tmplName] = struct{}{}
				}
			}
		}
	}
	return templateSet
}
func (t *TemplateInstructionsImpl) AdjustTemplate(item map[string]any, context map[string]any) ([]byte, error) {

	templateSet := findApplicableTemplate(t, item, context)

	result := applyTemplates(t, templateSet, item, context)

	if len(result) == 0 {
		fmt.Println("No templates applied. Check your View and Template instructions.")
	}

	return json.MarshalIndent(result, "", "  ")
}
func applyTemplates(t *TemplateInstructionsImpl, templateSet map[string]struct{}, item map[string]any, context map[string]any) map[string]any {
	result := make(map[string]any)
	for _, instr := range t.instructions {
		if instr.Kind != "Template" {
			continue
		}
		metadataName, ok := instr.Metadata["name"].(string)
		if !ok {
			continue
		}
		if _, exists := templateSet[metadataName]; !exists {
			continue
		}

		processedTemplate := processTemplate(instr.Spec, item, context)
		result[metadataName] = processedTemplate
	}
	return result
}
func processTemplate(spec map[string]any, item map[string]any, context map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range spec {
		if key == "if" {
			continue
		}
		processTemplateKey(key, value, result, item, context)
	}

	return result
}

func processTemplateKey(key string, value any, result map[string]any, item map[string]any, context map[string]any) {
	switch val := value.(type) {
	case map[string]any:
		processMapValue(key, val, result, item, context)
	default:
		result[key] = value
	}
}

func processMapValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if t, ok := val["type"].(string); ok {
		switch t {
		case "string":
			processStringValue(key, val, result, item, context)
		case "object":
			result[key] = processTemplate(val, item, context)
		case "number":
			processNumberValue(key, val, result, item, context)
		case "array":
			processArrayValue(key, val, result, item, context)
		case "bool":
			if boolValue, ok := val["value"].(bool); ok {
				result[key] = boolValue
			}
		default:
			result[key] = val
		}
	} else {
		result[key] = processTemplate(val, item, context)
	}
}

func processStringValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if path, ok := val["path"].(string); ok {
		resolvedValue := lookupJsonPath(path, item, context)
		if resolvedValue != nil {
			result[key] = resolvedValue
		} else {
			result[key] = nil
		}
	} else if tmpl, ok := val["value"].(string); ok {
		result[key] = interpolateString(tmpl, item, context)
	}
}

func processNumberValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if path, ok := val["path"].(string); ok {
		resolvedValue := lookupJsonPath(path, item, context)
		if resolvedValue != nil {
			result[key] = resolvedValue
		} else {
			result[key] = nil
		}
	}
}

func processArrayValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if subArray, ok := val["value"].([]any); ok {
		var processedArray []any
		for _, subItem := range subArray {
			if subMap, ok := subItem.(map[string]any); ok {
				if cond, hasIf := subMap["if"].(string); hasIf {
					match, err := evaluateCondition(cond, item, context)
					if err != nil || !match {
						continue
					}
				}
				processedArray = append(processedArray, processTemplate(subMap, item, context))
			}
		}
		result[key] = processedArray
	}
}

func interpolateString(templateStr string, item map[string]any, context map[string]any) string {

	tmpl, err := template.New("interpolation").Funcs(template.FuncMap{
		"item":    func() map[string]any { return item },
		"context": func() map[string]any { return context },
	}).Parse(templateStr)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		return templateStr
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		return templateStr
	}

	interpolatedStr := buf.String()

	decodedResult, err := url.QueryUnescape(interpolatedStr)
	if err != nil {
		fmt.Printf("Error decoding URL: %v\n", err)
		return interpolatedStr
	}

	return decodedResult
}

func lookupJsonPath(path string, item map[string]any, context map[string]any) any {
	pathParts := strings.Split(path, ".")

	var currentValue interface{}

	if strings.HasPrefix(path, "item.") {
		pathParts = pathParts[1:]
		currentValue = item
	} else if strings.HasPrefix(path, "context.") {
		pathParts = pathParts[1:]
		currentValue = context
	} else {
		currentValue = context
	}

	for _, part := range pathParts {
		if currentMap, isMap := currentValue.(map[string]interface{}); isMap {
			if nextValue, ok := currentMap[part]; ok {
				if subMap, isMap := nextValue.(map[string]interface{}); isMap {
					currentValue = subMap
				} else {
					return nextValue
				}
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return currentValue
}

func evaluateCondition(condition string, item map[string]any, context map[string]any) (bool, error) {
	var key, operator, value string

	_, err := fmt.Sscanf(condition, "%s %s %s", &key, &operator, &value)
	value = strings.Trim(value, "\"")
	key = strings.Trim(key, "\"")
	if err != nil {
		return false, fmt.Errorf("invalid condition format: %s", condition)
	}

	actualValue := lookupJsonPath(key, item, context)
	if actualValue == nil {
		actualValue = key
	}
	expectedValue := lookupJsonPath(value, item, context)
	if expectedValue == nil {
		expectedValue = value
	}

	actualInt, actualIsInt := tryConvertToInt(actualValue)
	expectedInt, expectedIsInt := tryConvertToInt(expectedValue)

	if actualIsInt && expectedIsInt {
		switch operator {
		case "==":
			return actualInt == expectedInt, nil
		case "!=":
			return actualInt != expectedInt, nil
		case ">":
			return actualInt > expectedInt, nil
		case "<":
			return actualInt < expectedInt, nil
		default:
			return false, fmt.Errorf("unsupported operator: %s", operator)
		}
	}

	switch operator {
	case "==":
		return fmt.Sprintf("%v", actualValue) == expectedValue, nil
	case "!=":
		return fmt.Sprintf("%v", actualValue) != expectedValue, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func tryConvertToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}
