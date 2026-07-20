package filenamedate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

var (
	isoDateRe = regexp.MustCompile(`(20\d{2})[-_]?(\d{2})[-_]?(\d{2})`)
	// e.g. 18-e-19_JUL_26 or 18-E-19-JUL-26
	rangeRe = regexp.MustCompile(`(?i)(\d{1,2})-e-(\d{1,2})[_-]?([a-z]{3})[_-]?(\d{2})`)
)

var mesPT = map[string]int{
	"jan": 1, "fev": 2, "mar": 3, "abr": 4, "mai": 5, "jun": 6,
	"jul": 7, "ago": 8, "set": 9, "out": 10, "nov": 11, "dez": 12,
}

// Parser extracts vigência hints from a Documento filename (ADR 0028).
type Parser struct{}

func (Parser) Parse(filename string) domain.FilenameVigencia {
	if m := rangeRe.FindStringSubmatch(filename); m != nil {
		mes, ok := mesPT[strings.ToLower(m[3])]
		if !ok {
			return domain.FilenameVigencia{}
		}
		ano, err := strconv.Atoi(m[4])
		if err != nil {
			return domain.FilenameVigencia{}
		}
		diaIni, err1 := strconv.Atoi(m[1])
		diaFim, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil || diaIni < 1 || diaIni > 31 || diaFim < 1 || diaFim > 31 {
			return domain.FilenameVigencia{}
		}
		y := 2000 + ano
		inicio := fmt.Sprintf("%04d-%02d-%02d", y, mes, diaIni)
		fim := fmt.Sprintf("%04d-%02d-%02d", y, mes, diaFim)
		return domain.FilenameVigencia{DataInicio: inicio, DataExpiracao: fim}
	}

	if m := isoDateRe.FindStringSubmatch(filename); m != nil {
		return domain.FilenameVigencia{
			DataExpiracao: m[1] + "-" + m[2] + "-" + m[3],
		}
	}
	return domain.FilenameVigencia{}
}
