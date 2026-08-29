import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { list } from "./api";
import { buildCatalogLookups, type CatalogLookups } from "./display";

export function useCatalogLookups(): { lookups: CatalogLookups; isLoading: boolean } {
  const mercados = useQuery({ queryKey: ["catalog", "mercados"], queryFn: () => list("/mercados") });
  const produtos = useQuery({ queryKey: ["catalog", "produtos"], queryFn: () => list("/produtos") });
  const marcas = useQuery({ queryKey: ["catalog", "marcas"], queryFn: () => list("/marcas") });
  const fontes = useQuery({ queryKey: ["catalog", "fontes"], queryFn: () => list("/fontes") });
  const documentos = useQuery({ queryKey: ["catalog", "documentos"], queryFn: () => list("/documentos") });

  const lookups = useMemo(
    () =>
      buildCatalogLookups({
        mercados: mercados.data,
        produtos: produtos.data,
        marcas: marcas.data,
        fontes: fontes.data,
        documentos: documentos.data,
      }),
    [documentos.data, fontes.data, marcas.data, mercados.data, produtos.data],
  );

  return {
    lookups,
    isLoading:
      mercados.isPending || produtos.isPending || marcas.isPending || fontes.isPending || documentos.isPending,
  };
}
