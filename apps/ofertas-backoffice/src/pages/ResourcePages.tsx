import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, Navigate, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { create, get, list, remove, update } from "../api";
import { AtivaToggle } from "../components/AtivaToggle";
import { ErrorAlert } from "../components/ErrorAlert";
import { FormField, fieldValueToString, parseFieldValue } from "../components/FormField";
import { FormattedValue, formatValue } from "../components/ValueView";
import { RefSelect } from "../components/RefSelect";
import type { ColumnFormat } from "../display";
import { isAtiva } from "../display";
import type { EntityField, EntityRecord, FilterField, ResourceConfig } from "../domain";
import { getResource, resources } from "../resources";
import { useCatalogLookups } from "../useCatalogLookups";
import { DocumentArtifacts } from "./DocumentArtifacts";

function useResourceConfig(): ResourceConfig | undefined {
  const { resourceSlug } = useParams();
  return getResource(resourceSlug);
}

function entityId(entity: EntityRecord, config: ResourceConfig): string {
  const key = config.idField ?? "id";
  return String(entity[key] ?? "");
}

function formatFromField(field: EntityField): ColumnFormat | undefined {
  switch (field.kind) {
    case "date":
      return "date";
    case "datetime":
      return "datetime";
    case "decimal":
      return "currency";
    case "ref":
      return "ref";
    case "boolean":
      return "boolean";
    default:
      return undefined;
  }
}

