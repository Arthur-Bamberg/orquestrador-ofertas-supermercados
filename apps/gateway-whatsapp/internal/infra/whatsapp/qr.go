package whatsapp

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mdp/qrterminal/v3"
)

func ImprimirQR(w io.Writer, code string) error {
	var buf bytes.Buffer
	if _, err := fmt.Fprintln(&buf, "WhatsApp QR — aponte o celular (Aparelhos conectados):"); err != nil {
		return err
	}
	qrterminal.GenerateHalfBlock(code, qrterminal.L, &buf)
	_, err := w.Write(buf.Bytes())
	return err
}
