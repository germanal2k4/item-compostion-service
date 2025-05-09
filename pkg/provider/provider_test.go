package provider

import (
	"os"

	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGRPCProviderParser_Parse(t *testing.T) {
	validSpec := `
version: v1
kind: ProviderGRPC
metadata:
  name: reaction
  labels:
    foo: bar
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: 1s
    logging:
      enabled: true
  payload:
    headers:
      x-api-key: ${PLANET_REACTION_API_KEY}
      x-app-name: my-application
  methods:
    - package: reaction.internal
      service: ReactionInternalService
      method: GetReactionCountersByDomainId
      type: DomainBatch
      timeout: 1s
      filter:
        if: 'item.createdAt > time.Now - 7 * time.Day'
      request:
        domain: item.domain
        domain_ids: item.id
      response:
        itemId: items.domain_id
`
	os.Setenv("PLANET_REACTION_API_KEY", "test-api-key")
	defer os.Unsetenv("PLANET_REACTION_API_KEY")

	parser := NewGRPCProviderParser()
	spec, err := parser.Parse([]byte(validSpec))
	if err != nil {
		t.Fatalf("Failed to parse valid spec: %v", err)
	}

	if spec.Version != "v1" {
		t.Errorf("Expected version v1, got %s", spec.Version)
	}
	if spec.Kind != "ProviderGRPC" {
		t.Errorf("Expected kind ProviderGRPC, got %s", spec.Kind)
	}
	if spec.Metadata.Name != "reaction" {
		t.Errorf("Expected name reaction, got %s", spec.Metadata.Name)
	}

	if spec.Spec.Transport.Address != "reaction.tbank.ru:443" {
		t.Errorf("Expected address reaction.tbank.ru:443, got %s", spec.Spec.Transport.Address)
	}
	if spec.Spec.Transport.Timeout != time.Second {
		t.Errorf("Expected timeout 1s, got %v", spec.Spec.Transport.Timeout)
	}
	if !spec.Spec.Transport.Logging.Enabled {
		t.Error("Expected logging to be enabled")
	}

	if len(spec.Spec.Payload.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(spec.Spec.Payload.Headers))
	}

	apiKey, exists := spec.Spec.Payload.Headers["x-api-key"]
	if !exists {
		t.Error("Expected x-api-key header to exist")
	}
	if !strings.HasPrefix(apiKey, "${") || !strings.HasSuffix(apiKey, "}") {
		t.Errorf("Expected x-api-key to be an environment variable reference, got %s", apiKey)
	}

	appName, exists := spec.Spec.Payload.Headers["x-app-name"]
	if !exists {
		t.Error("Expected x-app-name header to exist")
	}
	if appName == "" {
		t.Error("Expected x-app-name to have a value")
	}

	if len(spec.Spec.Methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(spec.Spec.Methods))
	}

	method := spec.Spec.Methods[0]
	if method.Package != "reaction.internal" {
		t.Errorf("Expected package reaction.internal, got %s", method.Package)
	}
	if method.Service != "ReactionInternalService" {
		t.Errorf("Expected service ReactionInternalService, got %s", method.Service)
	}
	if method.Method != "GetReactionCountersByDomainId" {
		t.Errorf("Expected method GetReactionCountersByDomainId, got %s", method.Method)
	}
	if method.Type != "DomainBatch" {
		t.Errorf("Expected type DomainBatch, got %s", method.Type)
	}
	if method.Timeout != time.Second {
		t.Errorf("Expected timeout 1s, got %v", method.Timeout)
	}
	if method.Filter.If != "item.createdAt > time.Now - 7 * time.Day" {
		t.Errorf("Expected filter condition, got %s", method.Filter.If)
	}
	if method.Request.Domain != "item.domain" {
		t.Errorf("Expected request domain item.domain, got %s", method.Request.Domain)
	}
	if method.Request.DomainIDs != "item.id" {
		t.Errorf("Expected request domain_ids item.id, got %s", method.Request.DomainIDs)
	}
	if method.Response.ItemID != "items.domain_id" {
		t.Errorf("Expected response itemId items.domain_id, got %s", method.Response.ItemID)
	}
}

