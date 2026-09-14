import { useQuery } from "@tanstack/react-query";
import { ErrorAlert } from "../components/ErrorAlert";
import { getCanal } from "../gateway";

const POLL_MS = 5000;

const estadoRotulo: Record<string, string> = {
  pronto: "Pronto — Cloud API",
  nao_configurado: "Não configurado",
};

export function CanalPage() {
  const query = useQuery({
    queryKey: ["canal"],
    queryFn: getCanal,
    refetchInterval: POLL_MS,
  });

  const sit = query.data;

  return (
    <section className="page narrow">
      <header className="page-header">
        <div>
          <h2>Canal</h2>
          <p>WhatsApp Cloud API deste processo — um Canal, não um Mercado.</p>
        </div>
      </header>

      <ErrorAlert error={query.error} />
      {query.isLoading ? <p>Carregando Canal...</p> : null}

      {sit ? (
        <section className="card">
          <dl className="detail-list">
            <div>
              <dt>Estado</dt>
              <dd>
                <span className={`canal-estado canal-estado-${sit.estado}`}>{estadoRotulo[sit.estado] ?? sit.estado}</span>
              </dd>
            </div>
            {sit.jid ? (
              <div>
                <dt>Número do Canal</dt>
                <dd>
                  <code>{sit.jid}</code>
                </dd>
              </div>
            ) : null}
          </dl>
        </section>
      ) : null}
    </section>
  );
}
