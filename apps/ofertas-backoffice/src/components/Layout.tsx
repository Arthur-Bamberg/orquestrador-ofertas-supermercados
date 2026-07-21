import { NavLink, Outlet } from "react-router-dom";

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
];

export function Layout() {
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
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
