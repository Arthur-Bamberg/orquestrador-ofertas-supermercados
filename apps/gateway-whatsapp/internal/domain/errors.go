package domain

import "errors"

// ErrProvedorDuplicado is returned by SalvarMensagem when another row already
// holds the same non-empty provedor_id (concurrent Receber race).
var ErrProvedorDuplicado = errors.New("provedor_id duplicado")

// ErrForaDaJanela is a free-form send when the Conversa has no open Janela.
var ErrForaDaJanela = errors.New("conversa fora da janela de 24h")

// ErrCanalNaoConfigurado is a send when Cloud API credentials are missing.
var ErrCanalNaoConfigurado = errors.New("canal não configurado")
