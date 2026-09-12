package operador

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

const CookieName = "operador"

var (
	ErrCredencial = errors.New("nome ou senha inválidos")
	ErrNome       = errors.New("nome obrigatório")
)

type Operador struct {
	ID   string
	Nome string
}

type Identificacao struct {
	ID       string
	Operador Operador
}

type Consulta interface {
	OperadorPorIdentificacao(ctx context.Context, identificacaoID string) (Operador, bool, error)
}

type Store interface {
	Consulta
	Criar(ctx context.Context, nome, senha string) (Operador, error)
	Identificar(ctx context.Context, nome, senha string) (Identificacao, error)
	Sair(ctx context.Context, identificacaoID string) error
	Apagar(ctx context.Context, operadorID string) error
	DefinirSenha(ctx context.Context, operadorID, senha string) error
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func normalizaNome(nome string) string {
	return strings.TrimSpace(nome)
}

func IDFromRequest(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil || c == nil {
		return ""
	}
	return c.Value
}

func SetCookie(w http.ResponseWriter, identificacaoID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    identificacaoID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 400,
	})
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
