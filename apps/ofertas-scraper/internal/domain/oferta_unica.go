package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// FingerprintPDF returns the SHA-256 hex digest of PDF bytes (ADR 0036).
func FingerprintPDF(pdf []byte) string {
	sum := sha256.Sum256(pdf)
	return hex.EncodeToString(sum[:])
}

// chaveUnicaPayload is the canonical Oferta uniqueness shape (ADR 0036).
type chaveUnicaPayload struct {
	ProdutoID     ProdutoID    `json:"produtoId"`
	MercadoID     MercadoID    `json:"mercadoId"`
	MarcaID       string       `json:"marcaId"`
	Valor         float64      `json:"valor"`
	Quantidades   []float64    `json:"quantidades"`
	Medida        Medida       `json:"medida"`
	DataInicio    string       `json:"dataInicio"`
	DataExpiracao string       `json:"dataExpiracao"`
	Promocao      *Promocao    `json:"promocao"`
	Comparativo   *Comparativo `json:"comparativo"`
}

// ChaveUnicaOferta is a stable hex digest of the Oferta catalog identity (ADR 0036).
func ChaveUnicaOferta(o Oferta) string {
	marca := ""
	if o.MarcaID != nil {
		marca = string(*o.MarcaID)
	}
	payload := chaveUnicaPayload{
		ProdutoID:     o.ProdutoID,
		MercadoID:     o.MercadoID,
		MarcaID:       marca,
		Valor:         o.Valor,
		Quantidades:   o.Quantidades,
		Medida:        o.Medida,
		DataInicio:    o.DataInicio,
		DataExpiracao: o.DataExpiracao,
		Promocao:      o.Promocao,
		Comparativo:   o.Comparativo,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		// Deterministic fallback — should not happen with domain types.
		return fmt.Sprintf("fallback:%s:%s:%g", o.ProdutoID, o.MercadoID, o.Valor)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
