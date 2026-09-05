package domain

import (
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
