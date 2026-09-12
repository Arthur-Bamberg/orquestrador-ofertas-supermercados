import { NavLink, Outlet, useNavigate, useOutletContext } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { sair } from "../api";

const navItems = [
  { to: "/mercados", label: "Mercados" },
  { to: "/fontes", label: "Fontes" },
  { to: "/documentos", label: "Documentos" },
  { to: "/ofertas", label: "Ofertas" },
  { to: "/produtos", label: "Produtos" },
  { to: "/marcas", label: "Marcas" },
  { to: "/falhas-extracao", label: "Falhas de Extração" },
  { to: "/usos-extrator", label: "Uso do Extrator" },
  { to: "/operacoes", label: "Operações" },
  { to: "/canal", label: "Canal" },
  { to: "/conversas", label: "Conversas" },
];

export function Layout() {
  const { nome } = useOutletContext<{ nome: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function onSair() {
    await sair();
    queryClient.clear();
    navigate("/entrar", { replace: true });
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div>
          <h1>Ofertas</h1>
          <p>Backoffice local</p>
        </div>
        <nav aria-label="Navegação principal">
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} className={({ isActive }) => (isActive ? "active" : undefined)}>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-operador">
          <p>{nome}</p>
          <button type="button" onClick={() => void onSair()}>
            Sair
          </button>
        </div>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
