import { FormEvent, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { eu, identificar } from "../api";
import { ErrorAlert } from "../components/ErrorAlert";
import { operadorIdentificado } from "../sessao";

export function EntrarPage() {
  const navigate = useNavigate();
  const jaIdentificado = useQuery({ queryKey: ["eu"], queryFn: eu, retry: false });
  const [nome, setNome] = useState("");
  const [senha, setSenha] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [pending, setPending] = useState(false);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      await identificar(nome, senha);
      navigate("/mercados", { replace: true });
    } catch (err) {
      setError(err);
    } finally {
      setPending(false);
    }
  }

  if (operadorIdentificado(jaIdentificado.data)) {
    return <Navigate to="/mercados" replace />;
  }

  return (
    <div className="entrar-shell">
      <form className="card entrar-card" onSubmit={onSubmit}>
        <h1>Pague Menos Mercado</h1>
        <p>Identifique-se para administrar o catálogo e o Canal.</p>
        <ErrorAlert error={error} />
        <label>
          Nome
          <input autoComplete="username" value={nome} onChange={(e) => setNome(e.target.value)} required />
        </label>
        <label>
          Senha
          <input
            type="password"
            autoComplete="current-password"
            value={senha}
            onChange={(e) => setSenha(e.target.value)}
            required
          />
        </label>
        <button className="primary" type="submit" disabled={pending}>
          {pending ? "Entrando…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
