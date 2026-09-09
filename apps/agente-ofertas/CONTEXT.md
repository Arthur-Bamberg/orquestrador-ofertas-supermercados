# Agente de ofertas

Três papéis num processo: parte a Lista, dispara Coleta em paralelo, filtra tipos (quando a flag pede) e manda a Resposta pelo Canal. Não é o gateway nem o Extrator. O assistente de catálogo entra no Agente da Lista, sem WhatsApp.

## Language

**Agente**:
O app com três papéis (Agente da Lista, Agente de Filtragem, Agente de Resposta). Não é o Canal, não é o Extrator, não é a Coleta.
_Avoid_: bot, matcher, worker, gateway, MCP, agente de pós-busca (como entidade)

**Agente da Lista**:
Papel chamado pelo Canal: parte a Lista em Itens, dispara Coleta de cada Item em paralelo nos sites, junta catálogo e dados consultados, e liga a flag de múltiplos tipos quando a base e/ou o consultado têm mais de um tipo vendável. Não envia a Resposta.
_Avoid_: orquestrador, scraper, gateway

**Agente de Filtragem**:
Papel opcional: só corre com a flag de múltiplos tipos. Descarta relacionados e devolve os tipos vendáveis com o preço observado e onde está o mais barato, olhando o catálogo e o que a Coleta consultou. Não busca no site e não envia a Resposta.
_Avoid_: agente de pós-busca, matcher, validador

**Agente de Resposta**:
Papel que monta o texto da Resposta e chama o Canal. Recebe o resultado da Filtragem ou, sem flag, o da Lista depois das Coletas.
_Avoid_: ack, bot reply, gateway (como quem interpreta)

**Lista**:
O que a pessoa quer comprar, tirado de **uma** Mensagem viva de texto na Allowlist (não FromMe, não Status, não histórico, não reacção / revogação / indecifrável). Mídia sem texto não é Lista neste recorte. A Mensagem seguinte na mesma Conversa é outra Lista — não acumula. Grupo na Allowlist: a Resposta vai no grupo.
_Avoid_: Pedido, ordem, carrinho, Mensagem, sessão, lista vigente, lista dividida

**Item**:
Linha da Lista: o texto como a pessoa disse, ainda sem identidade de catálogo; esse texto é o termo da Coleta. Relacionados ficam de fora da Resposta (molho quando pediu tomate). Zero tipos vendáveis: a Resposta diz que não achou. Vários tipos: cada tipo com o preço observado — não preço unitário derivado, não um único mínimo entre tipos. O mesmo tipo com várias Marcas no mesmo Mercado: cada Marca com o preço observado. O mais barato por tipo cruza catálogo e dados consultados.
_Avoid_: Produto, Oferta, SKU, query, agente de pós-busca, preço derivado

**Resposta**:
Texto em lista que o Agente de Resposta envia para uma Lista: um bloco por Item, com os tipos que valem, os preços observados e onde está o mais barato (base + consultado). Substitui o Ack nessas Mensagens quando a cadeia está ligada. Não é Comparativo (selo de embalagem no Catálogo).
_Avoid_: Comparativo, cotação, ack, match, bot reply, preço unitário derivado
