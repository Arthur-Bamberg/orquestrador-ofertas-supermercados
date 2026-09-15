import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorAlert } from "../components/ErrorAlert";
import { desparearCanal, getCanal } from "../gateway";

const POLL_MS = 2000;

const estadoRotulo: Record<string, string> = {
  pendente: "Pendente — aponte o celular no QR",
  conectado: "Conectado",
  desconectado: "Desconectado — sessão existe, sem socket",
};

export function CanalPage() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["canal"],
    queryFn: getCanal,
    refetchInterval: POLL_MS,
  });
  const desparear = useMutation({
    mutationFn: desparearCanal,
    onSuccess: (sit) => {
      queryClient.setQueryData(["canal"], sit);
    },
  });

  const sit = query.data;
  const podeDesparear = sit?.estado === "conectado" || sit?.estado === "desconectado";

  function confirmarDesparear() {
    if (
      !window.confirm(
        "Desparear o Canal? O telefone deixa de estar vinculado e um QR novo aparece. O rastro de Conversas permanece.",
      )
    ) {
      return;
    }
    desparear.mutate();
  }

  return (
    <section className="page narrow">
      <header className="page-header">
        <div>
          <h2>Canal</h2>
          <p>Pareamento WhatsApp deste processo — um Canal, não um Mercado.</p>
        </div>
      </header>

      <ErrorAlert error={query.error ?? desparear.error} />
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
                <dt>Telefone dedicado</dt>
                <dd>
                  <code>{sit.jid}</code>
                </dd>
              </div>
            ) : null}
          </dl>

          {sit.estado === "pendente" ? (
            sit.qrPngBase64 ? (
              <figure className="canal-qr">
                <img
                  alt="QR de Pareamento do Canal"
                  src={`data:image/png;base64,${sit.qrPngBase64}`}
                  width={280}
                  height={280}
                />
                <figcaption>WhatsApp → Aparelhos conectados → Vincular dispositivo</figcaption>
              </figure>
            ) : (
              <p>Aguardando QR...</p>
            )
          ) : null}

          {podeDesparear ? (
            <div className="form-actions">
              <button type="button" className="danger" disabled={desparear.isPending} onClick={confirmarDesparear}>
                {desparear.isPending ? "Despareando..." : "Desparear"}
              </button>
            </div>
          ) : null}
        </section>
      ) : null}
    </section>
  );
}
