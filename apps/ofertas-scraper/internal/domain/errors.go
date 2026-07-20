package domain

import "errors"

// ErrExtratorIndisponivel signals the Extrator adapter could not reach the backend
// (timeouts, 5xx, etc.). RunDailyJob retries with job-scoped backoff (ADR 0022).
// Rate-limit / quota uses ErrExtratorCota instead (ADR 0037).
var ErrExtratorIndisponivel = errors.New("extrator indisponivel")

// ErrExtratorCota signals rate-limit / quota exhaustion for an Extrator adapter (ADR 0037).
// Composite Extrator failovers; the job must not apply ADR 0022 backoff for this error.
var ErrExtratorCota = errors.New("extrator cota esgotada")

// Extrator adapter names persisted on Uso do Extrator (ADR 0037).
const (
	ExtratorProviderGemini = "gemini"
	ExtratorProviderCursor = "cursor"
	ExtratorProviderStub   = "stub"
)