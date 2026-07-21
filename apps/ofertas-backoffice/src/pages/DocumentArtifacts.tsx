import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { artifactDownloadUrl, deleteArtifact, normalizeArtifacts, request, uploadArtifact } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { ValueView } from "../components/ValueView";

type DocumentArtifactsProps = {
  documentoId: string;
};

export function DocumentArtifacts({ documentoId }: DocumentArtifactsProps) {
  const queryClient = useQueryClient();
  const [tentativa, setTentativa] = React.useState("");
  const [filepath, setFilepath] = React.useState("");
  const [file, setFile] = React.useState<File | null>(null);
  const query = useQuery({
    queryKey: ["document-artifacts", documentoId],
    queryFn: () => request<unknown>(`/documentos/${encodeURIComponent(documentoId)}/artefatos`),
  });
  const uploadMutation = useMutation({
    mutationFn: () => {
      if (!file) {
        throw new Error("Selecione um ficheiro para upload.");
      }

      return uploadArtifact(documentoId, tentativa, filepath, file);
    },
    onSuccess: () => {
      setFile(null);
      queryClient.invalidateQueries({ queryKey: ["document-artifacts", documentoId] });
    },
  });
  const deleteMutation = useMutation({
    mutationFn: (item: { tentativa: string; filepath: string }) => deleteArtifact(documentoId, item.tentativa, item.filepath),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["document-artifacts", documentoId] }),
  });
  const artifacts = normalizeArtifacts(query.data);

  function submitUpload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    uploadMutation.mutate();
  }

  function onFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    setFile(event.target.files?.[0] ?? null);
  }

  function removeItem(item: { tentativa: string; filepath: string }) {
    if (window.confirm(`Apagar Artefato ${item.filepath} da tentativa ${item.tentativa}?`)) {
      deleteMutation.mutate(item);
    }
  }

  return (
    <section className="card">
      <header className="section-header">
        <div>
          <h3>Artefatos</h3>
          <p>Materiais retidos por tentativa de processamento do Documento.</p>
        </div>
      </header>

      <ErrorAlert error={query.error ?? uploadMutation.error ?? deleteMutation.error} />

      <form className="upload-form" onSubmit={submitUpload}>
        <label>
          <span>Tentativa</span>
          <input value={tentativa} onChange={(event) => setTentativa(event.target.value)} required />
        </label>
        <label>
          <span>Path do Artefato</span>
          <input value={filepath} onChange={(event) => setFilepath(event.target.value)} placeholder="raw/extrator.json" required />
        </label>
        <label>
          <span>Ficheiro</span>
          <input type="file" onChange={onFileChange} required />
        </label>
        <button type="submit" disabled={uploadMutation.isPending}>
          Upload / substituir
        </button>
      </form>

      {query.isLoading ? <p>Carregando Artefatos...</p> : null}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Tentativa</th>
              <th>Artefato</th>
              <th>Tipo</th>
              <th>Tamanho</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            {artifacts.length === 0 ? (
              <tr>
                <td colSpan={5}>Nenhum Artefato listado pela API.</td>
              </tr>
            ) : (
              artifacts.map((item) => (
                <tr key={`${item.tentativa}:${item.filepath}`}>
                  <td>{item.tentativa || "-"}</td>
                  <td>
                    <ValueView value={item.filepath} />
                  </td>
                  <td>{item.contentType ?? "-"}</td>
                  <td>{item.size ?? "-"}</td>
                  <td className="actions">
                    {item.tentativa ? (
                      <a href={artifactDownloadUrl(documentoId, item.tentativa, item.filepath)} rel="noreferrer">
                        Download
                      </a>
                    ) : null}
                    {item.tentativa ? (
                      <button type="button" onClick={() => removeItem(item)} disabled={deleteMutation.isPending}>
                        Apagar
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {query.data ? (
        <details>
          <summary>Resposta bruta de Artefatos</summary>
          <pre className="json-preview">{JSON.stringify(query.data, null, 2)}</pre>
        </details>
      ) : null}
    </section>
  );
}
