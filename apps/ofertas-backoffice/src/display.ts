export type RefKind = "mercado" | "produto" | "marca" | "fonte" | "documento";

export type ColumnFormat = "text" | "date" | "datetime" | "currency" | "ref" | "quantidades" | "boolean";

export type CatalogLookups = Record<RefKind, Map<string, string>>;

export type CatalogLists = {
  mercados?: Record<string, unknown>[];
  produtos?: Record<string, unknown>[];
  marcas?: Record<string, unknown>[];
  fontes?: Record<string, unknown>[];
  documentos?: Record<string, unknown>[];
};

export type RefOption = {
  value: string;
  label: string;
};

export const refRoutes: Record<RefKind, string> = {
  mercado: "mercados",
  produto: "produtos",
  marca: "marcas",
  fonte: "fontes",
  documento: "documentos",
};

export const refApiPaths: Record<RefKind, string> = {
  mercado: "/mercados",
  produto: "/produtos",
  marca: "/marcas",
  fonte: "/fontes",
  documento: "/documentos",
};

export function emptyCatalogLookups(): CatalogLookups {
  return {
    mercado: new Map(),
    produto: new Map(),
    marca: new Map(),
    fonte: new Map(),
    documento: new Map(),
  };
}

export function formatDate(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  const raw = String(value);
  const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(raw);

  if (dateOnly) {
    const year = Number(dateOnly[1]);
    const month = Number(dateOnly[2]);
    const day = Number(dateOnly[3]);
    const parsed = new Date(year, month - 1, day);

    return new Intl.DateTimeFormat("pt-BR", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    }).format(parsed);
  }

  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return raw;
  }

  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    timeZone: "America/Sao_Paulo",
  }).format(parsed);
}

const saoPauloDateTime = new Intl.DateTimeFormat("pt-BR", {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
  timeZone: "America/Sao_Paulo",
});

export function formatDateTime(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  const parsed = value instanceof Date ? value : new Date(String(value));
  if (Number.isNaN(parsed.getTime())) {
    return String(value);
  }

  return saoPauloDateTime.format(parsed).replace(",", "").replace(/\s+/g, " ");
}

export function formatCurrency(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  const amount = typeof value === "number" ? value : Number(String(value).replace(",", "."));
  if (Number.isNaN(amount)) {
    return String(value);
  }

  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  })
    .format(amount)
    .replace(/\u00a0/g, " ");
}

export function formatQuantidades(value: unknown): string {
  if (!Array.isArray(value) || value.length === 0) {
    return value === null || value === undefined || value === "" || (Array.isArray(value) && value.length === 0)
      ? "-"
      : String(value);
  }

  return value.map((item) => new Intl.NumberFormat("pt-BR").format(Number(item))).join(" / ");
}

export function formatBoolean(value: unknown): string {
  if (value === false || value === "false") {
    return "Não";
  }
  if (value === true || value === "true" || value === null || value === undefined || value === "") {
    return "Sim";
  }
  return String(value);
}

/** Fonte.ativa defaults to true when omitted (nil / absent). */
export function isAtiva(value: unknown): boolean {
  return value !== false && value !== "false";
}

export function resolveRef(
  id: unknown,
  labels: Map<string, string> | undefined,
): { id: string; label: string; found: boolean } {
  if (id === null || id === undefined || id === "") {
    return { id: "", label: "-", found: false };
  }

  const key = String(id);
  const label = labels?.get(key);
  if (label) {
    return { id: key, label, found: true };
  }

  return { id: key, label: `${key} (não encontrado)`, found: false };
}

function mapRecords(
  records: Record<string, unknown>[] | undefined,
  label: (record: Record<string, unknown>) => string,
): Map<string, string> {
  const out = new Map<string, string>();

  for (const record of records ?? []) {
    if (record.id === null || record.id === undefined || record.id === "") {
      continue;
    }

    out.set(String(record.id), label(record));
  }

  return out;
}

function documentoLabel(record: Record<string, unknown>): string {
  const filename = String(record.filename ?? record.id ?? "");
  if (typeof record.dia === "string" && record.dia !== "") {
    return `${filename} (${formatDate(record.dia)})`;
  }

  return filename;
}

export function buildCatalogLookups(lists: CatalogLists): CatalogLookups {
  return {
    mercado: mapRecords(lists.mercados, (record) => String(record.nome ?? record.id ?? "")),
    produto: mapRecords(lists.produtos, (record) => String(record.nome ?? record.id ?? "")),
    marca: mapRecords(lists.marcas, (record) => String(record.nome ?? record.id ?? "")),
    fonte: mapRecords(lists.fontes, (record) => String(record.id ?? "")),
    documento: mapRecords(lists.documentos, documentoLabel),
  };
}

export function refOptions(lookups: CatalogLookups, kind: RefKind): RefOption[] {
  return [...lookups[kind].entries()]
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => a.label.localeCompare(b.label, "pt-BR"));
}

export function mergeRefOptions(options: RefOption[], current: string): RefOption[] {
  if (current === "" || options.some((option) => option.value === current)) {
    return options;
  }

  return [{ value: current, label: `${current} (não encontrado)` }, ...options];
}
