package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/provider"
	"strings"
	"sync"
	"text/template"
	"time"
)

type contextKey string

const (
	DataKey contextKey = "data"
)

type TemplateLib struct {
	mu        sync.RWMutex
	providers map[string]provider.Provider
	metrics   *metricsCollector
}

func NewTemplateLib(metricsRegistry metrics.MetricsRegistry) (*TemplateLib, error) {
	metrics, err := newMetricsCollector(metricsRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics collector: %w", err)
	}

	return &TemplateLib{
		providers: make(map[string]provider.Provider),
		metrics:   metrics,
	}, nil
}

func (t *TemplateLib) RegisterProvider(p provider.Provider) {
	t.mu.Lock()
	t.providers[p.GetName()] = p
	t.mu.Unlock()
}

type Instruction struct {
	Kind     string         `yaml:"kind"`
	Version  string         `yaml:"version"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
	If       string         `yaml:"if,omitempty"`
}

func (t *TemplateLib) ParseTemplate(templateData []byte) ([]Instruction, error) {
	startTime := time.Now()
	t.metrics.parseRequestCount.WithLabelValues().Inc()

	var tmpInstructions []Instruction
	decoder := yaml.NewDecoder(bytes.NewReader(templateData))

	for {
		var instr Instruction
		err := decoder.Decode(&instr)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.metrics.errorsCount.WithLabelValues("parse_error").Inc()
			return nil, fmt.Errorf("error parsing YAML: %w", err)
		}

		if instr.Kind == "ProviderGRPC" {
			parser := provider.NewGRPCProviderParser()
			spec, err := parser.Parse(templateData)
			if err != nil {
				t.metrics.errorsCount.WithLabelValues("provider_parse_error").Inc()
				return nil, fmt.Errorf("error parsing provider spec: %w", err)
			}

			grpcProvider, err := provider.NewGRPCProvider(spec)
			if err != nil {
				t.metrics.errorsCount.WithLabelValues("provider_create_error").Inc()
				return nil, fmt.Errorf("error creating provider: %w", err)
			}

			t.RegisterProvider(grpcProvider)
			continue
		}

		if ifValue, exists := instr.Spec["if"]; exists {
			if ifCondition, ok := ifValue.(string); ok {
				instr.If = ifCondition
			}
		}

		tmpInstructions = append(tmpInstructions, instr)
	}

	t.metrics.parseTime.WithLabelValues().Observe(time.Since(startTime).Seconds())
	return tmpInstructions, nil
}

func (t *TemplateLib) AdjustTemplate(ctx context.Context, item map[string]any, instructions []Instruction) ([]byte, error) {
	startTime := time.Now()
	t.metrics.adjustRequestCount.WithLabelValues().Inc()

	templateSet := t.findApplicableTemplate(ctx, instructions, item)

	combinedResult := t.combineTemplates(ctx, instructions, templateSet, item)

	if len(combinedResult) == 0 {
		logger.FromContext(ctx).With("component", "template_lib").Warn("No combined result")
	}

	finalJSON, err := json.MarshalIndent(combinedResult, "", "  ")
	if err != nil {
		t.metrics.errorsCount.WithLabelValues("adjust_error").Inc()
		return nil, fmt.Errorf("error marshaling final result: %w", err)
	}

	t.metrics.adjustTime.WithLabelValues().Observe(time.Since(startTime).Seconds())
	return finalJSON, nil
}

func (t *TemplateLib) findApplicableTemplate(ctx context.Context, instructions []Instruction, item map[string]any) map[string]struct{} {
	templateSet := make(map[string]struct{})

	for _, instr := range instructions {
		if strings.TrimSpace(instr.Kind) != "View" {
			continue
		}

		if instr.If != "" {
			match, err := t.evaluateCondition(ctx, instr.If, item)
			if err != nil {
				logger.FromContext(ctx).With("component", "template_lib").Warn("Failed to evaluate condition", zap.Error(err))
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

func (t *TemplateLib) combineTemplates(ctx context.Context, instructions []Instruction, templateSet map[string]struct{}, item map[string]any) map[string]any {
	combined := make(map[string]any)

	for _, instr := range instructions {
		if strings.TrimSpace(instr.Kind) != "Template" {
			continue
		}

		metaName, _ := instr.Metadata["name"].(string)
		if _, need := templateSet[metaName]; !need {
			continue
		}

		t.applyTemplateSpec(ctx, instr.Spec, combined, item)
	}

	return combined
}

func (t *TemplateLib) applyTemplateSpec(ctx context.Context, spec map[string]any, combined map[string]any, item map[string]any) {
	for key, value := range spec {
		if strings.TrimSpace(key) == "if" {
			continue
		}

		switch val := value.(type) {
		case map[string]any:
			t.applyMapValue(ctx, key, val, combined, item)
		default:
			combined[key] = val
		}
	}
}

func (t *TemplateLib) applyMapValue(ctx context.Context, key string, val map[string]any, combined map[string]any, item map[string]any) {
	typeRaw, hasType := val["type"]
	if !hasType {
		t.applyNestedObject(ctx, key, val, combined, item)
		return
	}

	typeStr, _ := typeRaw.(string)
	switch typeStr {
	case "string":
		if err := t.processStringValue(ctx, key, val, combined, item); err != nil {
			logger.FromContext(ctx).With("component", "template_lib").Warn("Failed to process string", zap.Error(err))
		}
	case "number":
		t.processNumberValue(ctx, key, val, combined, item)
	case "array":
		t.processArrayValue(ctx, key, val, combined, item)
	case "bool":
		if boolVal, ok := val["value"].(bool); ok {
			combined[key] = boolVal
		}
	case "object":
		t.applyNestedObject(ctx, key, val, combined, item)
	default:
		combined[key] = val
	}
}

func (t *TemplateLib) applyNestedObject(ctx context.Context, key string, val map[string]any, combined map[string]any, item map[string]any) {
	valueRaw, ok := val["value"].(map[string]any)
	if !ok {
		combined[key] = val
		return
	}

	subResult := make(map[string]any)
	for subKey, subVal := range valueRaw {
		switch typedVal := subVal.(type) {
		case map[string]any:
			t.applyMapValue(ctx, subKey, typedVal, subResult, item)
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

func (t *TemplateLib) processStringValue(ctx context.Context, key string, val map[string]any, result map[string]any, item map[string]any) error {
	var errList []error
	contextAny := ctx.Value(DataKey).(map[string]any)
	if pathValue, exists := val["path"]; exists {
		pathStr, ok := pathValue.(string)
		if !ok {
			errList = append(errList, fmt.Errorf("path value is not a string for key: %s", key))
		} else {
			resolvedValue, err := gval.Evaluate(pathStr, map[string]interface{}{
				"item":    item,
				"context": contextAny,
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
			interpolated, err := t.interpolateString(ctx, tmpl, item)
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

func (t *TemplateLib) processNumberValue(ctx context.Context, key string, val map[string]any, result map[string]any, item map[string]any) {
	pathValue, exists := val["path"]
	if !exists {
		return
	}
	pathStr, ok := pathValue.(string)
	if !ok {
		logger.FromContext(ctx).With("component", "template_lib").Warn("Path value is not a string for key: %s", key)
		return
	}
	contextAny := ctx.Value(DataKey).(map[string]any)
	resolvedValue, err := gval.Evaluate(pathStr, map[string]interface{}{
		"item":    item,
		"context": contextAny,
	})
	if err != nil {
		logger.FromContext(ctx).With("component", "template_lib").Warn("Error resolving path for key %s: %v", key, err)
		result[key] = nil
	} else {
		result[key] = resolvedValue
	}
}

func (t *TemplateLib) processArrayValue(ctx context.Context, key string, val map[string]any, result map[string]any, item map[string]any) {
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
				match, err := t.evaluateCondition(ctx, condStr, item)
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
				t.applyMapValue(ctx, k2, castV2, elemResult, item)
			default:
				elemResult[k2] = castV2
			}
		}
		processed = append(processed, elemResult)
	}
	result[key] = processed
}

func (t *TemplateLib) handleProviderCall(providerName, methodName string, data map[string]interface{}) (interface{}, error) {
	t.mu.RLock()
	p, exists := t.providers[providerName]
	t.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	return p.ExecuteMethod(context.Background(), methodName, data)
}

func (t *TemplateLib) interpolateString(ctx context.Context, templateStr string, item map[string]any) (string, error) {
	contextAny, _ := ctx.Value(DataKey).(map[string]any)
	tmpl, err := template.New("interpolation").Funcs(template.FuncMap{
		"item":    func() map[string]any { return item },
		"context": func() map[string]any { return contextAny },
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

func (t *TemplateLib) evaluateCondition(ctx context.Context, condition string, item map[string]any) (bool, error) {
	contextAny := ctx.Value(DataKey).(map[string]any)
	params := map[string]interface{}{
		"item":    item,
		"context": contextAny,
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
