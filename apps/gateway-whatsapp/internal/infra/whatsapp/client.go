package whatsapp

import (
	"context"
	"database/sql"
	"errors"
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
	client    *whatsmeow.Client
	container *sqlstore.Container
	life      context.Context
	mu        sync.RWMutex
	on        bool
	qr        string
	handle    func(context.Context, application.Entrada)
	recibo    func(context.Context, string, domain.StatusEnvio)
	assuntos  assuntosGrupo
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
	c := &Cliente{client: cli, container: container, life: ctx}
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
		chave := jidChave(g.JID)
		c.assuntos.lembrar(chave, g.Name)
		log.Printf("whatsapp grupo disponível: jid=%s nome=%q", chave, g.Name)
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

type socketQR interface {
	IsConnected() bool
	Disconnect()
	GetQRChannel(context.Context) (<-chan whatsmeow.QRChannelItem, error)
	Connect() error
}

func conectarComQR(ctx context.Context, pareado bool, sock socketQR, observar func(<-chan whatsmeow.QRChannelItem)) error {
	if !pareado {
		if sock.IsConnected() {
			sock.Disconnect()
		}
		ch, err := sock.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		observar(ch)
	}
	return sock.Connect()
}

func (c *Cliente) Connect(ctx context.Context) error {
	c.mu.RLock()
	cli := c.client
	qrCtx := c.life
	c.mu.RUnlock()
	if qrCtx == nil {
		qrCtx = ctx
	}
	pareado := cli.Store != nil && cli.Store.ID != nil
	return conectarComQR(qrCtx, pareado, cli, func(ch <-chan whatsmeow.QRChannelItem) {
		go c.watchQR(ch)
	})
}

func (c *Cliente) watchQR(ch <-chan whatsmeow.QRChannelItem) {
	for evt := range ch {
		switch evt.Event {
		case whatsmeow.QRChannelEventCode:
			c.setQR(evt.Code)
			if err := ImprimirQR(os.Stderr, evt.Code); err != nil {
				fmt.Fprintf(os.Stderr, "QR: %v\n", err)
			}
		case whatsmeow.QRChannelSuccess.Event:
			c.setQR("")
		case whatsmeow.QRChannelTimeout.Event:
			c.setQR("")
			go func() {
				log.Printf("whatsapp QR timeout: renovando Pareamento")
				if err := c.Connect(context.Background()); err != nil {
					log.Printf("whatsapp QR timeout: %v", err)
				}
			}()
		default:
			if evt.Event != whatsmeow.QRChannelEventError {
				continue
			}
			c.setQR("")
			log.Printf("whatsapp QR: %v", evt.Error)
		}
	}
}

func (c *Cliente) setQR(code string) {
	c.mu.Lock()
	c.qr = code
	c.mu.Unlock()
}

func (c *Cliente) Disconnect() {
	c.client.Disconnect()
}

func (c *Cliente) Conectado() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.on && c.client != nil && c.client.IsConnected()
}

func (c *Cliente) Situacao() domain.CanalSituacao {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var id *types.JID
	if c.client != nil && c.client.Store != nil {
		id = c.client.Store.ID
	}
	if id == nil {
		return domain.CanalSituacao{Estado: domain.CanalPendente, QR: c.qr}
	}
	jid := domain.JID(id.ToNonAD().String())
	if c.on && c.client.IsConnected() {
		return domain.CanalSituacao{Estado: domain.CanalConectado, JID: jid}
	}
	return domain.CanalSituacao{Estado: domain.CanalDesconectado, JID: jid}
}

