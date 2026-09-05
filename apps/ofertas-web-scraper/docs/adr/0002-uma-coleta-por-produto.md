# Unidade de execução é um Produto

O CLI `coletar <produtoId>` corre as vitrines daquele Produto e sai. Isola falha e deixa paralelizar por processo. `run` no mesmo binário itera Produtos em série chamando a mesma unidade. Rejeitado: um job único que mistura todos os Produtos sem fronteira de processo.
