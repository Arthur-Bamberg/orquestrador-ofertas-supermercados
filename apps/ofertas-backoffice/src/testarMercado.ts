import type { EntityRecord } from "./domain";

export type TipoFonte = "encarte" | "site";

export function tipoFonte(value: unknown): TipoFonte {
  return value === "site" ? "site" : "encarte";
}

export function fonteDoMercado(fontes: EntityRecord[], mercadoId: string): EntityRecord | undefined {
  if (!mercadoId) {
    return undefined;
  }

  const doMercado = fontes.filter((fonte) => String(fonte.mercadoId ?? "") === mercadoId);
  const ativas = doMercado.filter((fonte) => fonte.ativa !== false);
  const pool = ativas.length > 0 ? ativas : doMercado;

  return [...pool].sort((a, b) => String(a.id ?? "").localeCompare(String(b.id ?? "")))[0];
}

export function precisaTermoColeta(tipo: TipoFonte): boolean {
  return tipo === "site";
}

export function rotuloTestar(tipo: TipoFonte | undefined): string {
  return tipo === "site" ? "Testar scraping" : "Testar scan do encarte";
}

export function payloadTestar(mercadoId: string, termo: string): { mercadoId: string; termo?: string } {
  const trimmed = termo.trim();
  return trimmed ? { mercadoId, termo: trimmed } : { mercadoId };
}
