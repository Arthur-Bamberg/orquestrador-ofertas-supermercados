# Canal always-on numa VM; Cloud Run não é Canal

Produção do `gateway-whatsapp` é uma VM sempre ligada no projeto GCP, só com esse processo (ADR 0007: whatsmeow, um Canal por processo). Os outros apps ficam no Cloud Run. O Cloud Run `gateway-whatsapp` deixa de existir — dois processos seriam dois Canais. A VM começa sem Pareamento (Canal pendente, QR novo no backoffice); Contato/Conversa/Mensagem continuam no Postgres partilhado. Rejeitado: Cloud Run com min-instances, Agente ou Postgres na VM, e manter Cloud Run + VM ao mesmo tempo.
