import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { ErrorAlert } from "../components/ErrorAlert";
import { formatDateTime } from "../display";
import {
  conversaRotulo,
  enviarTexto,
  getConversa,
  listConversas,
  listMensagens,
  midiaUrl,
  type Mensagem,
} from "../gateway";

const PAGE_SIZE = 200;
const tiposConversa = ["", "direta", "grupo", "status"];

export function ConversasListPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tipo = searchParams.get("tipo") ?? "";
  const q = searchParams.get("q") ?? "";
  const query = useQuery({
    queryKey: ["conversas", tipo, q],
    queryFn: () => listConversas({ tipo, q }),
  });

  function submitFilters(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const next = new URLSearchParams();
    const nextTipo = String(formData.get("tipo") ?? "").trim();
    const nextQ = String(formData.get("q") ?? "").trim();
    if (nextTipo) {
      next.set("tipo", nextTipo);
    }
    if (nextQ) {
      next.set("q", nextQ);
    }
    setSearchParams(next);
  }

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>Conversas</h2>
          <p>Rastro do canal WhatsApp: todas as Conversas persistidas, não só a allowlist.</p>
        </div>
      </header>

      <form className="filters" onSubmit={submitFilters}>
        <label>
          <span>Tipo</span>
          <select name="tipo" defaultValue={tipo}>
            {tiposConversa.map((option) => (
              <option key={option || "todos"} value={option}>
                {option === "" ? "Todos" : option}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Busca</span>
          <input name="q" type="text" defaultValue={q} placeholder="JID ou push name" />
        </label>
        <div className="filter-actions">
          <button type="submit">Filtrar</button>
          <button type="button" onClick={() => setSearchParams(new URLSearchParams())}>
            Limpar
          </button>
        </div>
      </form>

      <ErrorAlert error={query.error} />
      {query.isLoading ? <p>Carregando Conversas...</p> : null}
      {query.data ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Tipo</th>
                <th>Conversa</th>
                <th>Última Mensagem</th>
                <th>Quando</th>
                <th>Msgs</th>
                <th>Envio</th>
              </tr>
            </thead>
            <tbody>
              {query.data.length === 0 ? (
                <tr>
                  <td colSpan={6}>Nenhuma Conversa encontrada.</td>
                </tr>
              ) : (
                query.data.map((conversa) => (
                  <tr key={conversa.id}>
                    <td>{conversa.tipo}</td>
                    <td>
                      <Link to={`/conversas/${encodeURIComponent(conversa.id)}`}>{conversaRotulo(conversa)}</Link>
                    </td>
                    <td>{previewCorpo(conversa.ultimaMensagem)}</td>
                    <td>{formatDateTime(conversa.ultimaMensagem?.criadoEm)}</td>
                    <td>{conversa.totalMensagens}</td>
                    <td>{conversa.permitido ? "permitido" : "só rastro"}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

export function ConversaDetailPage() {
  const { id = "" } = useParams();
  const conversaId = decodeURIComponent(id);
  const queryClient = useQueryClient();
  const [anteriores, setAnteriores] = React.useState<Mensagem[]>([]);
  const [temMais, setTemMais] = React.useState(false);
  const [corpo, setCorpo] = React.useState("");

  const conversaQuery = useQuery({
    queryKey: ["conversa", conversaId],
    queryFn: () => getConversa(conversaId),
    enabled: conversaId !== "",
  });
  const mensagensQuery = useQuery({
    queryKey: ["conversa-mensagens", conversaId],
    queryFn: () => listMensagens(conversaId, { limit: PAGE_SIZE }),
    enabled: conversaId !== "",
  });

  React.useEffect(() => {
    setAnteriores([]);
    setTemMais((mensagensQuery.data?.length ?? 0) >= PAGE_SIZE);
  }, [conversaId, mensagensQuery.data]);

  const sendMutation = useMutation({
    mutationFn: (texto: string) => enviarTexto(conversaQuery.data?.jid ?? "", texto),
    onSuccess: () => {
      setCorpo("");
      queryClient.invalidateQueries({ queryKey: ["conversa", conversaId] });
      queryClient.invalidateQueries({ queryKey: ["conversa-mensagens", conversaId] });
      queryClient.invalidateQueries({ queryKey: ["conversas"] });
    },
  });

  async function carregarAnteriores() {
    const mensagens = [...anteriores, ...(mensagensQuery.data ?? [])];
    const oldest = mensagens[0];
    if (!oldest) {
      return;
    }
    const more = await listMensagens(conversaId, {
      limit: PAGE_SIZE,
      antesCriadoEm: oldest.criadoEm,
      antesId: oldest.id,
    });
    setAnteriores((current) => [...more, ...current]);
    setTemMais(more.length >= PAGE_SIZE);
  }

  function submitEnvio(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const texto = corpo.trim();
    if (texto === "") {
      return;
    }
    sendMutation.mutate(texto);
  }

  const conversa = conversaQuery.data;
  const mensagens = [...anteriores, ...(mensagensQuery.data ?? [])];

  return (
    <section className="page">
      <header className="page-header">
        <div>
          <h2>{conversa ? conversaRotulo(conversa) : "Conversa"}</h2>
          <p>
            {conversa ? `${conversa.tipo} · ${conversa.totalMensagens} Mensagens` : "Detalhe do rastro do canal."}
          </p>
        </div>
        <Link className="button" to="/conversas">
          Voltar
        </Link>
      </header>

      <ErrorAlert error={conversaQuery.error ?? mensagensQuery.error ?? sendMutation.error} />
      {conversaQuery.isLoading || mensagensQuery.isLoading ? <p>Carregando Conversa...</p> : null}

      {conversa ? (
        <>
          <dl className="detail-list">
            <div>
              <dt>JID</dt>
              <dd>{conversa.jid}</dd>
            </div>
            {conversa.jidLid ? (
              <div>
                <dt>LID</dt>
                <dd>{conversa.jidLid}</dd>
              </div>
            ) : null}
            <div>
              <dt>Tipo</dt>
              <dd>{conversa.tipo}</dd>
            </div>
            <div>
              <dt>Envio</dt>
              <dd>{conversa.permitido ? "allowlist — compositor habilitado" : "fora da allowlist (só rastro)"}</dd>
            </div>
          </dl>

          <section className="card thread">
            <div className="section-header">
              <h3>Mensagens</h3>
              {temMais ? (
                <button type="button" onClick={() => void carregarAnteriores()}>
                  Carregar anteriores
                </button>
              ) : null}
            </div>
            {mensagens.length === 0 ? <p>Nenhuma Mensagem nesta Conversa.</p> : null}
            <ol className="message-list">
              {mensagens.map((mensagem) => (
                <li key={mensagem.id} className={`message ${mensagem.direcao}`}>
                  <header>
                    <strong>{mensagem.pushName || mensagem.direcao}</strong>
                    <time dateTime={mensagem.criadoEm}>{formatDateTime(mensagem.criadoEm)}</time>
                  </header>
                  {mensagem.corpo ? <p>{mensagem.corpo}</p> : null}
                  {mensagem.midia ? <MensagemMidia mensagem={mensagem} /> : null}
                  <p className="message-meta">
                    {mensagem.tipo || "texto"}
                    {mensagem.origem ? ` · ${mensagem.origem}` : ""}
                    {mensagem.status ? ` · ${mensagem.status}` : ""}
                  </p>
                </li>
              ))}
            </ol>
          </section>

          {conversa.permitido ? (
            <form className="card composer" onSubmit={submitEnvio}>
              <label>
                <span>Enviar texto</span>
                <textarea value={corpo} onChange={(event) => setCorpo(event.target.value)} rows={3} />
              </label>
              <div className="form-actions">
                <button type="submit" className="primary" disabled={sendMutation.isPending || corpo.trim() === ""}>
                  {sendMutation.isPending ? "Enviando..." : "Enviar"}
                </button>
              </div>
            </form>
          ) : null}
        </>
      ) : null}
    </section>
  );
}

function previewCorpo(mensagem: Mensagem | undefined): string {
  if (!mensagem) {
    return "—";
  }
  if (mensagem.corpo.trim() !== "") {
    return mensagem.corpo;
  }
  if (mensagem.midia) {
    return `[${mensagem.midia.tipo}]`;
  }
  return mensagem.tipo || "—";
}

function MensagemMidia({ mensagem }: { mensagem: Mensagem }) {
  const url = midiaUrl(mensagem.id);
  const tipo = mensagem.midia?.tipo ?? "";
  if (tipo === "imagem" || tipo === "figurinha" || (mensagem.midia?.mime ?? "").startsWith("image/")) {
    return <img className="message-midia" src={url} alt={mensagem.midia?.filename || "Mídia"} />;
  }
  return (
    <a className="button" href={url} target="_blank" rel="noreferrer">
      Abrir {tipo || "mídia"}
    </a>
  );
}
