package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

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

// sqliteBusyTimeoutMS is how long a writer waits for the device-store lock.
// After pairing, whatsmeow writes identities, push names and sender keys from
// several goroutines; without a timeout modernc returns SQLITE_BUSY immediately.
const sqliteBusyTimeoutMS = 10000

func sessionDSN(path string) string {
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(%d)",
		path, sqliteBusyTimeoutMS,
	)
}

type Cliente struct {
	client   *whatsmeow.Client
	mu       sync.RWMutex
	on       bool
	handle   func(context.Context, application.Entrada)
	recibo   func(context.Context, string, domain.StatusEnvio)
	assuntos assuntosGrupo
}

func Ligar(ctx context.Context, sessionPath string) (*Cliente, error) {
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sessionDSN(sessionPath))
	if err != nil {
		return nil, fmt.Errorf("whatsmeow store: %w", err)
	}
	// SQLite allows one writer. whatsmeow's pool would otherwise race and
	// surface SQLITE_BUSY (push name / identity / sender key after connect).
	db.SetMaxOpenConns(1)
	container := sqlstore.NewWithDB(db, "sqlite", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("whatsmeow store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	cli := whatsmeow.NewClient(device, waLog.Stdout("whatsmeow", "INFO", true))
	c := &Cliente{client: cli}
	cli.AddEventHandler(c.onEvent)
	return c, nil
}

func (c *Cliente) carregarAssuntos() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	groups, err := c.client.GetJoinedGroups(ctx)
	if err != nil {
		log.Printf("whatsapp assuntos de grupo: %v", err)
		return
	}
	for _, g := range groups {
		if g == nil {
			continue
		}
		c.assuntos.lembrar(jidChave(g.JID), g.Name)
	}
}

func (c *Cliente) assuntoSeGrupo(grupo bool, chat types.JID) string {
	if !grupo {
		return ""
	}
	return c.assuntos.nome(jidChave(chat))
}

func jidChave(j types.JID) string {
	return j.ToNonAD().String()
}

func (c *Cliente) SetHandler(h func(context.Context, application.Entrada)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handle = h
}

func (c *Cliente) SetReciboHandler(h func(context.Context, string, domain.StatusEnvio)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recibo = h
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
					if err := ImprimirQR(os.Stderr, evt.Code); err != nil {
						fmt.Fprintf(os.Stderr, "QR: %v\n", err)
					}
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

func (c *Cliente) Enviar(ctx context.Context, destino domain.JID, corpo string, midia *domain.MidiaBytes) (string, error) {
	to, err := types.ParseJID(string(destino))
	if err != nil {
		return "", err
	}
	msg, err := c.buildMessage(ctx, corpo, midia)
	if err != nil {
		return "", err
	}
	resp, err := c.client.SendMessage(ctx, to, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
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
		go c.carregarAssuntos()
	case *events.Disconnected:
		c.mu.Lock()
		c.on = false
		c.mu.Unlock()
	case *events.Message:
		c.dispatchMessage(v, domain.OrigemVivo)
	case *events.HistorySync:
		c.handleHistory(v)
	case *events.JoinedGroup:
		if v != nil {
			c.assuntos.lembrar(jidChave(v.JID), v.Name)
		}
	case *events.GroupInfo:
		if v != nil && v.Name != nil {
			c.assuntos.lembrar(jidChave(v.JID), v.Name.Name)
		}
	case *events.UndecryptableMessage:
		c.handleUndecryptable(v)
	case *events.Receipt:
		c.handleReceipt(v)
	}
}

func (c *Cliente) handleHistory(v *events.HistorySync) {
	if v == nil || v.Data == nil {
		return
	}
	n := 0
	for _, conv := range v.Data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		if chatJID.Server == types.GroupServer {
			c.assuntos.lembrar(jidChave(chatJID), conv.GetName())
		}
		for _, hm := range conv.GetMessages() {
			evt, err := c.client.ParseWebMessage(chatJID, hm.GetMessage())
			if err != nil {
				continue
			}
			c.dispatchMessage(evt, domain.OrigemHistorico)
			n++
		}
	}
	for _, st := range v.Data.GetStatusV3Messages() {
		evt, err := c.client.ParseWebMessage(types.StatusBroadcastJID, st)
		if err != nil {
			continue
		}
		c.dispatchMessage(evt, domain.OrigemHistorico)
		n++
	}
	log.Printf("whatsapp history sync ingerido=%d tipo=%s", n, v.Data.GetSyncType())
}

func (c *Cliente) handleUndecryptable(v *events.UndecryptableMessage) {
	if v == nil {
		return
	}
	chatPN, chatLID := splitPNLID(v.Info.Chat, v.Info.RecipientAlt)
	sendPN, sendLID := splitPNLID(v.Info.Sender, v.Info.SenderAlt)
	c.emit(Inbound{
		ProvedorID:   v.Info.ID,
		ChatJID:      v.Info.Chat.String(),
		ChatPN:       chatPN,
		ChatLID:      chatLID,
		SenderJID:    v.Info.Sender.String(),
		SenderPN:     sendPN,
		SenderLID:    sendLID,
		Grupo:        v.Info.IsGroup,
		FromMe:       v.Info.IsFromMe,
		CriadoEm:     v.Info.Timestamp,
		PushName:     v.Info.PushName,
		ConversaNome: c.assuntoSeGrupo(v.Info.IsGroup, v.Info.Chat),
		Origem:       domain.OrigemVivo,
		Tipo:         domain.MensagemIndecifravel,
		Status:       v.Info.Chat.Server == types.BroadcastServer,
	})
}

