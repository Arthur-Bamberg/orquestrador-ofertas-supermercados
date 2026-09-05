package shopfully

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var dcFlyerRe = regexp.MustCompile(`(?s)window\.DCFlyer\s*=\s*(\{.*?\});\s*`)

type dcFlyer struct {
	PublicationID string `json:"publicationId"`
	ID            string `json:"id"`
}

// ParsePublicationID reads window.DCFlyer.publicationId from a flyer viewer HTML.
func ParsePublicationID(htmlBody string) (string, error) {
	m := dcFlyerRe.FindStringSubmatch(htmlBody)
	if m == nil {
		return "", fmt.Errorf("DCFlyer ausente")
	}
	var flyer dcFlyer
	if err := json.Unmarshal([]byte(m[1]), &flyer); err != nil {
		return "", fmt.Errorf("DCFlyer json: %w", err)
	}
	if flyer.PublicationID == "" {
		return "", fmt.Errorf("publicationId vazio")
	}
	return flyer.PublicationID, nil
}
