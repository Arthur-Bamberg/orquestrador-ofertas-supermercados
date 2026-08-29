import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, normalizeList, request } from "./api";

afterEach(() => {
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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { id: "m1", nome: "Fort" }));
    vi.stubGlobal("fetch", fetchMock);

    const got = await request<{ id: string; nome: string }>("/mercados/m1");

    expect(got).toEqual({ id: "m1", nome: "Fort" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("/api/mercados/m1");
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

  it("uses domínio fallback when 409 has no message", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(409, {})));

    await expect(request("/mercados/m1", { method: "DELETE" })).rejects.toMatchObject({
      status: 409,
      message: "Operação bloqueada por conflito de domínio.",
    });
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