func (c *Cliente) handleReceipt(v *events.Receipt) {
	c.mu.RLock()
	h := c.recibo
	c.mu.RUnlock()
	if h == nil || v == nil {
		return
	}
	st := statusDoRecibo(v.Type)
	if st == "" {
		return
	}
	for _, id := range v.MessageIDs {
		h(context.Background(), id, st)
	}
}

func statusDoRecibo(t types.ReceiptType) domain.StatusEnvio {
	switch t {
	case types.ReceiptTypeDelivered:
		return domain.StatusEntregue
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		return domain.StatusLido
	case types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
		return domain.StatusReproduzido
	default:
		return ""
	}
}

func (c *Cliente) dispatchMessage(v *events.Message, origem domain.OrigemMensagem) {
	if v == nil {
		return
	}
	chatPN, chatLID := splitPNLID(v.Info.Chat, v.Info.RecipientAlt)
	sendPN, sendLID := splitPNLID(v.Info.Sender, v.Info.SenderAlt)
	tipo, texto, payload := tipoETexto(v)
	in := Inbound{
		ProvedorID:   v.Info.ID,
		ChatJID:      v.Info.Chat.String(),
		ChatPN:       chatPN,
		ChatLID:      chatLID,
		SenderJID:    v.Info.Sender.String(),
		SenderPN:     sendPN,
		SenderLID:    sendLID,
		Grupo:        v.Info.IsGroup,
		FromMe:       v.Info.IsFromMe,
		Texto:        texto,
		PushName:     v.Info.PushName,
		ConversaNome: c.assuntoSeGrupo(v.Info.IsGroup, v.Info.Chat),
		CriadoEm:     v.Info.Timestamp,
		Origem:       origem,
		Tipo:         tipo,
		Payload:      payload,
		Status:       v.Info.Chat.Server == types.BroadcastServer,
	}
	if midia, _ := c.downloadMidia(v); midia != nil {
		in.Midia = midia
		if in.Tipo == "" || in.Tipo == domain.MensagemTexto {
			in.Tipo = domain.MensagemMidia
		}
	}
	c.emit(in)
}

func (c *Cliente) emit(in Inbound) {
	c.mu.RLock()
	h := c.handle
	c.mu.RUnlock()
	if h == nil {
		return
	}
	entrada, ok := Entrada(in)
	if !ok {
		log.Printf("whatsapp ignorada chat=%s provedor=%s", in.ChatJID, in.ProvedorID)
		return
	}
	h(context.Background(), entrada)
}

func splitPNLID(primary, alt types.JID) (pn, lid string) {
	apply := func(j types.JID) {
		if j.IsEmpty() {
			return
		}
		j = j.ToNonAD()
		switch j.Server {
		case types.DefaultUserServer:
			pn = j.String()
		case types.HiddenUserServer:
			lid = j.String()
		}
	}
	apply(primary)
	apply(alt)
	return
}

func tipoETexto(v *events.Message) (domain.TipoMensagem, string, string) {
	if v.Message == nil {
		return "", "", ""
	}
	if r := v.Message.GetReactionMessage(); r != nil {
		alvo := ""
		if r.GetKey() != nil {
			alvo = r.GetKey().GetID()
		}
		return domain.MensagemReacao, r.GetText(), payloadJSON(map[string]string{"alvo": alvo})
	}
	if p := v.Message.GetProtocolMessage(); p != nil && p.GetType() == waE2E.ProtocolMessage_REVOKE {
		alvo := ""
		if p.GetKey() != nil {
			alvo = p.GetKey().GetID()
		}
		return domain.MensagemRevogacao, "", payloadJSON(map[string]string{"alvo": alvo})
	}
	texto := textoDe(v)
	if v.IsEdit {
		return domain.MensagemTexto, texto, payloadJSON(map[string]bool{"edit": true})
	}
	return domain.MensagemTexto, texto, ""
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
		img := v.Message.GetImageMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaImagem, MIME: img.GetMimetype(), Filename: "imagem"}
		b, err := c.client.Download(ctx, img)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case v.Message.GetAudioMessage() != nil:
		a := v.Message.GetAudioMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaAudio, MIME: a.GetMimetype(), Filename: "audio"}
		b, err := c.client.Download(ctx, a)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case v.Message.GetVideoMessage() != nil:
		vid := v.Message.GetVideoMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaVideo, MIME: vid.GetMimetype(), Filename: "video"}
		b, err := c.client.Download(ctx, vid)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case v.Message.GetDocumentMessage() != nil:
		d := v.Message.GetDocumentMessage()
		name := d.GetFileName()
		if name == "" {
			name = "documento"
		}
		stub := &domain.MidiaBytes{Tipo: domain.MidiaDocumento, MIME: d.GetMimetype(), Filename: name}
		b, err := c.client.Download(ctx, d)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case v.Message.GetStickerMessage() != nil:
		stub := &domain.MidiaBytes{Tipo: domain.MidiaFigurinha, Filename: "figurinha"}
		b, err := c.client.Download(ctx, v.Message.GetStickerMessage())
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	}
	return nil, nil
}
