import type { ApiListPayload, ArtifactItem, EntityRecord } from "./domain";

export const API_BASE = (import.meta.env.VITE_API_BASE ?? "http://localhost:8080").replace(/\/$/, "");

export class ApiError extends Error {
  status: number;
  details: unknown;

  constructor(status: number, message: string, details: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.details = details;
  }
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  headers?: HeadersInit;
};

function apiUrl(path: string, query?: Record<string, string>): string {
  const normalizedPath = path.startsWith("/api/") ? path : `/api${path.startsWith("/") ? path : `/${path}`}`;
  const url = new URL(`${API_BASE}${normalizedPath}`);

  if (query) {
    Object.entries(query).forEach(([key, value]) => {
      if (value.trim() !== "") {
        url.searchParams.set(key, value);
      }
    });
  }

  return url.toString();
}

async function parseResponse(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";

  if (response.status === 204) {
    return null;
  }

  if (contentType.includes("application/json")) {
    return response.json();
  }

  const text = await response.text();
  return text.length > 0 ? text : null;
}

function errorMessage(status: number, payload: unknown): string {
  if (typeof payload === "string" && payload.trim() !== "") {
    return payload;
  }

  if (payload && typeof payload === "object") {
    const record = payload as Record<string, unknown>;
    const message = record.message ?? record.error ?? record.erro ?? record.detalhe;

    if (typeof message === "string" && message.trim() !== "") {
      return message;
    }
  }

  return status === 409
    ? "Operação bloqueada por conflito de domínio."
    : `A API respondeu com HTTP ${status}.`;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const init: RequestInit = {
    method: options.method ?? "GET",
    headers,
  };

  if (options.body !== undefined) {
    headers.set("content-type", "application/json");
    init.body = JSON.stringify(options.body);
  }

  const response = await fetch(apiUrl(path), init);
  const payload = await parseResponse(response);

  if (!response.ok) {
    throw new ApiError(response.status, errorMessage(response.status, payload), payload);
  }

  return payload as T;
}

export function normalizeList(payload: ApiListPayload | unknown): EntityRecord[] {
  if (Array.isArray(payload)) {
    return payload as EntityRecord[];
  }

  if (!payload || typeof payload !== "object") {
    return [];
  }

  const record = payload as Record<string, unknown>;
  const candidateKeys = ["items", "data", "results", "rows", "mercados", "fontes", "documentos", "ofertas", "produtos", "marcas", "falhas", "usos", "artefatos", "operacoes", "queue", "history"];

  for (const key of candidateKeys) {
    const value = record[key];

    if (Array.isArray(value)) {
      return value as EntityRecord[];
    }
  }

  return [];
}

export function list(path: string, filters: Record<string, string> = {}): Promise<EntityRecord[]> {
  const query = Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value.trim() !== ""),
  );

  return request<ApiListPayload>(withQuery(path, query)).then(normalizeList);
}

export function get(path: string, id: string): Promise<EntityRecord> {
  return request<EntityRecord>(`${path}/${encodeURIComponent(id)}`);
}

export function create(path: string, payload: EntityRecord): Promise<EntityRecord> {
  return request<EntityRecord>(path, { method: "POST", body: payload });
}

export function update(path: string, id: string, payload: EntityRecord): Promise<EntityRecord> {
  return request<EntityRecord>(`${path}/${encodeURIComponent(id)}`, { method: "PUT", body: payload });
}

export function remove(path: string, id: string): Promise<void> {
  return request<void>(`${path}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function withQuery(path: string, query: Record<string, string>): string {
  const params = new URLSearchParams(query);
  const suffix = params.toString();

  return suffix.length > 0 ? `${path}?${suffix}` : path;
}

export function artifactDownloadUrl(documentoId: string, tentativa: string, filepath: string): string {
  const encodedPath = filepath.split("/").map(encodeURIComponent).join("/");
  return apiUrl(`/documentos/${encodeURIComponent(documentoId)}/artefatos/${encodeURIComponent(tentativa)}/${encodedPath}`);
}

export async function uploadArtifact(documentoId: string, tentativa: string, filepath: string, file: File): Promise<unknown> {
  const encodedPath = filepath.split("/").map(encodeURIComponent).join("/");
  const response = await fetch(
    apiUrl(`/documentos/${encodeURIComponent(documentoId)}/artefatos/${encodeURIComponent(tentativa)}/${encodedPath}`),
    {
      method: "PUT",
      headers: file.type ? { "content-type": file.type } : undefined,
      body: file,
    },
  );
  const payload = await parseResponse(response);

  if (!response.ok) {
    throw new ApiError(response.status, errorMessage(response.status, payload), payload);
  }

  return payload;
}

export function deleteArtifact(documentoId: string, tentativa: string, filepath: string): Promise<void> {
  const encodedPath = filepath.split("/").map(encodeURIComponent).join("/");
  return request<void>(`/documentos/${encodeURIComponent(documentoId)}/artefatos/${encodeURIComponent(tentativa)}/${encodedPath}`, {
    method: "DELETE",
  });
}

export function normalizeArtifacts(payload: unknown): ArtifactItem[] {
  const items: ArtifactItem[] = [];

  function add(raw: unknown, fallbackTentativa = ""): void {
    if (typeof raw === "string") {
      items.push({ tentativa: fallbackTentativa, filepath: raw, raw });
      return;
    }

    if (!raw || typeof raw !== "object") {
      return;
    }

    const record = raw as Record<string, unknown>;
    const tentativa = String(record.tentativa ?? record.attempt ?? fallbackTentativa);
    const filepath = record.filepath ?? record.path ?? record.nome ?? record.name;

    if (typeof filepath === "string") {
      items.push({
        tentativa,
        filepath,
        contentType: typeof record.contentType === "string" ? record.contentType : undefined,
        size: typeof record.size === "number" ? record.size : undefined,
        raw,
      });
    }

    for (const key of ["artefatos", "files", "items"]) {
      const nested = record[key];

      if (Array.isArray(nested)) {
        nested.forEach((nestedItem) => add(nestedItem, tentativa));
      }
    }
  }

  if (Array.isArray(payload)) {
    payload.forEach((item) => add(item));
    return items;
  }

  if (payload && typeof payload === "object") {
    const record = payload as Record<string, unknown>;

    for (const key of ["tentativas", "artefatos", "files", "items", "data"]) {
      const value = record[key];

      if (Array.isArray(value)) {
        value.forEach((item) => add(item));
      }
    }

    Object.entries(record).forEach(([key, value]) => {
      if (Array.isArray(value)) {
        value.forEach((item) => add(item, key));
      }
    });
  }

  return items;
}
