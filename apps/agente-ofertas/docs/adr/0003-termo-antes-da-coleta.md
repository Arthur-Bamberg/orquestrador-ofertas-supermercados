# Termo deriva do Item antes da Coleta; interpretação não escolhe Produto

A Coleta busca e o Agente de Resposta escolhem pelo **Termo**, não pelo texto cru do Item (`Comprar: cenoura` não vai ao site). O Agente da Lista interpreta cada Item já partido (sem ler o catálogo) e devolve o Termo; se a interpretação falhar ou vier vazia, cai a classe fechada de invólucro; Termo ainda vazio não dispara Coleta. Rejeitado: LLM matcher Item→Produto (ADR 0001), a interpretação partir a Lista de novo, e Coleta com o Item cru como recurso.
