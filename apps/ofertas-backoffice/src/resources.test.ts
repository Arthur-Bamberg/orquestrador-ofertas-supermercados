import { describe, expect, it } from "vitest";
import { documentoEstados, fonteTipos, getResource, medidas, origensData } from "./resources";

describe("constantes de domínio", () => {
  it("documentoEstados match persisted Documento states", () => {
    expect(documentoEstados).toEqual(["processando", "concluido", "parcial", "falhou"]);
  });

  it("medidas are the normalized Medida set", () => {
    expect(medidas).toEqual(["g", "ml", "unidade"]);
  });

  it("fonteTipos are encarte and site", () => {
    expect(fonteTipos).toEqual(["encarte", "site"]);
  });

  it("origensData match vigência origens", () => {
    expect(origensData).toEqual(["extrator", "filename", "primeiraDescoberta", "coleta"]);
  });

  it("getResource finds Oferta config", () => {
    const ofertas = getResource("ofertas");
    expect(ofertas?.singular).toBe("Oferta");
    expect(ofertas?.fields.some((field) => field.name === "medida" && field.options === medidas)).toBe(true);
  });

  it("lista de Ofertas declara FKs, datas, moeda e quantidades", () => {
    const ofertas = getResource("ofertas");
    expect(ofertas?.columns).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "produtoId", format: "ref", ref: "produto" }),
        expect.objectContaining({ name: "marcaId", format: "ref", ref: "marca" }),
        expect.objectContaining({ name: "mercadoId", format: "ref", ref: "mercado" }),
        expect.objectContaining({ name: "valor", format: "currency" }),
        expect.objectContaining({ name: "quantidades", format: "quantidades" }),
        expect.objectContaining({ name: "dataInicio", format: "date" }),
        expect.objectContaining({ name: "dataExpiracao", format: "date" }),
      ]),
    );
  });

  it("filtros e formulários de Oferta escolhem o relacionamento, não o UUID cru", () => {
    const ofertas = getResource("ofertas");
    expect(ofertas?.filters).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "produtoId", kind: "ref", ref: "produto" }),
        expect.objectContaining({ name: "mercadoId", kind: "ref", ref: "mercado" }),
        expect.objectContaining({ name: "marcaId", kind: "ref", ref: "marca" }),
      ]),
    );
    expect(ofertas?.fields).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "produtoId", kind: "ref", ref: "produto" }),
        expect.objectContaining({ name: "marcaId", kind: "ref", ref: "marca" }),
        expect.objectContaining({ name: "mercadoId", kind: "ref", ref: "mercado" }),
        expect.objectContaining({ name: "documentoId", kind: "ref", ref: "documento" }),
      ]),
    );
  });

  it("Fontes e Documentos também resolvem Mercado/Fonte nas listas", () => {
    expect(getResource("fontes")?.columns).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "mercadoId", format: "ref", ref: "mercado" }),
        expect.objectContaining({ name: "tipo" }),
        expect.objectContaining({ name: "ativa", format: "boolean" }),
      ]),
    );
    expect(getResource("fontes")?.fields).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "tipo", kind: "select", options: fonteTipos }),
        expect.objectContaining({ name: "ativa", kind: "boolean" }),
      ]),
    );
    expect(getResource("documentos")?.columns).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: "fonteId", format: "ref", ref: "fonte" }),
        expect.objectContaining({ name: "mercadoId", format: "ref", ref: "mercado" }),
        expect.objectContaining({ name: "dia", format: "date" }),
        expect.objectContaining({ name: "atualizado", format: "datetime" }),
      ]),
    );
  });
});
