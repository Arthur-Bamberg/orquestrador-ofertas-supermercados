# Resposta lista a Oferta mais barata por tipo vendável

Para cada tipo vendável que vale no Termo, a Resposta inclui só a(s) Oferta(s) com o menor preço efetivo (`valorPromocional` se houver promoção, senão `valor`); empate de preço lista todas. Marca e tamanho vão na linha. Marca no Termo restringe. Formato em lista (bloco por Item, `- Mercado — preço / tamanho`). Rejected: listar todos os Mercados (ruído no WhatsApp), listar todas as Marcas do tipo, e preço unitário normalizado por medida (fica para um ADR futuro).

Status: accepted. A cláusula “vários tipos que valem → um sub-bloco por Produto” foi substituída pelo [ADR 0004](./0004-resposta-um-produto-menos-extras.md).
