# Canal WhatsApp via Cloud API (Meta)

O `gateway-whatsapp` fala WhatsApp por HTTP da Cloud API: entrada num webhook (`GET` verify + `POST` assinado) que chama o mesmo `Receber()`; saída pela Graph API. O processo é HTTP como a `ofertas-api`, inclusive Cloud Run com scale-to-zero. Pareamento, QR, sessão sqlite e websocket whatsmeow saem. O recorte de produto é Conversa direta; grupo e Status não são ingeridos. Mensagem livre só dentro da Janela de 24 h; fora dela, Template. Rejeitado: manter whatsmeow ao lado, sidecar tipo Evolution, e fundir o canal na `ofertas-api`.

Status: accepted

Supersedes the transport and deploy model of [0007](./0007-gateway-whatsapp-whatsmeow.md) and the pairing surface of [0017](./0017-backoffice-pareamento-via-gateway.md). Schema `whatsapp` and Aceite ([0019](./0019-aceite-no-contato-em-vez-de-allowlist.md)) remain.
