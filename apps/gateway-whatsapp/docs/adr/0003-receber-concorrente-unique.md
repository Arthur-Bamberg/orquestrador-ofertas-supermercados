# Concurrent Receber treats unique races as upsert / Duplicada

Inbound events run in parallel goroutines; Contato, Conversa and Mensagem used check-then-insert, so duplicate WhatsApp deliveries hit `contato_jid*`, `conversa_jid*` or `mensagem_provedor_id`. Upserts retry find+merge on 23505; SalvarMensagem maps provedor unique to `ErrProvedorDuplicado` and Receber returns Duplicada (no second Ack/Agente). Rejected: serialising the whole handler (hurts throughput) and ignoring the error in logs only.
