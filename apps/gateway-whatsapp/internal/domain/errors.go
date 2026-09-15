package domain

import "errors"

// ErrProvedorDuplicado is returned by SalvarMensagem when another row already
// holds the same non-empty provedor_id (concurrent Receber race).
var ErrProvedorDuplicado = errors.New("provedor_id duplicado")

// ErrSemPareamento is Desparear when the Canal is already pendente.
var ErrSemPareamento = errors.New("canal sem pareamento")

// ErrDesparearIndisponivel is Desparear on the stub Canal (local WHATSAPP_STUB).
var ErrDesparearIndisponivel = errors.New("desparear não se aplica")
