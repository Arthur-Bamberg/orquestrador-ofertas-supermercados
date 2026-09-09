import { describe, expect, it } from "vitest";
import { fonteDoMercado, payloadTestar, precisaTermoColeta, rotuloTestar, tipoFonte } from "./testarMercado";

describe("testar Mercado", () => {
  it("tipo site dispara scraping e encarte dispara scan sem IA", () => {
    expect(tipoFonte("site")).toBe("site");
    expect(tipoFonte("encarte")).toBe("encarte");
    expect(tipoFonte(undefined)).toBe("encarte");
    expect(precisaTermoColeta("site")).toBe(true);
    expect(precisaTermoColeta("encarte")).toBe(false);
    expect(rotuloTestar("site")).toBe("Testar scraping");
    expect(rotuloTestar("encarte")).toBe("Testar scan do encarte");
  });

  it("escolhe a Fonte ativa do Mercado", () => {
    const got = fonteDoMercado(
      [
        { id: "fonte-via", mercadoId: "mercado-via", tipo: "encarte", ativa: true },
        { id: "fonte-fort-off", mercadoId: "mercado-fort", tipo: "site", ativa: false },
        { id: "fonte-fort", mercadoId: "mercado-fort", tipo: "site", ativa: true },
      ],
      "mercado-fort",
    );

    expect(got?.id).toBe("fonte-fort");
  });

  it("payload de teste manda termo só quando preenchido", () => {
    expect(payloadTestar("mercado-via", "  ")).toEqual({ mercadoId: "mercado-via" });
    expect(payloadTestar("mercado-fort", " tomate ")).toEqual({ mercadoId: "mercado-fort", termo: "tomate" });
  });
});
