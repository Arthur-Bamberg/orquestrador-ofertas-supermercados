import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { normalizeList, request } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { ValueView } from "../components/ValueView";
import type { EntityRecord, OperationStatus } from "../domain";

function pendingStatus(status: OperationStatus | unknown): boolean {
  const normalized = String(status ?? "").toLowerCase();
  return ["pendente", "pending", "queued", "agendado"].includes(normalized);
}

function operationId(operation: EntityRecord): string {
  return String(operation.id ?? operation.operacaoId ?? operation.operationId ?? "");
}

function operationSections(payload: unknown): { queue: EntityRecord[]; history: EntityRecord[]; all: EntityRecord[] } {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    const all = normalizeList(payload);
    return { queue: all.filter((item) => pendingStatus(item.status ?? item.estado)), history: all, all };
  }

  const record = payload as Record<string, unknown>;
  const hasExplicitSections = "queue" in record || "fila" in record || "history" in record || "historico" in record;
  const queue = normalizeList(record.queue ?? record.fila ?? []);
  const history = normalizeList(record.history ?? record.historico ?? record.operacoes ?? []);
  const all = hasExplicitSections ? history : normalizeList(payload);

  return { queue, history: all, all };
}

export function OperationsPage() {
  const [searchParams] = useSearchParams();
  const [fonteId, setFonteId] = React.useState("");
  const [documentoId, setDocumentoId] = React.useState(searchParams.get("documentoId") ?? "");
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["ops"],
    queryFn: () => request<unknown>("/ops"),
    refetchInterval: 5000,
  });
  const runMutation = useMutation({
    mutationFn: () => request<unknown>("/ops/run", { method: "POST", body: {} }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ops"] }),
  });
  const discoverMutation = useMutation({
    mutationFn: () => request<unknown>("/ops/discover", { method: "POST", body: { fonteId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ops"] }),
  });
  const reprocessMutation = useMutation({
    mutationFn: () => request<unknown>("/ops/reprocess", { method: "POST", body: { documentoId } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ops"] }),
  });
  const cancelMutation = useMutation({
    mutationFn: (id: string) => request<unknown>(`/ops/${encodeURIComponent(id)}/cancel`, { method: "POST", body: {} }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ops"] }),
  });
  const sections = operationSections(query.data);

  function submitDiscover(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    discoverMutation.mutate();
  }

  function submitReprocess(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    reprocessMutation.mutate();
  }

  function cancelOperation(operation: EntityRecord) {
    const id = operationId(operation);

    if (id && window.confirm(`Cancelar Operação de Pipeline ${id}?`)) {
      cancelMutation.mutate(id);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>Operações de Pipeline</h2>
          <p>Dispare trabalho do scraper e acompanhe fila serial e histórico.</p>
        </div>
      </header>

      <ErrorAlert error={query.error ?? runMutation.error ?? discoverMutation.error ?? reprocessMutation.error ?? cancelMutation.error} />

      <section className="operation-actions">
        <article className="card">
          <h3>Run diário</h3>
          <p>Executa o fluxo diário completo.</p>
          <button type="button" className="primary" onClick={() => runMutation.mutate()} disabled={runMutation.isPending}>
            Run diário
          </button>
        </article>

        <article className="card">
          <h3>Discover Fonte</h3>
          <p>Descobre novos Documentos de uma Fonte sem processá-los.</p>
          <form className="entity-form compact" onSubmit={submitDiscover}>
            <label>
              <span>Fonte ID</span>
              <input value={fonteId} onChange={(event) => setFonteId(event.target.value)} required />
            </label>
            <button type="submit" disabled={discoverMutation.isPending}>
              Discover Fonte
            </button>
          </form>
        </article>

        <article className="card">
          <h3>Reprocess Documento</h3>
          <p>Agenda nova tentativa de processamento para um Documento.</p>
          <form className="entity-form compact" onSubmit={submitReprocess}>
            <label>
              <span>Documento ID</span>
              <input value={documentoId} onChange={(event) => setDocumentoId(event.target.value)} required />
            </label>
            <button type="submit" disabled={reprocessMutation.isPending}>
              Reprocess Documento
            </button>
          </form>
        </article>
      </section>

      {query.isLoading ? <p>Carregando Operações de Pipeline...</p> : null}
      {query.data ? (
        <>
          <OperationTable title="Fila" operations={sections.queue} onCancel={cancelOperation} isCancelling={cancelMutation.isPending} />
          <OperationTable title="Histórico" operations={sections.history} onCancel={cancelOperation} isCancelling={cancelMutation.isPending} />
          <details>
            <summary>Resposta bruta de Operações</summary>
            <pre className="json-preview">{JSON.stringify(query.data, null, 2)}</pre>
          </details>
        </>
      ) : null}
    </section>
  );
}

type OperationTableProps = {
  title: string;
  operations: EntityRecord[];
  isCancelling: boolean;
  onCancel: (operation: EntityRecord) => void;
};

function OperationTable({ title, operations, isCancelling, onCancel }: OperationTableProps) {
  const columns = ["id", "tipo", "type", "status", "estado", "fonteId", "documentoId", "criadoEm", "createdAt", "atualizadoEm", "updatedAt"];

  return (
    <section className="card">
      <h3>{title}</h3>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {columns.map((column) => (
                <th key={column}>{column}</th>
              ))}
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            {operations.length === 0 ? (
              <tr>
                <td colSpan={columns.length + 1}>Nenhuma Operação de Pipeline.</td>
              </tr>
            ) : (
              operations.map((operation, index) => {
                const id = operationId(operation);
                const canCancel = id && pendingStatus(operation.status ?? operation.estado);

                return (
                  <tr key={id || index}>
                    {columns.map((column) => (
                      <td key={column}>
                        <ValueView value={operation[column]} />
                      </td>
                    ))}
                    <td className="actions">
                      {canCancel ? (
                        <button type="button" onClick={() => onCancel(operation)} disabled={isCancelling}>
                          Cancelar pendente
                        </button>
                      ) : (
                        "-"
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
