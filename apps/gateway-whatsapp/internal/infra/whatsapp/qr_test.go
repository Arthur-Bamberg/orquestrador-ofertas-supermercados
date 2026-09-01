package whatsapp_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestImprimirQR_desenhaBlocosNoTerminal(t *testing.T) {
	var buf bytes.Buffer
	if err := whatsapp.ImprimirQR(&buf, "2@example-pairing-code"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "\n") < 8 {
		t.Fatalf("QR precisa de várias linhas para o celular ler, got %d linhas:\n%s", strings.Count(out, "\n"), out)
	}
	if !temBlocoQR(out) {
		t.Fatalf("esperava arte de QR (blocos), não a string crua:\n%s", out)
	}
}

func temBlocoQR(s string) bool {
	return strings.ContainsAny(s, "█▀▄■▓░")
}
