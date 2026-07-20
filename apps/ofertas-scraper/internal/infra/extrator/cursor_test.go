package extrator

import (
	"errors"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestMapCursorBridgeError_Indisponivel(t *testing.T) {
	cases := []struct {
		msg  string
		code string
	}{
		{msg: "service unavailable", code: "unavailable"},
		{msg: "context deadline exceeded", code: ""},
	}
	for _, tc := range cases {
		got := mapCursorBridgeError(tc.msg, tc.code)
		if !errors.Is(got, domain.ErrExtratorIndisponivel) {
			t.Fatalf("msg=%q code=%q → %v, want ErrExtratorIndisponivel", tc.msg, tc.code, got)
		}
	}
}

func TestMapCursorBridgeError_Cota(t *testing.T) {
	cases := []struct {
		msg  string
		code string
	}{
		{msg: "too many requests", code: "rate_limit"},
		{msg: "HTTP 429", code: ""},
	}
	for _, tc := range cases {
		got := mapCursorBridgeError(tc.msg, tc.code)
		if !errors.Is(got, domain.ErrExtratorCota) {
			t.Fatalf("msg=%q code=%q → %v, want ErrExtratorCota", tc.msg, tc.code, got)
		}
	}
}

func TestMapCursorBridgeError_Passthrough(t *testing.T) {
	got := mapCursorBridgeError("bad schema", "decode")
	if errors.Is(got, domain.ErrExtratorIndisponivel) {
		t.Fatalf("decode should not be indisponivel: %v", got)
	}
}

func TestMaterializeCursorBridge(t *testing.T) {
	path, err := materializeCursorBridge()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty path")
	}
}
