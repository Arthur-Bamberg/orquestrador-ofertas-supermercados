package domain

import "errors"

// ErrExtratorIndisponivel signals the Extrator adapter could not reach the backend
// (timeouts, 5xx, etc.). RunDailyJob retries with job-scoped backoff (ADR 0022).
var ErrExtratorIndisponivel = errors.New("extrator indisponivel")
