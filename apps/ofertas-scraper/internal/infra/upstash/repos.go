package upstash

import store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"

type MercadoRepo = store.MercadoRepo
type FonteRepo = store.FonteRepo
type ProdutoRepo = store.ProdutoRepo
type MarcaRepo = store.MarcaRepo
type DocumentoRepo = store.DocumentoRepo
type OfertaRepo = store.OfertaRepo
type FalhaRepo = store.FalhaRepo
type UsoExtratorRepo = store.UsoExtratorRepo
type CotaRepo = store.CotaRepo

var NewMercadoRepo = store.NewMercadoRepo
var NewFonteRepo = store.NewFonteRepo
var NewProdutoRepo = store.NewProdutoRepo
var NewMarcaRepo = store.NewMarcaRepo
var NewDocumentoRepo = store.NewDocumentoRepo
var NewOfertaRepo = store.NewOfertaRepo
var NewFalhaRepo = store.NewFalhaRepo
var NewUsoExtratorRepo = store.NewUsoExtratorRepo
var NewCotaRepo = store.NewCotaRepo
