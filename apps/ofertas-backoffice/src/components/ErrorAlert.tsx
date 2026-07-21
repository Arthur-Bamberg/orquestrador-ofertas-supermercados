import { ApiError } from "../api";

type ErrorAlertProps = {
  error: unknown;
};

export function ErrorAlert({ error }: ErrorAlertProps) {
  if (!error) {
    return null;
  }

  if (error instanceof ApiError) {
    return (
      <div className={`alert ${error.status === 409 ? "alert-warning" : "alert-error"}`} role="alert">
        <strong>{error.status === 409 ? "Bloqueio de domínio" : "Erro da API"}</strong>
        <p>{error.message}</p>
        {error.details ? <pre>{JSON.stringify(error.details, null, 2)}</pre> : null}
      </div>
    );
  }

  if (error instanceof Error) {
    return (
      <div className="alert alert-error" role="alert">
        <strong>Erro</strong>
        <p>{error.message}</p>
      </div>
    );
  }

  return (
    <div className="alert alert-error" role="alert">
      <strong>Erro inesperado</strong>
      <pre>{JSON.stringify(error, null, 2)}</pre>
    </div>
  );
}
