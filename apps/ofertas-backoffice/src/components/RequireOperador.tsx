import { useQuery } from "@tanstack/react-query";
import { Navigate, Outlet } from "react-router-dom";
import { eu } from "../api";

export function RequireOperador() {
  const query = useQuery({
    queryKey: ["eu"],
    queryFn: eu,
    retry: false,
  });

  if (query.isLoading) {
    return (
      <div className="entrar-shell">
        <p>Carregando…</p>
      </div>
    );
  }

  if (query.isError || !query.data) {
    return <Navigate to="/entrar" replace />;
  }

  return <Outlet context={query.data} />;
}
