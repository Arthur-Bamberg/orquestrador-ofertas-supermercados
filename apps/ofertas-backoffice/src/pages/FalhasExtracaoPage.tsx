import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { normalizeList, request } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { ValueView } from "../components/ValueView";
import type { EntityRecord } from "../domain";

const emptyFalha = {
  codigo: "",
  detalhe: "",
  candidato: {},
};

export function FalhasExtracaoPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const documentoId = searchParams.get("documentoId") ?? "";
  const [draftDocumentoId, setDraftDocumentoId] = React.useState(documentoId);
  const [editor, setEditor] = React.useState("[]");
  const [parseError, setParseError] = React.useState<Error | null>(null);
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["falhas-extracao", documentoId],
    queryFn: () => request<unknown>(`/documentos/${encodeURIComponent(documentoId)}/falhas`),
    enabled: documentoId.trim() !== "",
  });
  const falhas = normalizeList(query.data);
  const saveMutation = useMutation({
    mutationFn: (payload: EntityRecord[]) =>
      request<unknown>(`/documentos/${encodeURIComponent(documentoId)}/falhas`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["falhas-extracao", documentoId] }),
  });

  React.useEffect(() => {
    if (query.data !== undefined) {
      setEditor(JSON.stringify(falhas, null, 2));
    }
  }, [query.data]);

  function applyFilter(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextParams = new URLSearchParams();

    if (draftDocumentoId.trim() !== "") {
      nextParams.set("documentoId", draftDocumentoId.trim());
    }

    setSearchParams(nextParams);
  }

  function parseEditor(): EntityRecord[] {
    const parsed = JSON.parse(editor);

    if (!Array.isArray(parsed)) {
      throw new Error("Falhas de Extração devem ser um array JSON.");
    }

    return parsed as EntityRecord[];
  }

  function saveEditor(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setParseError(null);

    try {
      saveMutation.mutate(parseEditor());
    } catch (error) {
      setParseError(error instanceof Error ? error : new Error("JSON inválido."));
    }
  }

  function appendFalha() {
    setParseError(null);

    try {
      setEditor(JSON.stringify([...parseEditor(), emptyFalha], null, 2));
    } catch (error) {
      setParseError(error instanceof Error ? error : new Error("JSON inválido."));
    }
  }

  function removeFalha(index: number) {
    if (!window.confirm(`Remover Falha de Extração #${index + 1}?`)) {
      return;
    }

    setParseError(null);

    try {
      setEditor(JSON.stringify(parseEditor().filter((_, itemIndex) => itemIndex !== index), null, 2));
    } catch (error) {
      setParseError(error instanceof Error ? error : new Error("JSON inválido."));
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>Falhas de Extração</h2>
          <p>Registros rejeitados na validação do Extrator, sempre vinculados a um Documento.</p>
        </div>
      </header>

      <form className="filters" onSubmit={applyFilter}>
        <label>
          <span>Documento ID</span>
          <input value={draftDocumentoId} onChange={(event) => setDraftDocumentoId(event.target.value)} />
        </label>
        <div className="filter-actions">
          <button type="submit">Carregar</button>
          {documentoId ? (
            <Link className="button" to={`/documentos/${encodeURIComponent(documentoId)}`}>
              Ver Documento
            </Link>
          ) : null}
        </div>
      </form>

      <ErrorAlert error={parseError ?? query.error ?? saveMutation.error} />

      {!documentoId ? <p>Informe um Documento ID para listar Falhas de Extração.</p> : null}
      {query.isLoading ? <p>Carregando Falhas de Extração...</p> : null}
      {documentoId && query.data ? (
        <div className="grid-two">
          <section className="card">
            <h3>Lista</h3>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>#</th>
                    <th>Código</th>
                    <th>Detalhe</th>
                    <th>Candidato</th>
                    <th>Ações</th>
                  </tr>
                </thead>
                <tbody>
                  {falhas.length === 0 ? (
                    <tr>
                      <td colSpan={5}>Nenhuma Falha de Extração encontrada.</td>
                    </tr>
                  ) : (
                    falhas.map((falha, index) => (
                      <tr key={index}>
                        <td>{index + 1}</td>
                        <td>
                          <ValueView value={falha.codigo ?? falha.code} />
                        </td>
                        <td>
                          <ValueView value={falha.detalhe ?? falha.detail ?? falha.message} />
                        </td>
                        <td>
                          <ValueView value={falha.candidato ?? falha.candidate} />
                        </td>
                        <td className="actions">
                          <button type="button" onClick={() => removeFalha(index)}>
                            Remover
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <section className="card">
            <h3>Criar / editar coleção</h3>
            <p>Edite o array JSON e salve para enviar PUT na coleção do Documento.</p>
            <form className="entity-form" onSubmit={saveEditor}>
              <textarea value={editor} onChange={(event) => setEditor(event.target.value)} rows={18} />
              <div className="form-actions">
                <button type="button" onClick={appendFalha}>
                  Adicionar Falha
                </button>
                <button type="submit" className="primary" disabled={saveMutation.isPending}>
                  Salvar Falhas de Extração
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}
    </section>
  );
}
