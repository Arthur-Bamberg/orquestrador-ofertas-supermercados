package domain

import "errors"

// ErrProvedorDuplicado is returned by SalvarMensagem when another row already
// holds the same non-empty provedor_id (concurrent Receber race).
var ErrProvedorDuplicado = errors.New("provedor_id duplicado")
