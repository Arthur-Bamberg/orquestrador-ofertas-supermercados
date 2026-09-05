# Um adapter de vitrine por Mercado

Cada Mercado (Fort, Carrefour, Asun, Rissul) tem código próprio para entrar no site, pesquisar e ler o cartão. O use case só vê a porta `Vitrine.Buscar`. Rejeitado: um parser HTML genérico compartilhado como se os sites fossem iguais, e descobrir produtos novos a partir da busca — a Coleta só enriquece o Produto já conhecido.
