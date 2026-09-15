import { ApiError } from "./api";

export const GATEWAY_BASE = (import.meta.env.VITE_GATEWAY_BASE ?? "/gateway").replace(/\/$/, "");

export type MidiaResumo = {
  tipo: string;
  filename?: string;
  mime?: string;
};

export type Mensagem = {
  id: string;
  conversaId: string;
  contatoId?: string;
  direcao: string;
  corpo: string;
  provedorId?: string;
  status?: string;
  criadoEm: string;
  origem?: string;
  pushName?: string;
  tipo?: string;
  payload?: string;
  midia?: MidiaResumo;
};

export type ConversaResumo = {
  id: string;
  jid: string;
  jidLid?: string;
  tipo: string;
  totalMensagens: number;
  permitido: boolean;
  ultimaMensagem?: Mensagem;
};

export type FiltroConversas = {
  tipo?: string;
  q?: string;
};

export type PaginaMensagens = {
  limit?: number;
  antesCriadoEm?: string;
  antesId?: string;
};

type RequestOptions = {
  method?: string;
  body?: unknown;
};

function gatewayUrl(path: string, query?: Record<string, string>): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  const params = new URLSearchParams();
  if (query) {
    Object.entries(query).forEach(([key, value]) => {
      if (value.trim() !== "") {
        params.set(key, value);
      }
    });
  }
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return `${GATEWAY_BASE}${normalized}${suffix}`;
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
    const message = record.message ?? record.error ?? record.erro;
    if (typeof message === "string" && message.trim() !== "") {
      return message;
    }
  }
  return `O gateway respondeu com HTTP ${status}.`;
}

export async function gatewayRequest<T>(path: string, query?: Record<string, string>, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers();
  const init: RequestInit = {
    method: options.method ?? "GET",
    headers,
    credentials: "include",
  };
  if (options.body !== undefined) {
    headers.set("content-type", "application/json");
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch(gatewayUrl(path, query), init);
  const payload = await parseResponse(response);
  if (!response.ok) {
    if (response.status === 401 && typeof window !== "undefined" && window.location.pathname !== "/entrar") {
      window.location.assign("/entrar");
    }
    throw new ApiError(response.status, errorMessage(response.status, payload), payload);
  }
  return payload as T;
}

export function listConversas(filters: FiltroConversas = {}): Promise<ConversaResumo[]> {
  return gatewayRequest<ConversaResumo[]>("/conversas", {
    tipo: filters.tipo ?? "",
    q: filters.q ?? "",
  });
}

export function getConversa(id: string): Promise<ConversaResumo> {
  return gatewayRequest<ConversaResumo>(`/conversas/${encodeURIComponent(id)}`);
}

export function listMensagens(conversaId: string, pagina: PaginaMensagens = {}): Promise<Mensagem[]> {
  return gatewayRequest<Mensagem[]>(`/conversas/${encodeURIComponent(conversaId)}/mensagens`, {
    limit: pagina.limit != null ? String(pagina.limit) : "",
    antesCriadoEm: pagina.antesCriadoEm ?? "",
    antesId: pagina.antesId ?? "",
  });
}

export function enviarTexto(conversaJid: string, corpo: string): Promise<unknown> {
  return gatewayRequest("/envios", undefined, {
    method: "POST",
    body: { conversaJid, corpo },
  });
}

export function midiaUrl(mensagemId: string): string {
  return gatewayUrl(`/mensagens/${encodeURIComponent(mensagemId)}/midia`);
}

export type CanalSituacao = {
  estado: "pronto" | "nao_configurado" | string;
  jid?: string;
};

export function getCanal(): Promise<CanalSituacao> {
  return gatewayRequest<CanalSituacao>("/canal");
}

export function conversaRotulo(conversa: Pick<ConversaResumo, "jid" | "ultimaMensagem">): string {
  const push = conversa.ultimaMensagem?.pushName?.trim();
  if (push) {
    return `${conversa.jid} · ${push}`;
  }
  return conversa.jid;
}
