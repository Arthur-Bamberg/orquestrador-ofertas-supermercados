package extrator

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"

	"google.golang.org/genai"
)

func TestMapGeminiError_Indisponivel(t *testing.T) {
	cases := []error{
		context.DeadlineExceeded,
		genai.APIError{Code: 429, Message: "rate", Status: "RESOURCE_EXHAUSTED"},
		genai.APIError{Code: 503, Message: "down", Status: "UNAVAILABLE"},
		fmt.Errorf("boom: %w", genai.APIError{Code: 500, Message: "err", Status: "INTERNAL"}),
	}
	for _, err := range cases {
		got := mapGeminiError(err)
		if !errors.Is(got, domain.ErrExtratorIndisponivel) {
			t.Fatalf("err %v → %v, want ErrExtratorIndisponivel", err, got)
		}
	}
}

func TestMapGeminiError_ClientErrorPassthrough(t *testing.T) {
	err := genai.APIError{Code: 400, Message: "bad schema", Status: "INVALID_ARGUMENT"}
	got := mapGeminiError(err)
	if errors.Is(got, domain.ErrExtratorIndisponivel) {
		t.Fatalf("400 should not be indisponivel: %v", got)
	}
}

func TestIsCacheMiss(t *testing.T) {
	if !isCacheMiss(genai.APIError{Code: 404, Message: "cachedContents/x not found"}) {
		t.Fatal("expected 404 cache miss")
	}
	if isCacheMiss(genai.APIError{Code: 400, Message: "bad"}) {
		t.Fatal("400 is not cache miss")
	}
}
