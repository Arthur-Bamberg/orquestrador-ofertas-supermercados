package domain
import (
	"regexp"
	"strings"
	"unicode"
)

type Item struct {
	Texto string
}

type Lista struct {
	Itens []Item
}

func ParseLista(texto string) Lista {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return Lista{}
	}
	separado := strings.NewReplacer("\n", ",", ";", ",", " e ", ",", " E ", ",").Replace(texto)
	var itens []Item
	for _, part := range strings.Split(separado, ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimFunc(part, func(r rune) bool { return unicode.IsSpace(r) || r == '-' })
		if part == "" {
			continue
		}
		itens = append(itens, Item{Texto: part})
	}
	return Lista{Itens: itens}
}

var medidaOuNumero = regexp.MustCompile(`^[0-9]+([.,][0-9]+)?(kg|g|ml|un|l)?$`)

var involucro = map[string]struct{}{
	"comprar": {}, "quero": {}, "preciso": {}, "gostaria": {},
	"lista": {}, "item": {}, "itens": {},
	"oi": {}, "ola": {}, "olá": {},
	"bom": {}, "dia": {}, "boa": {}, "tarde": {}, "noite": {},
	"por": {}, "favor": {}, "obrigado": {}, "obrigada": {}, "pfv": {}, "pf": {},
	"um": {}, "uma": {}, "uns": {}, "umas": {},
	"o": {}, "a": {}, "os": {}, "as": {},
}

var preposicao = map[string]struct{}{
	"de": {}, "da": {}, "do": {}, "das": {}, "dos": {},
}

var medidaToken = map[string]struct{}{
	"kg": {}, "g": {}, "ml": {}, "un": {}, "unidade": {}, "unidades": {},
	"litro": {}, "litros": {}, "kilo": {}, "kilos": {}, "grama": {}, "gramas": {},
	"l": {},
}

func TermoDoItem(texto string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(texto)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	toks := strings.Fields(b.String())
	var keep []string
	temTipo := false
	for i, tok := range toks {
		if _, prep := preposicao[tok]; prep {
			if tipoNoItem(toks, i-1) && tipoNoItem(toks, i+1) {
				keep = append(keep, tok)
			}
			continue
		}
		if _, skip := involucro[tok]; skip {
			continue
		}
		if soTamanho(tok) {
			keep = append(keep, tok)
			continue
		}
		temTipo = true
		keep = append(keep, tok)
	}
	if !temTipo {
		return ""
	}
	return strings.Join(keep, " ")
}

func TipoDoTermo(termo string) string {
	var keep []string
	for _, tok := range strings.Fields(strings.ToLower(strings.TrimSpace(termo))) {
		if soTamanho(tok) {
			continue
		}
		keep = append(keep, tok)
	}
	return strings.Join(keep, " ")
}

func soTamanho(tok string) bool {
	if _, ok := medidaToken[tok]; ok {
		return true
	}
	return medidaOuNumero.MatchString(tok)
}

func tipoNoItem(toks []string, i int) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	t := toks[i]
	if _, skip := involucro[t]; skip {
		return false
	}
	if _, prep := preposicao[t]; prep {
		return false
	}
	return !soTamanho(t)
}

func TokensSemPreposicao(s string) []string {
	var out []string
	for _, t := range strings.Fields(s) {
		if _, skip := preposicao[t]; skip {
			continue
		}
		out = append(out, t)
	}
	return out
}
