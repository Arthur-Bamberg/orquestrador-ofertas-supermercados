export function operadorIdentificado(payload: { nome?: string } | undefined): payload is { nome: string } {
  return typeof payload?.nome === "string" && payload.nome.trim() !== "";
}
