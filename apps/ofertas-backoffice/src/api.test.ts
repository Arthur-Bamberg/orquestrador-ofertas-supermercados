import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, normalizeList, request } from "./api";

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

describe("request", () => {
  it("GET JSON from /api path", async () => {
    vi.stubEnv("VITE_API_BASE", "");
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { id: "m1", nome: "Fort" }));
    vi.stubGlobal("fetch", fetchMock);

    const got = await request<{ id: string; nome: string }>("/mercados/m1");

    expect(got).toEqual({ id: "m1", nome: "Fort" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("/api/mercados/m1");
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ credentials: "include" });
  });

  it("throws ApiError 409 with payload error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse(409, { error: "delete blocked: mercado m1 tem fontes" })),
    );

    await expect(request("/mercados/m1", { method: "DELETE" })).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      message: "delete blocked: mercado m1 tem fontes",
    } satisfies Partial<ApiError>);
  });

  it("401 no catálogo manda para /entrar", async () => {
    const assign = vi.fn();
    vi.stubGlobal("window", { location: { pathname: "/mercados", assign } });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(401, { error: "não identificado" })));

    await expect(request("/mercados")).rejects.toMatchObject({ status: 401 });
    expect(assign).toHaveBeenCalledWith("/entrar");
  });

  it("401 em /identificar não redireciona", async () => {
    const assign = vi.fn();
    vi.stubGlobal("window", { location: { pathname: "/entrar", assign } });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(401, { error: "nome ou senha inválidos" })));

    await expect(request("/identificar", { method: "POST", body: { nome: "x", senha: "y" } })).rejects.toMatchObject({
      status: 401,
    });
    expect(assign).not.toHaveBeenCalled();
  });

  it("uses domínio fallback when 409 has no message", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(409, {})));

    await expect(request("/mercados/m1", { method: "DELETE" })).rejects.toMatchObject({
      status: 409,
      message: "Operação bloqueada por conflito de domínio.",
    });
  });
});

describe("identificar", () => {
  it("POST /api/identificar envia nome e senha com credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { nome: "arthur" }));
    vi.stubGlobal("fetch", fetchMock);

    const { identificar } = await import("./api");
    await identificar("arthur", "segredo");

    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("/api/identificar");
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: "POST", credentials: "include" });
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({ nome: "arthur", senha: "segredo" });
  });
});

describe("normalizeList", () => {
  it("returns arrays as-is", () => {
    expect(normalizeList([{ id: "m1" }])).toEqual([{ id: "m1" }]);
  });

  it("unwraps mercados wrapper", () => {
    expect(normalizeList({ mercados: [{ id: "m1" }] })).toEqual([{ id: "m1" }]);
  });
});
