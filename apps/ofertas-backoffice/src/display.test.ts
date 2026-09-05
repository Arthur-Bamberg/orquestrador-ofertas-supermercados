import { describe, expect, it } from "vitest";
import {
  buildCatalogLookups,
  formatBoolean,
  formatCurrency,
  formatDate,
  formatDateTime,
  formatQuantidades,
  isAtiva,
  refOptions,
  resolveRef,
} from "./display";

describe("formatDate", () => {
  it("mostra vigência YYYY-MM-DD como dia/mês/ano em pt-BR sem mudar o calendário", () => {
    expect(formatDate("2026-08-29")).toBe("29/08/2026");
  });

  it("mostra vazio como hífen", () => {
    expect(formatDate(null)).toBe("-");
    expect(formatDate("")).toBe("-");
  });
});

describe("formatDateTime", () => {
  it("mostra instante UTC em America/Sao_Paulo como dd/MM/yyyy HH:mm", () => {
    expect(formatDateTime("2026-08-29T15:04:00.000Z")).toBe("29/08/2026 12:04");
  });
});

describe("formatCurrency", () => {
  it("mostra valor da Oferta como moeda BRL", () => {
    expect(formatCurrency(19.9)).toBe("R$ 19,90");
  });
});

describe("formatQuantidades", () => {
  it("mostra lista de tamanhos sem colchetes JSON", () => {
    expect(formatQuantidades([990])).toBe("990");
    expect(formatQuantidades([500, 1000])).toBe("500 / 1.000");
  });
});

describe("formatBoolean / isAtiva", () => {
  it("Fonte ativa omite ou true → Sim; false → Não", () => {
    expect(formatBoolean(true)).toBe("Sim");
    expect(formatBoolean(undefined)).toBe("Sim");
    expect(formatBoolean(false)).toBe("Não");
    expect(isAtiva(true)).toBe(true);
    expect(isAtiva(undefined)).toBe(true);
    expect(isAtiva(false)).toBe(false);
  });
});

describe("resolveRef", () => {
  it("troca o id da FK pelo nome do catálogo", () => {
    const labels = new Map([["241d996b", "Arroz integral"]]);
    expect(resolveRef("241d996b", labels)).toEqual({
      id: "241d996b",
      label: "Arroz integral",
      found: true,
    });
  });

  it("mantém o id quando o registro não existe mais", () => {
    expect(resolveRef("apagado", new Map())).toEqual({
      id: "apagado",
      label: "apagado (não encontrado)",
      found: false,
    });
  });

  it("mostra hífen quando a FK é opcional e está vazia", () => {
    expect(resolveRef(undefined, new Map())).toEqual({
      id: "",
      label: "-",
      found: false,
    });
  });
});

describe("buildCatalogLookups", () => {
  it("indexa Produto e Marca por nome, Fonte por id e Documento por filename", () => {
    const lookups = buildCatalogLookups({
      produtos: [{ id: "p1", nome: "Arroz integral" }],
      marcas: [{ id: "m1", nome: "Camil" }],
      mercados: [{ id: "mercado-fort", nome: "Fort Atacadista" }],
      fontes: [{ id: "fonte-fort", mercadoId: "mercado-fort" }],
      documentos: [{ id: "d1", filename: "RS_Fort.pdf", dia: "2026-08-29" }],
    });

    expect(lookups.produto.get("p1")).toBe("Arroz integral");
    expect(lookups.marca.get("m1")).toBe("Camil");
    expect(lookups.mercado.get("mercado-fort")).toBe("Fort Atacadista");
    expect(lookups.fonte.get("fonte-fort")).toBe("fonte-fort");
    expect(lookups.documento.get("d1")).toBe("RS_Fort.pdf (29/08/2026)");
  });

  it("lista opções de select ordenadas pelo rótulo", () => {
    const lookups = buildCatalogLookups({
      produtos: [
        { id: "p2", nome: "Feijão" },
        { id: "p1", nome: "Arroz integral" },
      ],
    });

    expect(refOptions(lookups, "produto")).toEqual([
      { value: "p1", label: "Arroz integral" },
      { value: "p2", label: "Feijão" },
    ]);
  });
});
