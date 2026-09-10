# Agente de ofertas

Dois papéis num processo: o Agente da Lista reúne encarte e Coletas; o Agente de Resposta organiza e envia. Não é o gateway nem o Extrator. O assistente de catálogo entra no Agente da Lista — mesma cadeia, sem WhatsApp.

## Language

**Agente**:
O app com dois papéis (Agente da Lista, Agente de Resposta). Não é o Canal, não é o Extrator, não é a Coleta.
_Avoid_: bot, matcher, worker, gateway, MCP, Agente de Filtragem (como papel), agente de pós-busca (como entidade), agente pós escolha

**Agente da Lista**:
Papel que recebe a Lista (Canal ou assistente): parte em Itens; para cada Item obtém o Termo (interpretação; se falhar, invólucro) e, se o Termo não for vazio, dispara Coleta nos sites (reusa a do dia se já existir) e lê Ofertas vigentes de encarte — ligadas a Documento, cujo Produto/Marca casa com o Termo. Zero ou vários casamentos no encarte não cancelam a Coleta e não viram Resposta. Espera todas as consultas da Lista e só então chama o Agente de Resposta. Não envia a Resposta. Não escolhe Produto nem Oferta.
_Avoid_: orquestrador, scraper, gateway, Filtragem, Extrator (como quem deriva o Termo)

**Agente de Resposta**:
Papel que organiza a Resposta a partir do que o Agente da Lista reuniu: olha o Termo, fica com o Produto pedido (menos tokens a mais no nome), descarta relacionados e, se o Termo nomeou Marca, restringe a essa Marca. Envia pelo Canal quando a Lista veio de lá. No mesmo Produto e Mercado, vale a Oferta da Coleta do dia se essa Coleta trouxe esse Produto; senão a de encarte. Nesse Produto: a Oferta de menor preço efetivo (Marca e tamanho na linha se houver); empate de preço lista todas. Empate de “menos extras” entre Produtos: fica o de menor preço efetivo; empate desse preço lista esses Produtos. Não busca no site, não consulta o catálogo por conta própria, não deriva o Termo e não casa Item por substring para decidir a Resposta.
_Avoid_: ack, bot reply, gateway (como quem interpreta), Agente de Filtragem, casamento por substring (como regra da Resposta)

**Lista**:
O que a pessoa quer comprar, tirado de **uma** Mensagem viva de texto na Allowlist (não FromMe, não Status, não histórico, não reacção / revogação / indecifrável). Mídia sem texto não é Lista neste recorte. A Mensagem seguinte na mesma Conversa é outra Lista — não acumula. Grupo na Allowlist: a Resposta vai no grupo. O assistente de catálogo também entrega uma Lista, sem Canal.
_Avoid_: Pedido, ordem, carrinho, Mensagem, sessão, lista vigente, lista dividida

**Item**:
Linha da Lista: o texto como a pessoa disse, ainda sem identidade de catálogo. É o título do bloco na Resposta. Não é o Termo da Coleta.
_Avoid_: Produto, Oferta, SKU, query, Termo, agente de pós-busca, preço derivado

**Termo**:
Texto com que a Coleta busca e com que o Agente de Resposta escolhe o Produto: tipo vendável, cultivar ou processo se a pessoa disse, e Marca. Um por Item, depois da partição da Lista. Não lê o catálogo e não inventa o que a pessoa não disse. Vem da interpretação do Item no Agente da Lista; se essa interpretação falhar ou vier vazia, saem do Item (em qualquer posição) tokens de invólucro — verbo de compra, cumprimento, número, Medida, artigo, preposição, pontuação. Termo ainda vazio: aquele Item não dispara Coleta; a Resposta dele diz que não achou.
_Avoid_: Item, query, Extrator, matcher, palavras-chave (como entidade à parte)

**Resposta**:
Texto em lista que o Agente de Resposta monta para uma Lista: um bloco por Item (título = texto do Item) e, se couber, o Produto escolhido (nome visível) com a Oferta de menor preço efetivo (Marca e tamanho na linha se houver; empate de preço lista todas). Vários Produtos só quando empatam em extras **e** nesse menor preço. Preço de um Produto num Mercado vem da Coleta do dia quando ela o trouxe; encarte vigente cobre Mercado sem Coleta do dia (falhou, sem adapter) ou Produto que a Coleta não trouxe. Substitui o Ack nessas Mensagens quando a cadeia está ligada. Não é Comparativo (selo de embalagem no Catálogo).
_Avoid_: Comparativo, cotação, ack, match, bot reply, preço unitário derivado, listar todas as redes, listar todas as Marcas do tipo, um sub-bloco por primo prefix-casado
