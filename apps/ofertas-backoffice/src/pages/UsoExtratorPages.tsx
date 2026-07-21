import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { normalizeList, request } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { FormField, fieldValueToString, parseFieldValue } from "../components/FormField";
import { ValueView } from "../components/ValueView";
import type { EntityField, EntityRecord } from "../domain";

const usoFields: EntityField[] = [
  { name: "documentoId", label: "Documento ID", kind: "text", required: true },
  { name: "tentativa", label: "Tentativa", kind: "number", required: true },
  { name: "artefatoPath", label: "Path do Artefato", kind: "text", required: true },
  { name: "provider", label: "Provider", kind: "text" },
  { name: "model", label: "Model", kind: "text", required: true },
  { name: "promptTokens", label: "Prompt Tokens", kind: "number" },
  { name: "cacheTokens", label: "Cache Tokens", kind: "number" },
  { name: "outputTokens", label: "Output Tokens", kind: "number" },
  { name: "paginas", label: "Páginas", kind: "json", help: "JSON opcional com detalhe por página." },
];

const usoColumns = ["documentoId", "tentativa", "provider", "model", "promptTokens", "cacheTokens", "outputTokens", "artefatoPath"];

function usoPath(documentoId: string, tentativa: string): string {
  return `/usos-extrator/${encodeURIComponent(documentoId)}/${encodeURIComponent(tentativa)}`;
}

function usoKey(uso: EntityRecord): string {
  return `${String(uso.documentoId ?? "")}:${String(uso.tentativa ?? "")}`;
}

