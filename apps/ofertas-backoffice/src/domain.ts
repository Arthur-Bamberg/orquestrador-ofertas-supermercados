import type { ColumnFormat, RefKind } from "./display";

export type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue };

export type EntityRecord = Record<string, unknown>;

export type FieldKind =
  | "text"
  | "url"
  | "number"
  | "decimal"
  | "date"
  | "datetime"
  | "textarea"
  | "select"
  | "json"
  | "ref"
  | "boolean";

export type EntityField = {
  name: string;
  label: string;
  kind: FieldKind;
  required?: boolean;
  help?: string;
  options?: string[];
  ref?: RefKind;
};

export type EntityColumn = {
  name: string;
  label: string;
  format?: ColumnFormat;
  ref?: RefKind;
};

export type FilterField = {
  name: string;
  label: string;
  kind: "text" | "date" | "select" | "ref";
  options?: string[];
  ref?: RefKind;
};

export type ResourceConfig = {
  slug: string;
  apiPath: string;
  singular: string;
  plural: string;
  description: string;
  idField?: string;
  columns: EntityColumn[];
  fields: EntityField[];
  filters?: FilterField[];
};

export type ApiListPayload = unknown[] | Record<string, unknown>;

export type ArtifactItem = {
  tentativa: string;
  filepath: string;
  contentType?: string;
  size?: number;
  raw: unknown;
};

export type OperationStatus =
  | "pendente"
  | "pending"
  | "queued"
  | "agendado"
  | "executando"
  | "running"
  | "concluido"
  | "completed"
  | "falhou"
  | "failed"
  | "cancelado"
  | "cancelled"
  | string;
