package whatsapp

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	_ "modernc.org/sqlite"
)

type Cliente struct {
	client *whatsmeow.Client
	mu     sync.RWMutex
	on     bool
	handle func(context.Context, application.Entrada)
}

func Ligar(ctx context.Context, sessionPath string) (*Cliente, error) {
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + sessionPath + "?_pragma=foreign_keys(1)"
	container, err := sqlstore.New(ctx, "sqlite", dsn, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("whatsmeow store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	cli := whatsmeow.NewClient(device, waLog.Stdout("whatsmeow", "INFO", true))
	c := &Cliente{client: cli}
	cli.AddEventHandler(c.onEvent)
	return c, nil
}

func (c *Cliente) SetHandler(h func(context.Context, application.Entrada)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handle = h
}

func (c *Cliente) Connect(ctx context.Context) error {
	if c.client.Store.ID == nil {
		ch, err := c.client.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		go func() {
			for evt := range ch {
				if evt.Event == "code" {
					fmt.Fprintf(os.Stderr, "WhatsApp QR (pareie o número dedicado):\n%s\n", evt.Code)
				}
			}
		}()
	}
	return c.client.Connect()
}

func (c *Cliente) Disconnect() {
	c.client.Disconnect()
}

func (c *Cliente) Conectado() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.on && c.client.IsConnected()
}

func (c *Cliente) Enviar(ctx context.Context, destino domain.JID, corpo string, midia *domain.MidiaBytes) error {
	to, err := types.ParseJID(string(destino))
	if err != nil {
		return err
	}
	msg, err := c.buildMessage(ctx, corpo, midia)
	if err != nil {
		return err
	}
	_, err = c.client.SendMessage(ctx, to, msg)
	return err
}

func (c *Cliente) buildMessage(ctx context.Context, corpo string, midia *domain.MidiaBytes) (*waE2E.Message, error) {
	if midia == nil || len(midia.Conteudo) == 0 {
		return &waE2E.Message{Conversation: proto.String(corpo)}, nil
	}
	mediaType, err := uploadType(midia.Tipo)
	if err != nil {
		return nil, err
	}
	up, err := c.client.Upload(ctx, midia.Conteudo, mediaType)
	if err != nil {
		return nil, err
	}
	mimetype := midia.MIME
	if mimetype == "" {
		mimetype = mime.TypeByExtension(filepath.Ext(midia.Filename))
	}
	switch midia.Tipo {
	case domain.MidiaImagem:
		return &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(corpo),
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
		}}, nil
	case domain.MidiaAudio:
		return &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
		}}, nil
	case domain.MidiaVideo:
		return &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(corpo),
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
		}}, nil
	case domain.MidiaDocumento:
		return &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(corpo),
			FileName:      proto.String(midia.Filename),
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
		}}, nil
	case domain.MidiaFigurinha:
		return &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
		}}, nil
	default:
		return &waE2E.Message{Conversation: proto.String(corpo)}, nil
	}
}

func uploadType(t domain.TipoMidia) (whatsmeow.MediaType, error) {
	switch t {
	case domain.MidiaImagem, domain.MidiaFigurinha:
		return whatsmeow.MediaImage, nil
	case domain.MidiaAudio:
		return whatsmeow.MediaAudio, nil
	case domain.MidiaVideo:
		return whatsmeow.MediaVideo, nil
	case domain.MidiaDocumento:
		return whatsmeow.MediaDocument, nil
	default:
		return "", fmt.Errorf("tipo de mídia desconhecido: %s", t)
	}
}

func (c *Cliente) onEvent(raw any) {
	switch v := raw.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.on = true
		c.mu.Unlock()
	case *events.Disconnected:
		c.mu.Lock()
		c.on = false
		c.mu.Unlock()
	case *events.Message:
		c.handleMessage(v)
	}
}

func (c *Cliente) handleMessage(v *events.Message) {
	c.mu.RLock()
	h := c.handle
	c.mu.RUnlock()
	if h == nil {
		return
	}
	in := Inbound{
		ProvedorID: v.Info.ID,
		ChatJID:    v.Info.Chat.String(),
		SenderJID:  v.Info.Sender.String(),
		Grupo:      v.Info.IsGroup,
		FromMe:     v.Info.IsFromMe,
		Texto:      textoDe(v),
	}
	if midia, err := c.downloadMidia(v); err == nil {
		in.Midia = midia
	}
	entrada, ok := Entrada(in)
	if !ok {
		return
	}
	h(context.Background(), entrada)
}

func textoDe(v *events.Message) string {
	if v.Message == nil {
		return ""
	}
	if t := v.Message.GetConversation(); t != "" {
		return t
	}
	if t := v.Message.GetExtendedTextMessage().GetText(); t != "" {
		return t
	}
	if t := v.Message.GetImageMessage().GetCaption(); t != "" {
		return t
	}
	if t := v.Message.GetVideoMessage().GetCaption(); t != "" {
		return t
	}
	if t := v.Message.GetDocumentMessage().GetCaption(); t != "" {
		return t
	}
	return ""
}

func (c *Cliente) downloadMidia(v *events.Message) (*domain.MidiaBytes, error) {
	if v.Message == nil {
		return nil, nil
	}
	ctx := context.Background()
	switch {
	case v.Message.GetImageMessage() != nil:
		b, err := c.client.Download(ctx, v.Message.GetImageMessage())
		if err != nil {
			return nil, err
		}
		img := v.Message.GetImageMessage()
		return &domain.MidiaBytes{Tipo: domain.MidiaImagem, MIME: img.GetMimetype(), Filename: "imagem", Conteudo: b}, nil
	case v.Message.GetAudioMessage() != nil:
		b, err := c.client.Download(ctx, v.Message.GetAudioMessage())
		if err != nil {
			return nil, err
		}
		a := v.Message.GetAudioMessage()
		return &domain.MidiaBytes{Tipo: domain.MidiaAudio, MIME: a.GetMimetype(), Filename: "audio", Conteudo: b}, nil
	case v.Message.GetVideoMessage() != nil:
		b, err := c.client.Download(ctx, v.Message.GetVideoMessage())
		if err != nil {
			return nil, err
		}
		vid := v.Message.GetVideoMessage()
		return &domain.MidiaBytes{Tipo: domain.MidiaVideo, MIME: vid.GetMimetype(), Filename: "video", Conteudo: b}, nil
	case v.Message.GetDocumentMessage() != nil:
		b, err := c.client.Download(ctx, v.Message.GetDocumentMessage())
		if err != nil {
			return nil, err
		}
		d := v.Message.GetDocumentMessage()
		name := d.GetFileName()
		if name == "" {
			name = "documento"
		}
		return &domain.MidiaBytes{Tipo: domain.MidiaDocumento, MIME: d.GetMimetype(), Filename: name, Conteudo: b}, nil
	case v.Message.GetStickerMessage() != nil:
		b, err := c.client.Download(ctx, v.Message.GetStickerMessage())
		if err != nil {
			return nil, err
		}
		return &domain.MidiaBytes{Tipo: domain.MidiaFigurinha, Filename: "figurinha", Conteudo: b}, nil
	}
	return nil, nil
}