export function UsoExtratorListPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [draftDocumentoId, setDraftDocumentoId] = React.useState(searchParams.get("documentoId") ?? "");
  const documentoId = searchParams.get("documentoId") ?? "";
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["usos-extrator", documentoId],
    queryFn: () => request<unknown>(`/usos-extrator${documentoId ? `?documentoId=${encodeURIComponent(documentoId)}` : ""}`).then(normalizeList),
  });
  const deleteMutation = useMutation({
    mutationFn: (uso: EntityRecord) => {
      const currentDocumentoId = String(uso.documentoId ?? "");
      const tentativa = String(uso.tentativa ?? "");
      return request<void>(usoPath(currentDocumentoId, tentativa), { method: "DELETE" });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["usos-extrator"] }),
  });

  function applyFilter(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextParams = new URLSearchParams();

    if (draftDocumentoId.trim() !== "") {
      nextParams.set("documentoId", draftDocumentoId.trim());
    }

    setSearchParams(nextParams);
  }

  function removeUso(uso: EntityRecord) {
    if (window.confirm(`Apagar Uso do Extrator ${usoKey(uso)}?`)) {
      deleteMutation.mutate(uso);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>Uso do Extrator</h2>
          <p>Consumo de tokens e modelo usado em cada tentativa de extração.</p>
        </div>
        <Link className="button primary" to="novo">
          Novo Uso do Extrator
        </Link>
      </header>

      <form className="filters" onSubmit={applyFilter}>
        <label>
          <span>Documento ID</span>
          <input value={draftDocumentoId} onChange={(event) => setDraftDocumentoId(event.target.value)} />
        </label>
        <div className="filter-actions">
          <button type="submit">Filtrar</button>
          <button type="button" onClick={() => setSearchParams(new URLSearchParams())}>
            Limpar
          </button>
          {documentoId ? (
            <Link className="button" to={`/documentos/${encodeURIComponent(documentoId)}`}>
              Ver Documento
            </Link>
          ) : null}
        </div>
      </form>

      <ErrorAlert error={query.error ?? deleteMutation.error} />
      {query.isLoading ? <p>Carregando Uso do Extrator...</p> : null}
      {query.data ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                {usoColumns.map((column) => (
                  <th key={column}>{column}</th>
                ))}
                <th>Ações</th>
              </tr>
            </thead>
            <tbody>
              {query.data.length === 0 ? (
                <tr>
                  <td colSpan={usoColumns.length + 1}>Nenhum Uso do Extrator encontrado.</td>
                </tr>
              ) : (
                query.data.map((uso) => {
                  const currentDocumentoId = String(uso.documentoId ?? "");
                  const tentativa = String(uso.tentativa ?? "");

                  return (
                    <tr key={usoKey(uso)}>
                      {usoColumns.map((column) => (
                        <td key={column}>
                          <ValueView value={uso[column]} />
                        </td>
                      ))}
                      <td className="actions">
                        {currentDocumentoId && tentativa ? (
                          <>
                            <Link to={`${encodeURIComponent(currentDocumentoId)}/${encodeURIComponent(tentativa)}`}>Detalhe</Link>
                            <Link to={`${encodeURIComponent(currentDocumentoId)}/${encodeURIComponent(tentativa)}/editar`}>Editar</Link>
                            <button type="button" onClick={() => removeUso(uso)} disabled={deleteMutation.isPending}>
                              Apagar
                            </button>
                          </>
                        ) : (
                          "Sem chave"
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

export function UsoExtratorDetailPage() {
  const { documentoId = "", tentativa = "" } = useParams();
  const decodedDocumentoId = decodeURIComponent(documentoId);
  const decodedTentativa = decodeURIComponent(tentativa);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const query = useQuery({
    queryKey: ["uso-extrator-detail", decodedDocumentoId, decodedTentativa],
    queryFn: () => request<EntityRecord>(usoPath(decodedDocumentoId, decodedTentativa)),
  });
  const deleteMutation = useMutation({
    mutationFn: () => request<void>(usoPath(decodedDocumentoId, decodedTentativa), { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["usos-extrator"] });
      navigate("/usos-extrator");
    },
  });

  function removeUso() {
    if (window.confirm(`Apagar Uso do Extrator ${decodedDocumentoId}:${decodedTentativa}?`)) {
      deleteMutation.mutate();
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>Uso do Extrator</h2>
          <p>
            Documento {decodedDocumentoId}, tentativa {decodedTentativa}
          </p>
        </div>
        <div className="header-actions">
          <Link className="button" to="/usos-extrator">
            Voltar
          </Link>
          <Link className="button" to="editar">
            Editar
          </Link>
          <button type="button" className="danger" onClick={removeUso} disabled={deleteMutation.isPending}>
            Apagar
          </button>
        </div>
      </header>

      <ErrorAlert error={query.error ?? deleteMutation.error} />
      {query.isLoading ? <p>Carregando detalhe...</p> : null}
      {query.data ? (
        <dl className="detail-list">
          {Object.entries(query.data).map(([key, value]) => (
            <div key={key}>
              <dt>{key}</dt>
              <dd>
                <ValueView value={value} />
              </dd>
            </div>
          ))}
        </dl>
      ) : null}
    </section>
  );
}

export function UsoExtratorFormPage({ mode }: { mode: "create" | "edit" }) {
  const { documentoId = "", tentativa = "" } = useParams();
  const decodedDocumentoId = decodeURIComponent(documentoId);
  const decodedTentativa = decodeURIComponent(tentativa);
  const [values, setValues] = React.useState<Record<string, string>>(
    Object.fromEntries(usoFields.map((field) => [field.name, ""])),
  );
  const [parseError, setParseError] = React.useState<Error | null>(null);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const detailQuery = useQuery({
    queryKey: ["uso-extrator-detail", decodedDocumentoId, decodedTentativa],
    queryFn: () => request<EntityRecord>(usoPath(decodedDocumentoId, decodedTentativa)),
    enabled: mode === "edit",
  });
  const mutation = useMutation({
    mutationFn: (payload: EntityRecord) => {
      if (mode === "create") {
        return request<EntityRecord>("/usos-extrator", { method: "POST", body: payload });
      }

      return request<EntityRecord>(usoPath(decodedDocumentoId, decodedTentativa), { method: "PUT", body: payload });
    },
    onSuccess: (entity) => {
      queryClient.invalidateQueries({ queryKey: ["usos-extrator"] });
      const nextDocumentoId = String(entity.documentoId ?? values.documentoId);
      const nextTentativa = String(entity.tentativa ?? values.tentativa);
      navigate(`/usos-extrator/${encodeURIComponent(nextDocumentoId)}/${encodeURIComponent(nextTentativa)}`);
    },
  });

  React.useEffect(() => {
    if (detailQuery.data) {
      setValues(
        Object.fromEntries(
          usoFields.map((field) => [field.name, fieldValueToString(detailQuery.data?.[field.name], field.kind)]),
        ),
      );
    }
  }, [detailQuery.data]);

  function changeField(name: string, value: string) {
    setValues((current) => ({ ...current, [name]: value }));
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setParseError(null);

    try {
      const payload = Object.fromEntries(
        usoFields
          .map((field) => [field.name, parseFieldValue(field, values[field.name] ?? "")] as const)
          .filter(([, value]) => value !== undefined),
      );
      mutation.mutate(payload);
    } catch (error) {
      setParseError(error instanceof Error ? error : new Error("JSON inválido."));
    }
  }

  return (
    <section className="page narrow">
      <header className="page-header">
        <div>
          <h2>{mode === "create" ? "Novo Uso do Extrator" : "Editar Uso do Extrator"}</h2>
          <p>Registre ou ajuste consumo de tokens de uma tentativa.</p>
        </div>
        <Link className="button" to="/usos-extrator">
          Cancelar
        </Link>
      </header>

      <ErrorAlert error={parseError ?? detailQuery.error ?? mutation.error} />
      {detailQuery.isLoading ? <p>Carregando registro...</p> : null}
      {mode === "create" || detailQuery.data ? (
        <form className="entity-form" onSubmit={submit}>
          {usoFields.map((field) => (
            <FormField key={field.name} field={field} value={values[field.name] ?? ""} onChange={changeField} />
          ))}
          <div className="form-actions">
            <button type="submit" className="primary" disabled={mutation.isPending}>
              Salvar
            </button>
          </div>
        </form>
      ) : null}
    </section>
  );
}