func (c *Cliente) Desparear(ctx context.Context) error {
	c.mu.Lock()
	cli := c.client
	container := c.container
	c.mu.Unlock()
	if cli == nil || cli.Store == nil || cli.Store.ID == nil {
		return domain.ErrSemPareamento
	}
	if err := cli.Logout(ctx); err != nil {
		cli.Disconnect()
		if delErr := cli.Store.Delete(ctx); delErr != nil {
			return fmt.Errorf("desparear: %w", errors.Join(err, delErr))
		}
	}
	if container == nil {
		return fmt.Errorf("desparear: store indisponível")
	}
	device := container.NewDevice()
	novo := whatsmeow.NewClient(device, waLog.Stdout("whatsmeow", "INFO", true))
	c.mu.Lock()
	c.client = novo
	c.on = false
	c.qr = ""
	novo.AddEventHandler(c.onEvent)
	c.mu.Unlock()
	return c.Connect(ctx)
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
		c.qr = ""
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
			chave := jidChave(v.JID)
			c.assuntos.lembrar(chave, v.Name)
			log.Printf("whatsapp entrou no grupo: jid=%s nome=%q", chave, v.Name)
		}
	case *events.GroupInfo:
		if v != nil && v.Name != nil {
			chave := jidChave(v.JID)
			c.assuntos.lembrar(chave, v.Name.Name)
			log.Printf("whatsapp info de grupo atualizada: jid=%s nome=%q", chave, v.Name.Name)
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
	if v.Message == nil && v.RawMessage != nil {
		v.Message = v.RawMessage
	}
	v.Message = UnwrapMessage(v.Message)
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
	log.Printf("whatsapp mensagem recebida: chat=%s remetente=%s grupo=%v fromMe=%v tipo=%s texto=%q provedor=%s",
		in.ChatJID, in.SenderJID, in.Grupo, in.FromMe, in.Tipo, in.Texto, in.ProvedorID)
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

// UnwrapMessage unwraps nested message containers until reaching the inner message.
func UnwrapMessage(m *waE2E.Message) *waE2E.Message {
	if m == nil {
		return nil
	}
	for {
		switch {
		case m.GetEphemeralMessage().GetMessage() != nil:
			m = m.GetEphemeralMessage().GetMessage()
		case m.GetDeviceSentMessage().GetMessage() != nil:
			m = m.GetDeviceSentMessage().GetMessage()
		case m.GetViewOnceMessage().GetMessage() != nil:
			m = m.GetViewOnceMessage().GetMessage()
		case m.GetViewOnceMessageV2().GetMessage() != nil:
			m = m.GetViewOnceMessageV2().GetMessage()
		case m.GetViewOnceMessageV2Extension().GetMessage() != nil:
			m = m.GetViewOnceMessageV2Extension().GetMessage()
		case m.GetDocumentWithCaptionMessage().GetMessage() != nil:
			m = m.GetDocumentWithCaptionMessage().GetMessage()
		case m.GetGroupMentionedMessage().GetMessage() != nil:
			m = m.GetGroupMentionedMessage().GetMessage()
		case m.GetBotInvokeMessage().GetMessage() != nil:
			m = m.GetBotInvokeMessage().GetMessage()
		case m.GetEditedMessage().GetMessage() != nil:
			m = m.GetEditedMessage().GetMessage()
		default:
			return m
		}
	}
}

func tipoETexto(v *events.Message) (domain.TipoMensagem, string, string) {
	m := UnwrapMessage(v.Message)
	if m == nil {
		return "", "", ""
	}
	if r := m.GetReactionMessage(); r != nil {
		alvo := ""
		if r.GetKey() != nil {
			alvo = r.GetKey().GetID()
		}
		return domain.MensagemReacao, r.GetText(), payloadJSON(map[string]string{"alvo": alvo})
	}
	if p := m.GetProtocolMessage(); p != nil && p.GetType() == waE2E.ProtocolMessage_REVOKE {
		alvo := ""
		if p.GetKey() != nil {
			alvo = p.GetKey().GetID()
		}
		return domain.MensagemRevogacao, "", payloadJSON(map[string]string{"alvo": alvo})
	}
	texto := TextoDe(m)
	if v.IsEdit {
		return domain.MensagemTexto, texto, payloadJSON(map[string]bool{"edit": true})
	}
	return domain.MensagemTexto, texto, ""
}

// TextoDe extracts the textual content from an unwrapped or wrapped WhatsApp message.
func TextoDe(m *waE2E.Message) string {
	m = UnwrapMessage(m)
	if m == nil {
		return ""
	}
	if t := m.GetConversation(); t != "" {
		return t
	}
	if t := m.GetExtendedTextMessage().GetText(); t != "" {
		return t
	}
	if t := m.GetImageMessage().GetCaption(); t != "" {
		return t
	}
	if t := m.GetVideoMessage().GetCaption(); t != "" {
		return t
	}
	if t := m.GetDocumentMessage().GetCaption(); t != "" {
		return t
	}
	if b := m.GetButtonsResponseMessage(); b != nil {
		if t := b.GetSelectedDisplayText(); t != "" {
			return t
		}
		if t := b.GetSelectedButtonID(); t != "" {
			return t
		}
	}
	if l := m.GetListResponseMessage(); l != nil {
		if t := l.GetTitle(); t != "" {
			return t
		}
	}
	if t := m.GetTemplateButtonReplyMessage().GetSelectedDisplayText(); t != "" {
		return t
	}
	if ir := m.GetInteractiveResponseMessage(); ir != nil {
		if b := ir.GetBody(); b != nil && b.GetText() != "" {
			return b.GetText()
		}
		if nf := ir.GetNativeFlowResponseMessage(); nf != nil && nf.GetParamsJSON() != "" {
			return nf.GetParamsJSON()
		}
	}
	if im := m.GetInteractiveMessage(); im != nil {
		if b := im.GetBody(); b != nil && b.GetText() != "" {
			return b.GetText()
		}
	}
	return ""
}

func (c *Cliente) downloadMidia(v *events.Message) (*domain.MidiaBytes, error) {
	m := UnwrapMessage(v.Message)
	if m == nil {
		return nil, nil
	}
	ctx := context.Background()
	switch {
	case m.GetImageMessage() != nil:
		img := m.GetImageMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaImagem, MIME: img.GetMimetype(), Filename: "imagem"}
		b, err := c.client.Download(ctx, img)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case m.GetAudioMessage() != nil:
		a := m.GetAudioMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaAudio, MIME: a.GetMimetype(), Filename: "audio"}
		b, err := c.client.Download(ctx, a)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case m.GetVideoMessage() != nil:
		vid := m.GetVideoMessage()
		stub := &domain.MidiaBytes{Tipo: domain.MidiaVideo, MIME: vid.GetMimetype(), Filename: "video"}
		b, err := c.client.Download(ctx, vid)
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	case m.GetDocumentMessage() != nil:
		d := m.GetDocumentMessage()
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
	case m.GetStickerMessage() != nil:
		stub := &domain.MidiaBytes{Tipo: domain.MidiaFigurinha, Filename: "figurinha"}
		b, err := c.client.Download(ctx, m.GetStickerMessage())
		if err != nil {
			return stub, err
		}
		stub.Conteudo = b
		return stub, nil
	}
	return nil, nil
}
