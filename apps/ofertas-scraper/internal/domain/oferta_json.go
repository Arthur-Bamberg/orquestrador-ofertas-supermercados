package domain

import "encoding/json"

// UnmarshalJSON accepts quantidades[] and legacy singular quantidade (ADR 0030).
func (o *Oferta) UnmarshalJSON(data []byte) error {
	var j struct {
		ID                  OfertaID    `json:"id"`
		DocumentoID         DocumentoID `json:"documentoId"`
		ProdutoID           ProdutoID   `json:"produtoId"`
		MarcaID             *MarcaID    `json:"marcaId,omitempty"`
		MercadoID           MercadoID   `json:"mercadoId"`
		Valor               float64     `json:"valor"`
		Quantidades         []float64   `json:"quantidades"`
		Quantidade          *float64    `json:"quantidade"`
		Medida              Medida      `json:"medida"`
		DataInicio          string      `json:"dataInicio"`
		DataExpiracao       string      `json:"dataExpiracao"`
		OrigemDataInicio    OrigemData  `json:"origemDataInicio"`
		OrigemDataExpiracao OrigemData  `json:"origemDataExpiracao"`
		Promocao            *Promocao   `json:"promocao,omitempty"`
	}
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	o.ID = j.ID
	o.DocumentoID = j.DocumentoID
	o.ProdutoID = j.ProdutoID
	o.MarcaID = j.MarcaID
	o.MercadoID = j.MercadoID
	o.Valor = j.Valor
	o.Quantidades = coalesceQuantidades(j.Quantidades, j.Quantidade)
	o.Medida = j.Medida
	o.DataInicio = j.DataInicio
	o.DataExpiracao = j.DataExpiracao
	o.OrigemDataInicio = j.OrigemDataInicio
	o.OrigemDataExpiracao = j.OrigemDataExpiracao
	o.Promocao = j.Promocao
	return nil
}
