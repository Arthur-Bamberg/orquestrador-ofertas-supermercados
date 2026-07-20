package extrator

import (
	"encoding/json"
	"fmt"
	"os"
)

// loadGeminiResponseSchema builds a ResponseJsonSchema from the Extrator contract files,
// inlining oferta.json under $defs so Gemini can resolve $ref without external files.
func loadGeminiResponseSchema(extracaoPath, ofertaPath string) (schema any, schemaJSON string, err error) {
	extracaoRaw, err := os.ReadFile(extracaoPath)
	if err != nil {
		return nil, "", fmt.Errorf("read extracao schema: %w", err)
	}
	ofertaRaw, err := os.ReadFile(ofertaPath)
	if err != nil {
		return nil, "", fmt.Errorf("read oferta schema: %w", err)
	}

	var extracao map[string]any
	if err := json.Unmarshal(extracaoRaw, &extracao); err != nil {
		return nil, "", fmt.Errorf("parse extracao schema: %w", err)
	}
	var oferta map[string]any
	if err := json.Unmarshal(ofertaRaw, &oferta); err != nil {
		return nil, "", fmt.Errorf("parse oferta schema: %w", err)
	}

	oferta = sanitizeSchemaMap(oferta)
	delete(oferta, "$schema")
	delete(oferta, "$id")

	extracao = sanitizeSchemaMap(extracao)
	delete(extracao, "$schema")
	delete(extracao, "$id")

	props, _ := extracao["properties"].(map[string]any)
	if props == nil {
		return nil, "", fmt.Errorf("extracao schema missing properties")
	}
	ofertas, _ := props["ofertas"].(map[string]any)
	if ofertas == nil {
		return nil, "", fmt.Errorf("extracao schema missing properties.ofertas")
	}
	ofertas["items"] = map[string]any{"$ref": "#/$defs/oferta"}
	extracao["$defs"] = map[string]any{"oferta": oferta}

	out, err := json.Marshal(extracao)
	if err != nil {
		return nil, "", err
	}
	var asAny any
	if err := json.Unmarshal(out, &asAny); err != nil {
		return nil, "", err
	}
	return asAny, string(out), nil
}

func sanitizeSchemaMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "exclusiveMinimum":
			// Gemini ResponseJsonSchema lists minimum/maximum, not exclusiveMinimum.
			if _, hasMin := m["minimum"]; !hasMin {
				if n, ok := asFloat(v); ok {
					out["minimum"] = n
				}
			}
			continue
		case "minLength", "exclusiveMaximum":
			continue
		case "const":
			out["enum"] = []any{v}
			continue
		}
		out[k] = sanitizeSchemaValue(v)
	}
	return out
}

func sanitizeSchemaValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return sanitizeSchemaMap(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = sanitizeSchemaValue(item)
		}
		return out
	default:
		return v
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
