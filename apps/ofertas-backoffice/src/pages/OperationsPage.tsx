import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { normalizeList, request } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { FormattedValue } from "../components/ValueView";
import { RefSelect } from "../components/RefSelect";
import type { ColumnFormat, RefKind } from "../display";
import type { EntityRecord, OperationStatus } from "../domain";
import { useCatalogLookups } from "../useCatalogLookups";

function pendingStatus(status: OperationStatus | unknown): boolean {
  const normalized = String(status ?? "").toLowerCase();
  return ["pendente", "pending", "queued", "agendado"].includes(normalized);
}

function operationId(operation: EntityRecord): string {
  return String(operation.id ?? operation.operacaoId ?? operation.operationId ?? "");
}

type OperationColumn = {
  name: string;
  label: string;
  aliases?: string[];
  format?: ColumnFormat;
  ref?: RefKind;
};

const operationColumns: OperationColumn[] = [
  { name: "id", label: "ID" },
  { name: "kind", label: "Tipo", aliases: ["tipo", "type"] },
  { name: "status", label: "Status", aliases: ["estado"] },
  { name: "fonteId", label: "Fonte", format: "ref", ref: "fonte" },
  { name: "documentoId", label: "Documento", format: "ref", ref: "documento" },
  { name: "createdAt", label: "Criado", format: "datetime", aliases: ["criadoEm"] },
  { name: "updatedAt", label: "Atualizado", format: "datetime", aliases: ["atualizadoEm", "finishedAt"] },
];

function columnValue(record: EntityRecord, column: OperationColumn): unknown {
  if (record[column.name] !== undefined && record[column.name] !== null && record[column.name] !== "") {
    return record[column.name];
  }

  for (const alias of column.aliases ?? []) {
    if (record[alias] !== undefined && record[alias] !== null && record[alias] !== "") {
      return record[alias];
    }
  }

  return record[column.name];
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
  const { lookups, isLoading: catalogLoading } = useCatalogLookups();
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
              <span>Fonte</span>
              {catalogLoading ? (
                <select disabled>
                  <option>Carregando...</option>
                </select>
              ) : (
                <RefSelect
                  kind="fonte"
                  lookups={lookups}
                  required
                  allowEmpty
                  emptyLabel="Selecione"
                  value={fonteId}
                  onChange={setFonteId}
                />
              )}
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
              <span>Documento</span>
              {catalogLoading ? (
                <select disabled>
                  <option>Carregando...</option>
                </select>
              ) : (
                <RefSelect
                  kind="documento"
                  lookups={lookups}
                  required
                  allowEmpty
                  emptyLabel="Selecione"
                  value={documentoId}
                  onChange={setDocumentoId}
                />
              )}
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
          <OperationTable
            title="Fila"
            operations={sections.queue}
            lookups={lookups}
            lookupsLoading={catalogLoading}
            onCancel={cancelOperation}
            isCancelling={cancelMutation.isPending}
          />
          <OperationTable
            title="Histórico"
            operations={sections.history}
            lookups={lookups}
            lookupsLoading={catalogLoading}
            onCancel={cancelOperation}
            isCancelling={cancelMutation.isPending}
          />
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
  lookups: ReturnType<typeof useCatalogLookups>["lookups"];
  lookupsLoading: boolean;
  isCancelling: boolean;
  onCancel: (operation: EntityRecord) => void;
};

function OperationTable({ title, operations, lookups, lookupsLoading, isCancelling, onCancel }: OperationTableProps) {
  return (
    <section className="card">
      <h3>{title}</h3>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {operationColumns.map((column) => (
                <th key={column.name}>{column.label}</th>
              ))}
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            {operations.length === 0 ? (
              <tr>
                <td colSpan={operationColumns.length + 1}>Nenhuma Operação de Pipeline.</td>
              </tr>
            ) : (
              operations.map((operation, index) => {
                const id = operationId(operation);
                const canCancel = id && pendingStatus(operation.status ?? operation.estado);

                return (
                  <tr key={id || index}>
                    {operationColumns.map((column) => (
                      <td key={column.name}>
                        <FormattedValue
                          value={columnValue(operation, column)}
                          format={column.format}
                          refKind={column.ref}
                          lookups={lookups}
                          lookupsLoading={lookupsLoading}
                        />
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
