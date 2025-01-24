package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"gopkg.in/yaml.v3"
	"io"
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
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error parsing YAML: %w", err)
		}
		if ifValue, exists := instr.Spec["if"]; exists {
			if ifCondition, ok := ifValue.(string); ok {
				instr.If = ifCondition
			}
		}
		instructions = append(instructions, instr)
	}

	return &TemplateInstructionsImpl{instructions: instructions}, nil
}

func findApplicableTemplate(t *TemplateInstructionsImpl, item map[string]any, context map[string]any) map[string]struct{} {
	templateSet := make(map[string]struct{})
	for _, instr := range t.instructions {
		if strings.TrimSpace(instr.Kind) != "View" {
			continue
		}
		if instr.If != "" {
			match, err := evaluateCondition(instr.If, item, context)
			if err != nil {
				// TODO: Log error while evaluating condition
				continue
			}
			if !match {
				continue
			}
		}
		if templateValue, exists := instr.Spec["template"]; exists {
			if specTemplates, ok := templateValue.(map[string]any); ok {
				if templatesValue, exists := specTemplates["templates"]; exists {
					if templates, ok := templatesValue.([]any); ok {
						for _, tmpl := range templates {
							if tmplName, ok := tmpl.(string); ok {
								templateSet[tmplName] = struct{}{}
							}
						}
					}
				}
			}
		}
	}
	return templateSet
}

func (t *TemplateInstructionsImpl) AdjustTemplate(item map[string]any, context map[string]any) ([]byte, error) {
	templateSet := findApplicableTemplate(t, item, context)
	result := combineTemplates(t, templateSet, item, context)

	if len(result) == 0 {
		// TODO: Log warning about no templates applied
	}

	return json.MarshalIndent(result, "", "  ")
}

func combineTemplates(t *TemplateInstructionsImpl, templateSet map[string]struct{}, item map[string]any, context map[string]any) map[string]any {
	combinedResult := make(map[string]any)

	for _, instr := range t.instructions {
		if strings.TrimSpace(instr.Kind) != "Template" {
			continue
		}
		if metadataValue, exists := instr.Metadata["name"]; exists {
			if metadataName, ok := metadataValue.(string); ok {
				if _, exists := templateSet[metadataName]; exists {
					processedTemplate := processTemplate(instr.Spec, item, context)

					for key, value := range processedTemplate {
						if existingValue, exists := combinedResult[key]; exists {
							switch existingValue.(type) {
							case map[string]any:
								if newMap, ok := value.(map[string]any); ok {
									combinedResult[key] = newMap
								} else {
									combinedResult[key] = value
								}
							default:
								combinedResult[key] = value
							}
						} else {
							combinedResult[key] = value
						}
					}
				}
			}
		}
	}
	return combinedResult
}

func processTemplate(spec map[string]any, item map[string]any, context map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range spec {
		if strings.TrimSpace(key) == "if" {
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
	if typeValue, exists := val["type"]; exists {
		if t, ok := typeValue.(string); ok {
			switch t {
			case "string":
				err := processStringValue(key, val, result, item, context)
				if err != nil {
					// TODO: Log warning about wrong path of json
				}
			case "object":
				result[key] = processTemplate(val, item, context)
			case "number":
				processNumberValue(key, val, result, item, context)
			case "array":
				processArrayValue(key, val, result, item, context)
			case "bool":
				if boolValue, exists := val["value"]; exists {
					if boolResult, ok := boolValue.(bool); ok {
						result[key] = boolResult
					}
				}
			default:
				result[key] = val
			}
		}
	} else {
		result[key] = processTemplate(val, item, context)
	}
}

func processStringValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) error {
	var errList []error

	if pathValue, exists := val["path"]; exists {
		if path, ok := pathValue.(string); ok {
			resolvedValue, err := gval.Evaluate(path, map[string]interface{}{
				"item":    item,
				"context": context,
			})
			if err != nil {
				errList = append(errList, fmt.Errorf("error resolving path for key %s: %w", key, err))
				result[key] = nil
			} else {
				result[key] = resolvedValue
			}
		} else {
			errList = append(errList, fmt.Errorf("path value is not a string for key: %s", key))
		}
	} else if tmplValue, exists := val["value"]; exists {
		if tmpl, ok := tmplValue.(string); ok {
			interpolated, err := interpolateString(tmpl, item, context)
			if err != nil {
				errList = append(errList, fmt.Errorf("error interpolating string for key %s: %w", key, err))
				result[key] = tmpl
			} else {
				result[key] = interpolated
			}
		} else {
			errList = append(errList, fmt.Errorf("value is not a string for key: %s", key))
		}
	}

	if len(errList) > 0 {
		return errors.Join(errList...)
	}

	return nil
}

func processNumberValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if pathValue, exists := val["path"]; exists {
		if path, ok := pathValue.(string); ok {
			resolvedValue, err := gval.Evaluate(path, map[string]interface{}{
				"item":    item,
				"context": context,
			})
			if err != nil {
				// TODO: Log error resolving path
				result[key] = nil
			} else {
				result[key] = resolvedValue
			}
		}
	}
}

func processArrayValue(key string, val map[string]any, result map[string]any, item map[string]any, context map[string]any) {
	if value, exists := val["value"]; exists {
		if subArray, ok := value.([]any); ok {
			processedArray := make([]any, 0, len(subArray))
			for _, subItem := range subArray {
				if subMap, ok := subItem.(map[string]any); ok {
					if condValue, exists := subMap["if"]; exists {
						if cond, ok := condValue.(string); ok && cond != "" {
							match, err := evaluateCondition(cond, item, context)
							if err != nil || !match {
								continue
							}
						}
					}
					processedArray = append(processedArray, processTemplate(subMap, item, context))
				}
			}
			result[key] = processedArray
		}
	}
}

func interpolateString(templateStr string, item map[string]any, context map[string]any) (string, error) {
	tmpl, err := template.New("interpolation").Funcs(template.FuncMap{
		"item":    func() map[string]any { return item },
		"context": func() map[string]any { return context },
	}).Parse(templateStr)
	if err != nil {
		return templateStr, fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return templateStr, fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func evaluateCondition(condition string, item map[string]any, context map[string]any) (bool, error) {
	params := map[string]interface{}{
		"item":    item,
		"context": context,
	}

	if err := validateKeys(condition, params); err != nil {
		return false, err
	}

	exprLanguage := gval.Full()
	expr, err := exprLanguage.Evaluate(condition, params)
	if err != nil {
		return false, fmt.Errorf("error evaluating condition: %w", err)
	}

	if result, ok := expr.(bool); ok {
		return result, nil
	}

	return false, fmt.Errorf("condition did not evaluate to boolean: %v", expr)
}

func validateKeys(condition string, params map[string]interface{}) error {
	exprLanguage := gval.Full()
	_, err := exprLanguage.Evaluate(condition, params)
	if err != nil {
		return errors.New("undefined keys in condition: " + err.Error())
	}
	return nil
}
