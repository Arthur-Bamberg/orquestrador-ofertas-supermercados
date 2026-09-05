import { Link } from "react-router-dom";
import {
  type CatalogLookups,
  type ColumnFormat,
  type RefKind,
  formatBoolean,
  formatCurrency,
  formatDate,
  formatDateTime,
  formatQuantidades,
  refRoutes,
  resolveRef,
} from "../display";

export function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }

  if (Array.isArray(value) || typeof value === "object") {
    return JSON.stringify(value);
  }

  return String(value);
}

type ValueViewProps = {
  value: unknown;
};

export function ValueView({ value }: ValueViewProps) {
  if (Array.isArray(value) || (value && typeof value === "object")) {
    return <code>{JSON.stringify(value)}</code>;
  }

  return <>{formatValue(value)}</>;
}

type FormattedValueProps = {
  value: unknown;
  format?: ColumnFormat;
  refKind?: RefKind;
  lookups?: CatalogLookups;
  lookupsLoading?: boolean;
};

export function FormattedValue({ value, format, refKind, lookups, lookupsLoading }: FormattedValueProps) {
  if (format === "ref" && refKind) {
    const resolved = resolveRef(value, lookups?.[refKind]);

    if (!resolved.id) {
      return <>-</>;
    }

    if (resolved.found) {
      return (
        <Link to={`/${refRoutes[refKind]}/${encodeURIComponent(resolved.id)}`} title={resolved.id}>
          {resolved.label}
        </Link>
      );
    }

    return <span title={resolved.id}>{lookupsLoading ? resolved.id : resolved.label}</span>;
  }

  if (format === "date") {
    return <>{formatDate(value)}</>;
  }

  if (format === "datetime") {
    return <>{formatDateTime(value)}</>;
  }

  if (format === "currency") {
    return <>{formatCurrency(value)}</>;
  }

  if (format === "quantidades") {
    return <>{formatQuantidades(value)}</>;
  }

  if (format === "boolean") {
    return <>{formatBoolean(value)}</>;
  }

  return <ValueView value={value} />;
}