func TestGRPCProviderParser_Validate(t *testing.T) {
	os.Setenv("PLANET_REACTION_API_KEY", "test-api-key")
	defer os.Unsetenv("PLANET_REACTION_API_KEY")

	validSpec := `
version: v1
kind: ProviderGRPC
metadata:
  name: reaction
  labels:
    foo: bar
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: 1s
    logging:
      enabled: true
  payload:
    headers:
      x-api-key: ${PLANET_REACTION_API_KEY}
      x-app-name: my-application
  methods:
    - package: reaction.internal
      service: ReactionInternalService
      method: GetReactionCountersByDomainId
      type: DomainBatch
      timeout: 1s
      filter:
        if: 'item.createdAt > time.Now - 7 * time.Day'
      request:
        domain: item.domain
        domain_ids: item.id
      response:
        itemId: items.domain_id
`

	parser := NewGRPCProviderParser()
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name:    "valid spec",
			spec:    validSpec,
			wantErr: false,
		},
		{
			name: "missing version",
			spec: `
kind: ProviderGRPC
metadata:
  name: reaction
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: 1s
`,
			wantErr: true,
		},
		{
			name: "invalid kind",
			spec: `
version: v1
kind: InvalidKind
metadata:
  name: reaction
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: 1s
`,
			wantErr: true,
		},
		{
			name: "missing name",
			spec: `
version: v1
kind: ProviderGRPC
metadata: {}
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: 1s
`,
			wantErr: true,
		},
		{
			name: "missing transport address",
			spec: `
version: v1
kind: ProviderGRPC
metadata:
  name: reaction
spec:
  transport:
    timeout: 1s
`,
			wantErr: true,
		},
		{
			name: "invalid timeout",
			spec: `
version: v1
kind: ProviderGRPC
metadata:
  name: reaction
spec:
  transport:
    address: reaction.tbank.ru:443
    timeout: -1s
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parser.Parse([]byte(tt.spec))
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, spec)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, spec)
			}
		})
	}
}

func TestGRPCProviderParser_FromFile(t *testing.T) {
	os.Setenv("PLANET_REACTION_API_KEY", "test-api-key")
	defer os.Unsetenv("PLANET_REACTION_API_KEY")

	parser := NewGRPCProviderParser()

	data, err := os.ReadFile("../../config/providers/reaction.yaml")
	if err != nil {
		t.Fatalf("Failed to read provider spec file: %v", err)
	}

	spec, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse provider spec: %v", err)
	}

	assert.Equal(t, "v1", spec.Version)
	assert.Equal(t, "ProviderGRPC", spec.Kind)
	assert.Equal(t, "reaction", spec.Metadata.Name)
	assert.Equal(t, "reaction.tbank.ru:443", spec.Spec.Transport.Address)
	assert.Equal(t, time.Second, spec.Spec.Transport.Timeout)
	assert.True(t, spec.Spec.Transport.Logging.Enabled)

	assert.Equal(t, 2, len(spec.Spec.Payload.Headers))
	assert.True(t, strings.HasPrefix(spec.Spec.Payload.Headers["x-api-key"], "${"))
	assert.True(t, strings.HasSuffix(spec.Spec.Payload.Headers["x-api-key"], "}"))
	assert.Equal(t, "my-application", spec.Spec.Payload.Headers["x-app-name"])

	assert.Equal(t, 1, len(spec.Spec.Methods))
	method := spec.Spec.Methods[0]
	assert.Equal(t, "reaction.internal", method.Package)
	assert.Equal(t, "ReactionInternalService", method.Service)
	assert.Equal(t, "GetReactionCountersByDomainId", method.Method)
	assert.Equal(t, "DomainBatch", string(method.Type))
	assert.Equal(t, time.Second, method.Timeout)
	assert.Equal(t, "item.createdAt > time.Now - 7 * time.Day", method.Filter.If)
	assert.Equal(t, "item.domain", method.Request.Domain)
	assert.Equal(t, "item.id", method.Request.DomainIDs)
	assert.Equal(t, "items.domain_id", method.Response.ItemID)
}
