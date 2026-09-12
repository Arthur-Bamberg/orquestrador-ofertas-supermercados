import { describe, expect, it } from "vitest";
import { operadorIdentificado } from "./sessao";

describe("operadorIdentificado", () => {
  it("só reconhece Operador com nome", () => {
    expect(operadorIdentificado(undefined)).toBe(false);
    expect(operadorIdentificado({ nome: "" })).toBe(false);
    expect(operadorIdentificado({ nome: "   " })).toBe(false);
    expect(operadorIdentificado({ nome: "arthur" })).toBe(true);
  });
});
