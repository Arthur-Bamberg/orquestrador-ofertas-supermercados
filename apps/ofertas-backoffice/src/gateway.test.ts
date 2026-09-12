import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./api";
import { conversaRotulo, desparearCanal, getCanal, listConversas, midiaUrl, enviarTexto } from "./gateway";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("gateway client", () => {
  it("lista Conversas no proxy /gateway, sem prefixo /api", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, []));
    vi.stubGlobal("fetch", fetchMock);

    await listConversas({ tipo: "grupo", q: "Beto" });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("/gateway/conversas");
    expect(url).toContain("tipo=grupo");
    expect(url).toContain("q=Beto");
    expect(url).not.toContain("/api/");
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ credentials: "include" });
  });

  it("POST /envios envia cookie, sem Bearer", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, { ID: "m1" }));
    vi.stubGlobal("fetch", fetchMock);

    await enviarTexto("5511999999999", "oi");

    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers;
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/envios");
    expect(headers.get("Authorization")).toBeNull();
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ credentials: "include" });
  });

  it("propaga 403 do gateway como ApiError", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(403, { error: "conversa fora da allowlist" })));

    await expect(enviarTexto("120363abc@g.us", "oi")).rejects.toMatchObject({
      name: "ApiError",
      status: 403,
      message: "conversa fora da allowlist",
    } satisfies Partial<ApiError>);
  });

  it("monta URL de Mídia por id da Mensagem", () => {
    expect(midiaUrl("abc")).toBe("/gateway/mensagens/abc/midia");
  });

  it("rótulo usa JID e pushName da última Mensagem", () => {
    expect(conversaRotulo({ jid: "5511@s.whatsapp.net", ultimaMensagem: { pushName: "Ana" } as never })).toBe(
      "5511@s.whatsapp.net · Ana",
    );
  });

  it("GET /canal lê estado do Canal sem Bearer", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { estado: "pendente", qrPngBase64: "iVBOR" }));
    vi.stubGlobal("fetch", fetchMock);

    const got = await getCanal();

    expect(got.estado).toBe("pendente");
    expect(got.qrPngBase64).toBe("iVBOR");
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/gateway/canal");
    expect(String(fetchMock.mock.calls[0]?.[0])).not.toContain("/api/");
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers;
    expect(headers.get("Authorization")).toBeNull();
  });

  it("POST /canal/desparear usa cookie", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { estado: "pendente" }));
    vi.stubGlobal("fetch", fetchMock);

    await desparearCanal();

    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/canal/desparear");
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBe("POST");
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers;
    expect(headers.get("Authorization")).toBeNull();
  });
});
