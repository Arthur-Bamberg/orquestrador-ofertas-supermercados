import type { ChangeEvent } from "react";
import type { CatalogLookups, RefKind } from "../display";
import { mergeRefOptions, refOptions } from "../display";

type RefSelectProps = {
  id?: string;
  name?: string;
  kind: RefKind;
  lookups: CatalogLookups;
  required?: boolean;
  allowEmpty?: boolean;
  emptyLabel?: string;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
};

export function RefSelect({
  id,
  name,
  kind,
  lookups,
  required,
  allowEmpty = true,
  emptyLabel = "Todos",
  value,
  defaultValue,
  onChange,
}: RefSelectProps) {
  const current = value ?? defaultValue ?? "";
  const options = mergeRefOptions(refOptions(lookups, kind), current);
  const controlled = onChange !== undefined;

  return (
    <select
      id={id}
      name={name}
      required={required}
      {...(controlled
        ? {
            value: value ?? "",
            onChange: (event: ChangeEvent<HTMLSelectElement>) => onChange(event.target.value),
          }
        : { defaultValue: defaultValue ?? "" })}
    >
      {allowEmpty ? <option value="">{emptyLabel}</option> : null}
      {options.map((option) => (
        <option key={option.value} value={option.value} title={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );
}
