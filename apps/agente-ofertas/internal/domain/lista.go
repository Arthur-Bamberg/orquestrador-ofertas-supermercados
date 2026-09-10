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
	"de": {}, "da": {}, "do": {}, "das": {}, "dos": {},
	"um": {}, "uma": {}, "uns": {}, "umas": {},
	"o": {}, "a": {}, "os": {}, "as": {},
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
	var keep []string
	for _, tok := range strings.Fields(b.String()) {
		if _, skip := involucro[tok]; skip {
			continue
		}
		if medidaOuNumero.MatchString(tok) {
			continue
		}
		keep = append(keep, tok)
	}
	return strings.Join(keep, " ")
}
