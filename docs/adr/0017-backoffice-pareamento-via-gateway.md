# Backoffice pareia o Canal via HTTP do gateway

QR, estado do Canal e Desparear ficam no `gateway-whatsapp` (`GET`/`POST /canal`, Bearer `GATEWAY_TOKEN`) e o SPA lê essa superfície — o rastro GET continua aberto (ADR 0009), mas o QR é capacidade de tomar o Canal. `/ready` permanece sonda binária. QR no stderr continua para compose. Rejeitado: parear só pelos logs, websocket, e passar o Pareamento pela `ofertas-api`.

Status: superseded by [0020](./0020-gateway-whatsapp-cloud-api.md). The Operador still reads Canal state via `GET /canal` (cookie, ADR 0018); there is no QR and no Desparear.
