package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
)

type contextKey string

const (
	DataKey contextKey = "data"
)

type TemplateLib struct {
}
type Instruction struct {
	Kind     string         `yaml:"kind"`
	Version  string         `yaml:"version"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
	If       string         `yaml:"if,omitempty"`
}

func (t *TemplateLib) ParseTemplate(templateData []byte) ([]Instruction, error) {
	var tmpInstructions []Instruction
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
		tmpInstructions = append(tmpInstructions, instr)
	}

	return tmpInstructions, nil
}

func (t *TemplateLib) AdjustTemplate(ctx context.Context, item map[string]any, instructions []Instruction) ([]byte, error) {
	contextAny := ctx.Value(DataKey)

	dataMap, _ := contextAny.(map[string]any)
	if dataMap == nil {
		dataMap = map[string]any{}
	}

	templateSet := t.findApplicableTemplate(instructions, item, dataMap)

	combinedResult := t.combineTemplates(instructions, templateSet, item, dataMap)

	if len(combinedResult) == 0 {
		// TODO: Log warning about no templates applied
	}

	finalJSON, err := json.MarshalIndent(combinedResult, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling final result: %w", err)
	}

	return finalJSON, nil
}

func (t *TemplateLib) findApplicableTemplate(instructions []Instruction, item map[string]any, ctx map[string]any) map[string]struct{} {
	templateSet := make(map[string]struct{})

	for _, instr := range instructions {
		if strings.TrimSpace(instr.Kind) != "View" {
			continue
		}

		if instr.If != "" {
			match, err := t.evaluateCondition(instr.If, item, ctx)
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

func (t *TemplateLib) combineTemplates(instructions []Instruction, templateSet map[string]struct{}, item map[string]any, ctx map[string]any) map[string]any {
	combined := make(map[string]any)

	for _, instr := range instructions {
		if strings.TrimSpace(instr.Kind) != "Template" {
			continue
		}

		metaName, _ := instr.Metadata["name"].(string)
		if _, need := templateSet[metaName]; !need {
			continue
		}

		t.applyTemplateSpec(instr.Spec, combined, item, ctx)
	}

	return combined
}

func (t *TemplateLib) applyTemplateSpec(spec map[string]any, combined map[string]any, item, ctx map[string]any) {
	for key, value := range spec {
		if strings.TrimSpace(key) == "if" {
			continue
		}

		switch val := value.(type) {
		case map[string]any:
			t.applyMapValue(key, val, combined, item, ctx)
		default:
			combined[key] = val
		}
	}
}

func (t *TemplateLib) applyMapValue(key string, val map[string]any, combined map[string]any, item, ctx map[string]any) {
	typeRaw, hasType := val["type"]
	if !hasType {
		t.applyNestedObject(key, val, combined, item, ctx)
		return
	}

	typeStr, _ := typeRaw.(string)
	switch typeStr {
	case "string":
		if err := t.processStringValue(key, val, combined, item, ctx); err != nil {
			// TODO: log error
		}
	case "number":
		t.processNumberValue(key, val, combined, item, ctx)
	case "array":
		t.processArrayValue(key, val, combined, item, ctx)
	case "bool":
		if boolVal, ok := val["value"].(bool); ok {
			combined[key] = boolVal
		}
	case "object":
		t.applyNestedObject(key, val, combined, item, ctx)
	default:
		combined[key] = val
	}
}

func (t *TemplateLib) applyNestedObject(key string, val map[string]any, combined map[string]any, item, ctx map[string]any) {
	valueRaw, ok := val["value"].(map[string]any)
	if !ok {
		combined[key] = val
		return
	}

	subResult := make(map[string]any)
	for subKey, subVal := range valueRaw {
		switch typedVal := subVal.(type) {
		case map[string]any:
			t.applyMapValue(subKey, typedVal, subResult, item, ctx)
		default:
			subResult[subKey] = typedVal
		}
	}

	if existingObj, ok := combined[key].(map[string]any); ok {
		for srK, srV := range subResult {
			existingObj[srK] = srV
		}
		combined[key] = existingObj
	} else {
		combined[key] = subResult
	}
}

func (t *TemplateLib) processStringValue(key string, val map[string]any, result map[string]any, item map[string]any, ctx map[string]any) error {
	var errList []error

	if pathValue, exists := val["path"]; exists {
		pathStr, ok := pathValue.(string)
		if !ok {
			errList = append(errList, fmt.Errorf("path value is not a string for key: %s", key))
		} else {
			resolvedValue, err := gval.Evaluate(pathStr, map[string]interface{}{
				"item":    item,
				"context": ctx,
			})
			if err != nil {
				errList = append(errList, fmt.Errorf("error resolving path for key %s: %w", key, err))
				result[key] = nil
			} else {
				result[key] = resolvedValue
			}
		}
	} else if valValue, exists := val["value"]; exists {
		if tmpl, ok := valValue.(string); ok {
			interpolated, err := t.interpolateString(tmpl, item, ctx)
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

func (t *TemplateLib) processNumberValue(key string, val map[string]any, result map[string]any, item map[string]any, ctx map[string]any) {
	pathValue, exists := val["path"]
	if !exists {
		return
	}
	pathStr, ok := pathValue.(string)
	if !ok {
		// TODO: log error
		return
	}

	resolvedValue, err := gval.Evaluate(pathStr, map[string]interface{}{
		"item":    item,
		"context": ctx,
	})
	if err != nil {
		// TODO: log error
		result[key] = nil
	} else {
		result[key] = resolvedValue
	}
}

func (t *TemplateLib) processArrayValue(key string, val map[string]any, result map[string]any, item map[string]any, ctx map[string]any) {
	rawArr, exists := val["value"]
	if !exists {
		return
	}
	subArray, ok := rawArr.([]any)
	if !ok {
		return
	}

	processed := make([]any, 0, len(subArray))
	for _, subItem := range subArray {
		subMap, ok := subItem.(map[string]any)
		if !ok {
			continue
		}
		if condRaw, hasCond := subMap["if"]; hasCond {
			if condStr, ok := condRaw.(string); ok && condStr != "" {
				match, err := t.evaluateCondition(condStr, item, ctx)
				if err != nil || !match {
					continue
				}
			}
		}
		elemResult := make(map[string]any)
		for k2, v2 := range subMap {
			if strings.TrimSpace(k2) == "if" {
				continue
			}
			switch castV2 := v2.(type) {
			case map[string]any:
				t.applyMapValue(k2, castV2, elemResult, item, ctx)
			default:
				elemResult[k2] = castV2
			}
		}
		processed = append(processed, elemResult)
	}
	result[key] = processed
}

func (t *TemplateLib) interpolateString(templateStr string, item map[string]any, ctx map[string]any) (string, error) {
	tmpl, err := template.New("interpolation").Funcs(template.FuncMap{
		"item":    func() map[string]any { return item },
		"context": func() map[string]any { return ctx },
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

func (t *TemplateLib) evaluateCondition(condition string, item map[string]any, ctx map[string]any) (bool, error) {
	params := map[string]interface{}{
		"item":    item,
		"context": ctx,
	}

	if err := t.validateKeys(condition, params); err != nil {
		return false, err
	}

	exprLanguage := gval.Full()
	expr, err := exprLanguage.Evaluate(condition, params)
	if err != nil {
		return false, fmt.Errorf("error evaluating condition: %w", err)
	}

	result, ok := expr.(bool)
	if !ok {
		return false, fmt.Errorf("condition did not evaluate to a boolean: %v", expr)
	}
	return result, nil
}

func (t *TemplateLib) validateKeys(condition string, params map[string]interface{}) error {
	exprLanguage := gval.Full()
	_, err := exprLanguage.Evaluate(condition, params)
	if err != nil {
		return errors.New("undefined keys in condition: " + err.Error())
	}
	return nil
}
