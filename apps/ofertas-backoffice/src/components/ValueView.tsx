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