function FilterControl({
  filter,
  defaultValue,
  lookups,
  isLoading,
}: {
  filter: FilterField;
  defaultValue: string;
  lookups: ReturnType<typeof useCatalogLookups>["lookups"];
  isLoading: boolean;
}) {
  if (filter.kind === "ref" && filter.ref) {
    if (isLoading) {
      return (
        <select name={filter.name} defaultValue={defaultValue} disabled>
          <option value={defaultValue}>{defaultValue || "Carregando..."}</option>
        </select>
      );
    }

    return (
      <RefSelect name={filter.name} kind={filter.ref} lookups={lookups} defaultValue={defaultValue} emptyLabel="Todos" />
    );
  }

  if (filter.kind === "select") {
    return (
      <select name={filter.name} defaultValue={defaultValue}>
        <option value="">Todos</option>
        {(filter.options ?? []).map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  }

  return <input name={filter.name} type={filter.kind} defaultValue={defaultValue} />;
}

function activeFilters(config: ResourceConfig, searchParams: URLSearchParams): Record<string, string> {
  return Object.fromEntries(
    (config.filters ?? []).map((filter) => [filter.name, searchParams.get(filter.name) ?? ""]),
  );
}

export function ResourceListPage() {
  const config = useResourceConfig();
  const resource = config ?? resources[0];
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const filters = activeFilters(resource, searchParams);
  const { lookups, isLoading: catalogLoading } = useCatalogLookups();
  const query = useQuery({
    queryKey: ["resource-list", resource.slug, filters],
    queryFn: () => list(resource.apiPath, filters),
    enabled: Boolean(config),
  });
  const deleteMutation = useMutation({
    mutationFn: (id: string) => remove(resource.apiPath, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resource-list", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["catalog"] });
    },
  });
  const toggleAtivaMutation = useMutation({
    mutationFn: ({ id, entity }: { id: string; entity: EntityRecord }) =>
      update(resource.apiPath, id, { ...entity, ativa: !isAtiva(entity.ativa) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resource-list", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["resource-detail", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["catalog"] });
    },
  });

  function submitFilters(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const nextParams = new URLSearchParams();

    (resource.filters ?? []).forEach((filter) => {
      const value = String(formData.get(filter.name) ?? "");

      if (value.trim() !== "") {
        nextParams.set(filter.name, value);
      }
    });

    setSearchParams(nextParams);
  }

  function deleteEntity(id: string) {
    if (window.confirm(`Apagar ${resource.singular} ${id}?`)) {
      deleteMutation.mutate(id);
    }
  }

  if (!config) {
    return <Navigate to="/mercados" replace />;
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>{config.plural}</h2>
          <p>{config.description}</p>
        </div>
        <Link className="button primary" to="novo">
          Novo {config.singular}
        </Link>
      </header>

      {config.filters && config.filters.length > 0 ? (
        <form className="filters" onSubmit={submitFilters}>
          {config.filters.map((filter) => (
            <label key={filter.name}>
              <span>{filter.label}</span>
              <FilterControl
                filter={filter}
                defaultValue={filters[filter.name] ?? ""}
                lookups={lookups}
                isLoading={catalogLoading}
              />
            </label>
          ))}
          <div className="filter-actions">
            <button type="submit">Filtrar</button>
            <button type="button" onClick={() => setSearchParams(new URLSearchParams())}>
              Limpar
            </button>
          </div>
        </form>
      ) : null}

      <ErrorAlert error={query.error ?? deleteMutation.error ?? toggleAtivaMutation.error} />

      {query.isLoading ? <p>Carregando {config.plural}...</p> : null}
      {query.data ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                {config.columns.map((column) => (
                  <th key={column.name}>{column.label}</th>
                ))}
                <th>Ações</th>
              </tr>
            </thead>
            <tbody>
              {query.data.length === 0 ? (
                <tr>
                  <td colSpan={config.columns.length + 1}>Nenhum registro encontrado.</td>
                </tr>
              ) : (
                query.data.map((entity) => {
                  const id = entityId(entity, config);
                  const ativa = isAtiva(entity.ativa);

                  return (
                    <tr key={id || JSON.stringify(entity)}>
                      {config.columns.map((column) => (
                        <td key={column.name}>
                          {column.name === (config.idField ?? "id") && id ? (
                            <Link to={encodeURIComponent(id)}>{id}</Link>
                          ) : column.name === "ativa" && config.slug === "fontes" && id ? (
                            <AtivaToggle
                              id={`ativa-${id}`}
                              ativa={ativa}
                              disabled={toggleAtivaMutation.isPending}
                              onChange={() => toggleAtivaMutation.mutate({ id, entity })}
                            />
                          ) : (
                            <FormattedValue
                              value={entity[column.name]}
                              format={column.format}
                              refKind={column.ref}
                              lookups={lookups}
                              lookupsLoading={catalogLoading}
                            />
                          )}
                        </td>
                      ))}
                      <td className="actions">
                        {id ? (
                          <>
                            <Link to={encodeURIComponent(id)}>Detalhe</Link>
                            <Link to={`${encodeURIComponent(id)}/editar`}>Editar</Link>
                            {config.slug === "fontes" && entity.mercadoId ? (
                              <Link to={`/operacoes?mercadoId=${encodeURIComponent(String(entity.mercadoId))}`}>
                                Testar
                              </Link>
                            ) : null}
                            <button type="button" onClick={() => deleteEntity(id)} disabled={deleteMutation.isPending}>
                              Apagar
                            </button>
                          </>
                        ) : (
                          "Sem ID"
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

export function ResourceDetailPage() {
  const config = useResourceConfig();
  const resource = config ?? resources[0];
  const { id = "" } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { lookups } = useCatalogLookups();

  const decodedId = decodeURIComponent(id);
  const query = useQuery({
    queryKey: ["resource-detail", resource.slug, decodedId],
    queryFn: () => get(resource.apiPath, decodedId),
    enabled: Boolean(config),
  });
  const deleteMutation = useMutation({
    mutationFn: () => remove(resource.apiPath, decodedId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resource-list", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["catalog"] });
      navigate(`/${resource.slug}`);
    },
  });
  const toggleAtivaMutation = useMutation({
    mutationFn: (entity: EntityRecord) =>
      update(resource.apiPath, decodedId, { ...entity, ativa: !isAtiva(entity.ativa) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resource-list", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["resource-detail", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["catalog"] });
    },
  });

  function deleteEntity() {
    if (window.confirm(`Apagar ${resource.singular} ${decodedId}?`)) {
      deleteMutation.mutate();
    }
  }

  if (!config) {
    return <Navigate to="/mercados" replace />;
  }

  const ativa = isAtiva(query.data?.ativa);

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>
            {config.singular} {decodedId}
          </h2>
          <p>Detalhe do registro.</p>
        </div>
        <div className="header-actions">
          <Link className="button" to={`/${config.slug}`}>
            Voltar
          </Link>
          {config.slug === "fontes" && query.data?.mercadoId ? (
            <Link className="button" to={`/operacoes?mercadoId=${encodeURIComponent(String(query.data.mercadoId))}`}>
              Testar
            </Link>
          ) : null}
          <Link className="button" to="editar">
            Editar
          </Link>
          <button type="button" className="danger" onClick={deleteEntity} disabled={deleteMutation.isPending}>
            Apagar
          </button>
        </div>
      </header>

      <ErrorAlert error={query.error ?? deleteMutation.error ?? toggleAtivaMutation.error} />
      {query.isLoading ? <p>Carregando detalhe...</p> : null}
      {query.data ? (
        <>
          <dl className="detail-list">
            {config.fields.map((field) => (
              <div key={field.name}>
                <dt>{field.label}</dt>
                <dd>
                  {field.name === "ativa" && config.slug === "fontes" ? (
                    <AtivaToggle
                      id={`ativa-field-${decodedId}`}
                      ativa={ativa}
                      disabled={toggleAtivaMutation.isPending}
                      onChange={() => toggleAtivaMutation.mutate(query.data)}
                    />
                  ) : (
                    <FormattedValue
                      value={query.data?.[field.name]}
                      format={formatFromField(field)}
                      refKind={field.ref}
                      lookups={lookups}
                    />
                  )}
                </dd>
              </div>
            ))}
            {Object.entries(query.data)
              .filter(([key]) => !config.fields.some((field) => field.name === key))
              .map(([key, value]) => (
                <div key={key}>
                  <dt>{key}</dt>
                  <dd>
                    <FormattedValue value={value} lookups={lookups} />
                  </dd>
                </div>
              ))}
          </dl>
          {config.slug === "documentos" ? <DocumentDetailLinks documentoId={decodedId} /> : null}
          {config.slug === "documentos" ? <DocumentArtifacts documentoId={decodedId} /> : null}
        </>
      ) : null}
    </section>
  );
}

function DocumentDetailLinks({ documentoId }: { documentoId: string }) {
  const query = new URLSearchParams({ documentoId }).toString();

  return (
    <section className="card">
      <h3>Relações do Documento</h3>
      <div className="inline-actions">
        <Link className="button" to={`/falhas-extracao?${query}`}>
          Ver Falhas de Extração
        </Link>
        <Link className="button" to={`/usos-extrator?${query}`}>
          Ver Uso do Extrator
        </Link>
        <Link className="button" to={`/operacoes?documentoId=${encodeURIComponent(documentoId)}`}>
          Reprocessar em Operações
        </Link>
      </div>
    </section>
  );
}

export function ResourceFormPage({ mode }: { mode: "create" | "edit" }) {
  const config = useResourceConfig();
  const resource = config ?? resources[0];
  const { id = "" } = useParams();
  const decodedId = decodeURIComponent(id);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { lookups } = useCatalogLookups();

  const detailQuery = useQuery({
    queryKey: ["resource-detail", resource.slug, decodedId],
    queryFn: () => get(resource.apiPath, decodedId),
    enabled: mode === "edit" && Boolean(config),
  });
  const mutation = useMutation({
    mutationFn: (payload: EntityRecord) =>
      mode === "create" ? create(resource.apiPath, payload) : update(resource.apiPath, decodedId, payload),
    onSuccess: (entity) => {
      queryClient.invalidateQueries({ queryKey: ["resource-list", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["resource-detail", resource.slug] });
      queryClient.invalidateQueries({ queryKey: ["catalog"] });
      const nextId = entityId(entity, resource) || decodedId;
      navigate(nextId ? `/${resource.slug}/${encodeURIComponent(nextId)}` : `/${resource.slug}`);
    },
  });
  const [values, setValues] = React.useState<Record<string, string>>({});
  const [parseError, setParseError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    if (mode === "create") {
      setValues(
        Object.fromEntries(
          resource.fields.map((field) => [field.name, field.kind === "boolean" ? "true" : ""]),
        ),
      );
      return;
    }

    if (detailQuery.data) {
      setValues(
        Object.fromEntries(
          resource.fields.map((field) => [field.name, fieldValueToString(detailQuery.data?.[field.name], field.kind)]),
        ),
      );
    }
  }, [detailQuery.data, mode, resource]);

  function changeField(name: string, value: string) {
    setValues((current) => ({ ...current, [name]: value }));
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setParseError(null);

    try {
      const payload = Object.fromEntries(
        resource.fields
          .map((field) => [field.name, parseFieldValue(field, values[field.name] ?? "")] as const)
          .filter(([, value]) => value !== undefined),
      );
      mutation.mutate(payload);
    } catch (error) {
      setParseError(error instanceof Error ? error : new Error("JSON inválido no formulário."));
    }
  }

  if (!config) {
    return <Navigate to="/mercados" replace />;
  }

  return (
    <section className="page narrow">
      <header className="page-header">
        <div>
          <h2>
            {mode === "create" ? "Novo" : "Editar"} {resource.singular}
          </h2>
          <p>{mode === "create" ? "Crie um registro administrativo." : `Atualize ${decodedId}.`}</p>
        </div>
        <Link className="button" to={mode === "create" ? `/${resource.slug}` : `/${resource.slug}/${encodeURIComponent(decodedId)}`}>
          Cancelar
        </Link>
      </header>

      <ErrorAlert error={parseError ?? detailQuery.error ?? mutation.error} />
      {detailQuery.isLoading ? <p>Carregando registro...</p> : null}
      {mode === "create" || detailQuery.data ? (
        <form className="entity-form" onSubmit={submit}>
          {resource.fields.map((field) => (
            <FormField
              key={field.name}
              field={field}
              value={values[field.name] ?? ""}
              onChange={changeField}
              lookups={lookups}
            />
          ))}
          <div className="form-actions">
            <button type="submit" className="primary" disabled={mutation.isPending}>
              {mutation.isPending ? "Salvando..." : "Salvar"}
            </button>
          </div>
        </form>
      ) : null}
    </section>
  );
}

export function ResourceRawPreview({ data }: { data: unknown }) {
  return <pre className="json-preview">{JSON.stringify(data, null, 2)}</pre>;
}

export function CompactValue({ value }: { value: unknown }) {
  return <span title={formatValue(value)}>{formatValue(value)}</span>;
}
