# Agente de ofertas

Interpreta o que alguém quer comprar, consulta Ofertas do catálogo e devolve uma Resposta. Não opera o Canal nem extrai encartes. O assistente de catálogo é a mesma interpretação, sem WhatsApp.

## Language

**Agente**:
Quem interpreta um pedido de compra contra o catálogo e produz a Resposta. Não é o Canal (não transporta WhatsApp) nem o Extrator (não lê encarte).
_Avoid_: bot, matcher, worker, gateway, MCP (como entidade)

**Lista**:
O que a pessoa quer comprar, tirado de **uma** Mensagem viva de texto na Allowlist (não FromMe, não Status, não histórico, não reacção / revogação / indecifrável). Mídia sem texto não é Lista neste recorte. A Mensagem seguinte na mesma Conversa é outra Lista — não acumula. Grupo na Allowlist: a Resposta vai no grupo.
_Avoid_: Pedido, ordem, carrinho, Mensagem, sessão, lista vigente, lista dividida

**Item**:
Linha da Lista: o texto como a pessoa disse, ainda sem identidade de catálogo. Zero Produtos: a Resposta diz que não achou. Vários: lista os candidatos (a próxima Mensagem é outra Lista, mais específica se a pessoa quiser). Um: a(s) Oferta(s) **vigente(s) hoje** desse Produto com o **menor preço** (`dataInicio` ≤ hoje ≤ `dataExpiracao`; hoje = dia civil em America/Sao_Paulo; preço efetivo = `valorPromocional` se houver promoção, senão `valor`). Empate no menor preço: lista todos os Mercados empatados. Se nenhuma vigente: diz que não há Oferta agora. Histórico fica de fora. Marca só filtra se o Item a trouxer (é a Marca do Catálogo); sem Marca no Item, qualquer Marca (incluindo ausente).
_Avoid_: Produto, Oferta, SKU, query

**Resposta**:
Texto em lista que o Agente envia para uma Lista: um bloco por Item, com o(s) Mercado(s) mais barato(s) e o preço. Substitui o Ack nessas Mensagens quando o Agente está ligado. Não é Comparativo (selo de embalagem no Catálogo).
_Avoid_: Comparativo, cotação, ack, match, bot reply
