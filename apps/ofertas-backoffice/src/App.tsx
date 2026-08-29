import { Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { FalhasExtracaoPage } from "./pages/FalhasExtracaoPage";
import { OperationsPage } from "./pages/OperationsPage";
import { ResourceDetailPage, ResourceFormPage, ResourceListPage } from "./pages/ResourcePages";
import { UsoExtratorDetailPage, UsoExtratorFormPage, UsoExtratorListPage } from "./pages/UsoExtratorPages";

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Navigate to="/mercados" replace />} />
        <Route path=":resourceSlug" element={<ResourceListPage />} />
        <Route path=":resourceSlug/novo" element={<ResourceFormPage mode="create" />} />
        <Route path=":resourceSlug/:id" element={<ResourceDetailPage />} />
        <Route path=":resourceSlug/:id/editar" element={<ResourceFormPage mode="edit" />} />
        <Route path="falhas-extracao" element={<FalhasExtracaoPage />} />
        <Route path="usos-extrator" element={<UsoExtratorListPage />} />
        <Route path="usos-extrator/novo" element={<UsoExtratorFormPage mode="create" />} />
        <Route path="usos-extrator/:documentoId/:tentativa" element={<UsoExtratorDetailPage />} />
        <Route path="usos-extrator/:documentoId/:tentativa/editar" element={<UsoExtratorFormPage mode="edit" />} />
        <Route path="operacoes" element={<OperationsPage />} />
      </Route>
    </Routes>
  );
}
