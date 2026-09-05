import type { ChangeEvent } from "react";

type AtivaToggleProps = {
  ativa: boolean;
  disabled?: boolean;
  onChange: (next: boolean) => void;
  id?: string;
};

export function AtivaToggle({ ativa, disabled, onChange, id }: AtivaToggleProps) {
  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    onChange(event.target.checked);
  }

  return (
    <label className={`ativa-toggle${ativa ? " is-on" : " is-off"}`} htmlFor={id}>
      <input
        id={id}
        type="checkbox"
        role="switch"
        checked={ativa}
        disabled={disabled}
        onChange={handleChange}
        aria-label={ativa ? "Fonte ativa" : "Fonte inativa"}
      />
      <span className="ativa-toggle-track" aria-hidden="true">
        <span className="ativa-toggle-thumb" />
      </span>
      <span className="ativa-toggle-label">{ativa ? "Ativa" : "Inativa"}</span>
    </label>
  );
}
