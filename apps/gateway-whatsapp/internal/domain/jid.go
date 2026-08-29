package domain

import (
	"strings"
	"unicode"
)

type JID string

type Allowlist struct {
	jids map[JID]struct{}
}

func NovaAllowlist(csv string) Allowlist {
	a := Allowlist{jids: map[JID]struct{}{}}
	for _, part := range strings.Split(csv, ",") {
		jid := NormalizarJID(part)
		if jid == "" {
			continue
		}
		a.jids[jid] = struct{}{}
	}
	return a
}

func (a Allowlist) PermiteConversa(raw string) bool {
	jid := NormalizarJID(raw)
	if jid == "" {
		return false
	}
	_, ok := a.jids[jid]
	return ok
}

func NormalizarJID(raw string) JID {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "@") {
		return JID(s)
	}
	var digits strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return ""
	}
	return JID(digits.String() + "@s.whatsapp.net")
}
