import type { ResourceConfig } from "./domain";

export const documentoEstados = ["processando", "concluido", "parcial", "falhou"];
export const medidas = ["g", "ml", "unidade"];
export const origensData = ["extrator", "filename", "primeiraDescoberta", "coleta"];
export const fonteTipos = ["encarte", "site"];

export const resources: ResourceConfig[] = [
  {
    slug: "mercados",
    apiPath: "/mercados",
    singular: "Mercado",
    plural: "Mercados",
    description: "Identidades comerciais usadas para comparar Ofertas.",
    columns: [
      { name: "id", label: "ID" },
      { name: "nome", label: "Nome" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "nome", label: "Nome", kind: "text", required: true },
    ],
  },
  {
    slug: "fontes",
    apiPath: "/fontes",
    singular: "Fonte",
    plural: "Fontes",
    description: "URLs configuradas que revelam Documentos de um Mercado.",
    columns: [
      { name: "id", label: "ID" },
      { name: "mercadoId", label: "Mercado", format: "ref", ref: "mercado" },
      { name: "url", label: "URL" },
      { name: "filtroNomeDocumento", label: "Filtro Documento" },
      { name: "tipo", label: "Tipo" },
      { name: "ativa", label: "Ativa", format: "boolean" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "mercadoId", label: "Mercado", kind: "ref", ref: "mercado", required: true },
      { name: "url", label: "URL", kind: "url", required: true },
      { name: "filtroNomeDocumento", label: "Filtro Nome Documento", kind: "text" },
      { name: "tipo", label: "Tipo", kind: "select", options: fonteTipos },
      { name: "ativa", label: "Ativa", kind: "boolean" },
    ],
    filters: [{ name: "mercadoId", label: "Mercado", kind: "ref", ref: "mercado" }],
  },
  {
    slug: "documentos",
    apiPath: "/documentos",
    singular: "Documento",
    plural: "Documentos",
    description: "PDFs descobertos e rastreados pelo pipeline.",
    columns: [
      { name: "id", label: "ID" },
      { name: "fonteId", label: "Fonte", format: "ref", ref: "fonte" },
      { name: "mercadoId", label: "Mercado", format: "ref", ref: "mercado" },
      { name: "filename", label: "Nome" },
      { name: "dia", label: "Dia", format: "date" },
      { name: "estado", label: "Estado" },
      { name: "atualizado", label: "Atualizado", format: "datetime" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "fonteId", label: "Fonte", kind: "ref", ref: "fonte", required: true },
      { name: "mercadoId", label: "Mercado", kind: "ref", ref: "mercado", required: true },
      { name: "filename", label: "Filename", kind: "text", required: true },
      { name: "dia", label: "Dia", kind: "date", required: true },
      { name: "estado", label: "Estado", kind: "select", options: documentoEstados },
      { name: "fingerprint", label: "Fingerprint", kind: "text" },
      { name: "conteudoIdenticoA", label: "Conteúdo Idêntico A", kind: "ref", ref: "documento" },
      { name: "ultimoErro", label: "Último Erro", kind: "textarea" },
      { name: "atualizado", label: "Atualizado", kind: "datetime" },
    ],
    filters: [
      { name: "fonteId", label: "Fonte", kind: "ref", ref: "fonte" },
      { name: "estado", label: "Estado", kind: "select", options: documentoEstados },
      { name: "dia", label: "Dia", kind: "date" },
    ],
  },
  {
    slug: "ofertas",
    apiPath: "/ofertas",
    singular: "Oferta",
    plural: "Ofertas",
    description: "Observações de preço extraídas ou administradas.",
    columns: [
      { name: "id", label: "ID" },
      { name: "produtoId", label: "Produto", format: "ref", ref: "produto" },
      { name: "marcaId", label: "Marca", format: "ref", ref: "marca" },
      { name: "mercadoId", label: "Mercado", format: "ref", ref: "mercado" },
      { name: "valor", label: "Valor", format: "currency" },
      { name: "quantidades", label: "Quantidades", format: "quantidades" },
      { name: "medida", label: "Medida" },
      { name: "dataInicio", label: "Início", format: "date" },
      { name: "dataExpiracao", label: "Expiração", format: "date" },
      { name: "indicacaoPromocional", label: "Indicação Promocional", format: "boolean" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "documentoId", label: "Documento", kind: "ref", ref: "documento", help: "Exigido pela API ao criar manualmente." },
      { name: "produtoId", label: "Produto", kind: "ref", ref: "produto", required: true },
      { name: "marcaId", label: "Marca", kind: "ref", ref: "marca" },
      { name: "mercadoId", label: "Mercado", kind: "ref", ref: "mercado", required: true },
      { name: "valor", label: "Valor", kind: "decimal", required: true },
      { name: "quantidades", label: "Quantidades", kind: "json", required: true, help: "JSON, ex.: [500, 1000]." },
      { name: "medida", label: "Medida", kind: "select", required: true, options: medidas },
      { name: "dataInicio", label: "Data Início", kind: "date", required: true },
      { name: "dataExpiracao", label: "Data Expiração", kind: "date", required: true },
      { name: "origemDataInicio", label: "Origem Data Início", kind: "select", options: origensData },
      { name: "origemDataExpiracao", label: "Origem Data Expiração", kind: "select", options: origensData },
      { name: "indicacaoPromocional", label: "Indicação Promocional", kind: "boolean" },
      { name: "promocao", label: "Promoção", kind: "json" },
      { name: "comparativo", label: "Comparativo", kind: "json" },
    ],
    filters: [
      { name: "produtoId", label: "Produto", kind: "ref", ref: "produto" },
      { name: "mercadoId", label: "Mercado", kind: "ref", ref: "mercado" },
      { name: "marcaId", label: "Marca", kind: "ref", ref: "marca" },
      { name: "texto", label: "Texto", kind: "text" },
    ],
  },
  {
    slug: "produtos",
    apiPath: "/produtos",
    singular: "Produto",
    plural: "Produtos",
    description: "Identidades de catálogo sem Marca.",
    columns: [
      { name: "id", label: "ID" },
      { name: "nome", label: "Nome" },
      { name: "nomeNorm", label: "Nome Normalizado" },
      { name: "categorias", label: "Categorias" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "nome", label: "Nome", kind: "text", required: true },
      { name: "nomeNorm", label: "Nome Normalizado", kind: "text" },
      { name: "categorias", label: "Categorias", kind: "json", help: "JSON, ex.: [\"mercearia\", \"arroz\"]." },
    ],
    filters: [
      { name: "nome", label: "Nome", kind: "text" },
      { name: "categoria", label: "Categoria", kind: "text" },
    ],
  },
  {
    slug: "marcas",
    apiPath: "/marcas",
    singular: "Marca",
    plural: "Marcas",
    description: "Identidades comerciais de fabricantes ou rótulos.",
    columns: [
      { name: "id", label: "ID" },
      { name: "nome", label: "Nome" },
      { name: "nomeNorm", label: "Nome Normalizado" },
    ],
    fields: [
      { name: "id", label: "ID", kind: "text", help: "Opcional se a API gerar IDs." },
      { name: "nome", label: "Nome", kind: "text", required: true },
      { name: "nomeNorm", label: "Nome Normalizado", kind: "text" },
    ],
  },
];

export function getResource(slug: string | undefined): ResourceConfig | undefined {
  return resources.find((resource) => resource.slug === slug);
}
