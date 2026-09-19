package whatsapp

import (
	"context"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
)

func TestConectarComQR_CanalPendentePrimeiraVezNaoDesconecta(t *testing.T) {
	sock := &fakeSocket{}
	err := conectarComQR(context.Background(), false, sock, func(<-chan whatsmeow.QRChannelItem) {})
	if err != nil {
		t.Fatal(err)
	}
	if sock.disconnects != 0 {
		t.Fatalf("desconexões=%d, want 0 no primeiro Connect", sock.disconnects)
	}
	if !sock.gotQR || sock.connects != 1 {
		t.Fatalf("gotQR=%v connects=%d, want GetQRChannel e Connect", sock.gotQR, sock.connects)
	}
}

func TestConectarComQR_aposTimeoutDesconectaAntesDePedirQR(t *testing.T) {
	sock := &fakeSocket{connected: true}
	var watched bool
	err := conectarComQR(context.Background(), false, sock, func(<-chan whatsmeow.QRChannelItem) {
		watched = true
	})
	if err != nil {
		t.Fatalf("renovar QR após timeout: %v", err)
	}
	if sock.disconnects != 1 {
		t.Fatalf("desconexões=%d, want 1 (GetQRChannel exige socket fechado)", sock.disconnects)
	}
	if !sock.gotQR {
		t.Fatal("esperava GetQRChannel depois de desconectar")
	}
	if !watched {
		t.Fatal("esperava voltar a observar o canal de QR")
	}
	if sock.connects != 1 || !sock.connected {
		t.Fatalf("connects=%d connected=%v, want Connect após o QR", sock.connects, sock.connected)
	}
}

type fakeSocket struct {
	connected   bool
	gotQR       bool
	connects    int
	disconnects int
}

func (f *fakeSocket) IsConnected() bool { return f.connected }

func (f *fakeSocket) Disconnect() {
	f.connected = false
	f.disconnects++
}

func (f *fakeSocket) GetQRChannel(context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	if f.connected {
		return nil, errors.New("GetQRChannel must be called before connecting")
	}
	f.gotQR = true
	ch := make(chan whatsmeow.QRChannelItem)
	close(ch)
	return ch, nil
}

func (f *fakeSocket) Connect() error {
	f.connected = true
	f.connects++
	return nil
}
