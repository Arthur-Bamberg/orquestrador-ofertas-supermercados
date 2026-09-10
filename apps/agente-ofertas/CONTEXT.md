# Agente de ofertas

Dois papéis num processo: o Agente da Lista reúne encarte e Coletas; o Agente de Resposta organiza e envia. Não é o gateway nem o Extrator. O assistente de catálogo entra no Agente da Lista — mesma cadeia, sem WhatsApp.

## Language

**Agente**:
O app com dois papéis (Agente da Lista, Agente de Resposta). Não é o Canal, não é o Extrator, não é a Coleta.
_Avoid_: bot, matcher, worker, gateway, MCP, Agente de Filtragem (como papel), agente de pós-busca (como entidade)

**Agente da Lista**:
Papel que recebe a Lista (Canal ou assistente): parte em Itens; para cada Item dispara Coleta nos sites (reusa a do dia se já existir) e lê Ofertas vigentes de encarte — ligadas a Documento, cujo Produto/Marca casa com o texto do Item. Zero ou vários casamentos no encarte não cancelam a Coleta e não viram Resposta. Espera todas as consultas da Lista e só então chama o Agente de Resposta. Não envia a Resposta.
_Avoid_: orquestrador, scraper, gateway, Filtragem

**Agente de Resposta**:
Papel que organiza a Resposta a partir do que o Agente da Lista reuniu: olha o texto do Item, fica só com os tipos vendáveis pedidos (Produto), descarta relacionados e, se o Item nomeou Marca, restringe a essa Marca. Envia pelo Canal quando a Lista veio de lá. No mesmo Produto e Mercado, vale a Oferta da Coleta do dia se essa Coleta trouxe esse Produto; senão a de encarte. Por tipo: a Oferta de menor preço efetivo (Marca e tamanho na linha se houver); empate lista todas. Vários tipos que valem: um sub-bloco por Produto. Não busca no site, não consulta o catálogo por conta própria e não casa Item por substring para decidir a Resposta.
_Avoid_: ack, bot reply, gateway (como quem interpreta), Agente de Filtragem, casamento por substring (como regra da Resposta)

**Lista**:
O que a pessoa quer comprar, tirado de **uma** Mensagem viva de texto na Allowlist (não FromMe, não Status, não histórico, não reacção / revogação / indecifrável). Mídia sem texto não é Lista neste recorte. A Mensagem seguinte na mesma Conversa é outra Lista — não acumula. Grupo na Allowlist: a Resposta vai no grupo. O assistente de catálogo também entrega uma Lista, sem Canal.
_Avoid_: Pedido, ordem, carrinho, Mensagem, sessão, lista vigente, lista dividida

**Item**:
Linha da Lista: o texto como a pessoa disse, ainda sem identidade de catálogo; esse texto é o termo da Coleta e a chave do casamento no encarte. Relacionados ficam de fora da Resposta (molho quando pediu tomate) — quem decide é o Agente de Resposta, não o casamento de encarte. Marca no texto restringe a Resposta a essa Marca; sem Oferta dessa Marca, não achou. Zero tipos vendáveis: a Resposta diz que não achou. Vários tipos que valem (tomate e tomate italiano): cada um com a Oferta mais barata daquele tipo — não um único mínimo entre tipos, não preço unitário derivado.
_Avoid_: Produto, Oferta, SKU, query, agente de pós-busca, preço derivado

**Resposta**:
Texto em lista que o Agente de Resposta monta para uma Lista: um bloco por Item e, se couber, um sub-bloco por tipo vendável, cada um com a Oferta de menor preço efetivo daquele tipo (Marca e tamanho na linha se houver; empate lista todas). Preço de um Produto num Mercado vem da Coleta do dia quando ela o trouxe; encarte vigente cobre Mercado sem Coleta do dia (falhou, sem adapter) ou Produto que a Coleta não trouxe. Substitui o Ack nessas Mensagens quando a cadeia está ligada. Não é Comparativo (selo de embalagem no Catálogo).
_Avoid_: Comparativo, cotação, ack, match, bot reply, preço unitário derivado, listar todas as redes, listar todas as Marcas do tipo
