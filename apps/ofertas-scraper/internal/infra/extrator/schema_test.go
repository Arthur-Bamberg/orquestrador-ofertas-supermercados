package extrator

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadGeminiResponseSchema(t *testing.T) {
	root := repoRoot(t)
	schema, schemaJSON, err := loadGeminiResponseSchema(
		filepath.Join(root, "schemas", "extracao.json"),
		filepath.Join(root, "schemas", "oferta.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if schema == nil {
		t.Fatal("schema nil")
	}
	if schemaJSON == "" {
		t.Fatal("schemaJSON empty")
	}
	m, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema type %T", schema)
	}
	defs, ok := m["$defs"].(map[string]any)
	if !ok || defs["oferta"] == nil {
		t.Fatalf("expected $defs.oferta, got %#v", m["$defs"])
	}
	oferta := defs["oferta"].(map[string]any)
	if _, has := oferta["exclusiveMinimum"]; has {
		t.Fatal("exclusiveMinimum should be sanitized away")
	}
	promo, _ := oferta["properties"].(map[string]any)["promocao"].(map[string]any)
	oneOf, _ := promo["oneOf"].([]any)
	if len(oneOf) != 4 {
		t.Fatalf("promocao oneOf len=%d", len(oneOf))
	}
	cartao := oneOf[2].(map[string]any)
	props := cartao["properties"].(map[string]any)
	cartaoField := props["promocaoCartao"].(map[string]any)
	if _, hasConst := cartaoField["const"]; hasConst {
		t.Fatal("const should become enum")
	}
	if cartaoField["enum"] == nil {
		t.Fatalf("expected enum for promocaoCartao: %#v", cartaoField)
	}
	clube := oneOf[3].(map[string]any)
	clubeProps := clube["properties"].(map[string]any)
	clubeField := clubeProps["promocaoClube"].(map[string]any)
	if clubeField["enum"] == nil {
		t.Fatalf("expected enum for promocaoClube: %#v", clubeField)
	}
	required, _ := oferta["required"].([]any)
	hasInicio, hasQuantidades := false, false
	for _, r := range required {
		if r == "dataInicio" {
			hasInicio = true
		}
		if r == "quantidades" {
			hasQuantidades = true
		}
	}
	if !hasInicio {
		t.Fatalf("dataInicio must be required: %#v", required)
	}
	if !hasQuantidades {
		t.Fatalf("quantidades must be required: %#v", required)
	}
	ofertaProps, _ := oferta["properties"].(map[string]any)
	qtd, _ := ofertaProps["quantidades"].(map[string]any)
	if qtd["type"] != "array" {
		t.Fatalf("quantidades type: %#v", qtd["type"])
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/infra/extrator → repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
