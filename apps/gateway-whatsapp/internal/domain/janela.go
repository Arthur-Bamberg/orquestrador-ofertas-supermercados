package domain

import "time"

const DuracaoJanela = 24 * time.Hour

func DentroDaJanela(ultimaEntrada, agora time.Time) bool {
	if ultimaEntrada.IsZero() {
		return false
	}
	if agora.Before(ultimaEntrada) {
		return true
	}
	return agora.Sub(ultimaEntrada) <= DuracaoJanela
}
