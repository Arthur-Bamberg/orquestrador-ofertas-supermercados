# Gateway notifies the Agente over HTTP; Item matches Produto by substring

The Canal already persists Mensagem and exposes `POST /envios`. After persist, if the Mensagem would have been Acked and the Agente URL is set, the gateway POSTs the Lista text to the Agente instead of sending Ack; the Agente replies through `POST /envios`. Item→Produto uses case-insensitive contains on `nomeNorm` (no live Extrator in this slice). Rejected: NATS until poll/HTTP is not enough, LLM matcher in the default suite, and Ack plus Resposta on the same Mensagem.
