import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./api";
import { conversaRotulo, listConversas, midiaUrl, enviarTexto } from "./gateway";

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
  it("lista Conversas no host do gateway, sem prefixo /api", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, []));
    vi.stubGlobal("fetch", fetchMock);

    await listConversas({ tipo: "grupo", q: "Beto" });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("http://localhost:8090/conversas");
    expect(url).toContain("tipo=grupo");
    expect(url).toContain("q=Beto");
    expect(url).not.toContain("/api/");
  });

  it("recusa envio sem VITE_GATEWAY_TOKEN", async () => {
    vi.stubEnv("VITE_GATEWAY_TOKEN", "");
    await expect(enviarTexto("5511999999999", "oi")).rejects.toThrow(/VITE_GATEWAY_TOKEN/);
  });

  it("envia Bearer em POST /envios", async () => {
    vi.stubEnv("VITE_GATEWAY_TOKEN", "secret");
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, { ID: "m1" }));
    vi.stubGlobal("fetch", fetchMock);

    await enviarTexto("5511999999999", "oi");

    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers;
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/envios");
    expect(headers.get("Authorization")).toBe("Bearer secret");
  });

  it("propaga 403 do gateway como ApiError", async () => {
    vi.stubEnv("VITE_GATEWAY_TOKEN", "secret");
    vi.stubEnv("VITE_GATEWAY_TOKEN", "secret");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(403, { error: "conversa fora da allowlist" })));

    await expect(enviarTexto("120363abc@g.us", "oi")).rejects.toMatchObject({
      name: "ApiError",
      status: 403,
      message: "conversa fora da allowlist",
    } satisfies Partial<ApiError>);
  });

  it("monta URL de Mídia por id da Mensagem", () => {
    expect(midiaUrl("abc")).toBe("http://localhost:8090/mensagens/abc/midia");
  });

  it("rótulo usa JID e pushName da última Mensagem", () => {
    expect(conversaRotulo({ jid: "5511@s.whatsapp.net", ultimaMensagem: { pushName: "Ana" } as never })).toBe(
      "5511@s.whatsapp.net · Ana",
    );
  });
});
