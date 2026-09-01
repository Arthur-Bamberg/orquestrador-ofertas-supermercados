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
		a.add(jid)
	}
	return a
}

func (a Allowlist) add(jid JID) {
	a.jids[jid] = struct{}{}
	for _, v := range variantesBR(jid) {
		a.jids[v] = struct{}{}
	}
}

func (a Allowlist) PermiteConversa(raws ...string) bool {
	for _, raw := range raws {
		jid := NormalizarJID(raw)
		if jid == "" {
			continue
		}
		if _, ok := a.jids[jid]; ok {
			return true
		}
		for _, v := range variantesBR(jid) {
			if _, ok := a.jids[v]; ok {
				return true
			}
		}
	}
	return false
}

func NormalizarJID(raw string) JID {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "@") {
		user, server, ok := strings.Cut(s, "@")
		if !ok || server == "" {
			return ""
		}
		if i := strings.IndexByte(user, '.'); i >= 0 {
			user = user[:i]
		}
		if i := strings.IndexByte(user, ':'); i >= 0 {
			user = user[:i]
		}
		if user == "" {
			return JID(server)
		}
		return JID(user + "@" + server)
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

func variantesBR(jid JID) []JID {
	user, server, ok := strings.Cut(string(jid), "@")
	if !ok || server != "s.whatsapp.net" || !strings.HasPrefix(user, "55") {
		return nil
	}
	for _, r := range user {
		if r < '0' || r > '9' {
			return nil
		}
	}
	switch len(user) {
	case 12:
		return []JID{JID(user[:4] + "9" + user[4:] + "@s.whatsapp.net")}
	case 13:
		if user[4] == '9' {
			return []JID{JID(user[:4] + user[5:] + "@s.whatsapp.net")}
		}
	}
	return nil
}

func Identidade(raw, pn, lid string) (JID, JID) {
	rawN := NormalizarJID(raw)
	pnN := NormalizarJID(pn)
	lidN := NormalizarJID(lid)
	if pnN == "" && strings.HasSuffix(string(rawN), "@s.whatsapp.net") {
		pnN = rawN
	}
	if lidN == "" && strings.HasSuffix(string(rawN), "@lid") {
		lidN = rawN
	}
	jid := pnN
	if jid == "" {
		jid = rawN
	}
	if lidN == "" && strings.HasSuffix(string(jid), "@lid") {
		lidN = jid
	}
	return jid, lidN
}
