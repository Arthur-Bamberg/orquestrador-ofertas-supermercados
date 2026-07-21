import type { ChangeEvent } from "react";
import type { EntityField, FieldKind } from "../domain";

type FormFieldProps = {
  field: EntityField;
  value: string;
  onChange: (name: string, value: string) => void;
};

export function fieldValueToString(value: unknown, kind: FieldKind): string {
  if (value === null || value === undefined) {
    return "";
  }

  if (kind === "json") {
    return JSON.stringify(value, null, 2);
  }

  return String(value);
}

export function parseFieldValue(field: EntityField, value: string): unknown {
  if (value.trim() === "") {
    return undefined;
  }

  switch (field.kind) {
    case "text":
    case "url":
    case "date":
    case "datetime":
    case "textarea":
    case "select":
      return value;
    case "number":
      return Number.parseInt(value, 10);
    case "decimal":
      return Number.parseFloat(value.replace(",", "."));
    case "json":
      return JSON.parse(value);
    default: {
      const exhaustive: never = field.kind;
      return exhaustive;
    }
  }
}

export function FormField({ field, value, onChange }: FormFieldProps) {
  const inputId = `field-${field.name}`;
  const commonProps = {
    id: inputId,
    name: field.name,
    value,
    required: field.required,
    onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      onChange(field.name, event.target.value);
    },
  };

  return (
    <label className="form-field" htmlFor={inputId}>
      <span>
        {field.label}
        {field.required ? <em aria-label="obrigatório"> *</em> : null}
      </span>
      {renderInput(field, commonProps)}
      {field.help ? <small>{field.help}</small> : null}
    </label>
  );
}

type CommonInputProps = {
  id: string;
  name: string;
  value: string;
  required?: boolean;
  onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => void;
};

function renderInput(field: EntityField, commonProps: CommonInputProps) {
  switch (field.kind) {
    case "textarea":
    case "json":
      return <textarea {...commonProps} rows={field.kind === "json" ? 8 : 4} />;
    case "select":
      return (
        <select {...commonProps}>
          <option value="">Selecione</option>
          {(field.options ?? []).map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      );
    case "url":
      return <input {...commonProps} type="url" />;
    case "number":
      return <input {...commonProps} type="number" step="1" />;
    case "decimal":
      return <input {...commonProps} type="number" step="0.01" />;
    case "date":
      return <input {...commonProps} type="date" />;
    case "datetime":
      return <input {...commonProps} type="datetime-local" />;
    case "text":
      return <input {...commonProps} type="text" />;
    default: {
      const exhaustive: never = field.kind;
      return exhaustive;
    }
  }
}
