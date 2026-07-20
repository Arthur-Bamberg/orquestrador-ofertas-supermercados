package application_test

import (
	"bytes"
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/filenamedate"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/raster"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type memFontes struct {
	items []domain.Fonte
}

func (m *memFontes) List(context.Context) ([]domain.Fonte, error) { return m.items, nil }
func (m *memFontes) Save(context.Context, domain.Fonte) error     { return nil }

type memDocs struct {
	mu   sync.Mutex
	byID map[string]domain.Documento
	idx  map[string]string
}

func newMemDocs() *memDocs {
	return &memDocs{byID: map[string]domain.Documento{}, idx: map[string]string{}}
}

func (m *memDocs) key(f domain.FonteID, filename, dia string) string {
	return string(f) + "|" + filename + "|" + dia
}

func (m *memDocs) GetByIdentity(_ context.Context, fonteID domain.FonteID, filename, dia string) (domain.Documento, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.idx[m.key(fonteID, filename, dia)]
	if !ok {
		return domain.Documento{}, false, nil
	}
	return m.byID[id], true, nil
}

func (m *memDocs) Save(_ context.Context, d domain.Documento) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[string(d.ID)] = d
	m.idx[m.key(d.FonteID, d.Filename, d.Dia)] = string(d.ID)
	return nil
}

func (m *memDocs) EarliestDia(_ context.Context, fonteID domain.FonteID, filename string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var earliest string
	prefix := string(fonteID) + "|" + filename + "|"
	for k := range m.idx {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			dia := k[len(prefix):]
			if earliest == "" || dia < earliest {
				earliest = dia
			}
		}
	}
	if earliest == "" {
		return "", false, nil
	}
	return earliest, true, nil
}

type memProdutos struct {
	byNorm map[string]domain.Produto
}

func (m *memProdutos) GetByNomeNorm(_ context.Context, n string) (domain.Produto, bool, error) {
	p, ok := m.byNorm[n]
	return p, ok, nil
}
func (m *memProdutos) Save(_ context.Context, p domain.Produto) error {
	if m.byNorm == nil {
		m.byNorm = map[string]domain.Produto{}
	}
	m.byNorm[p.NomeNorm] = p
	return nil
}

type memMarcas struct {
	byNorm map[string]domain.Marca
}

func (m *memMarcas) GetByNomeNorm(_ context.Context, n string) (domain.Marca, bool, error) {
	x, ok := m.byNorm[n]
	return x, ok, nil
}
func (m *memMarcas) Save(_ context.Context, x domain.Marca) error {
	if m.byNorm == nil {
		m.byNorm = map[string]domain.Marca{}
	}
	m.byNorm[x.NomeNorm] = x
	return nil
}

type memOfertas struct {
	byDoc     map[domain.DocumentoID][]domain.Oferta
	byProduto map[domain.ProdutoID]map[domain.DocumentoID]struct{}
}

func (m *memOfertas) SaveAll(_ context.Context, id domain.DocumentoID, ofertas []domain.Oferta) error {
	if m.byDoc == nil {
		m.byDoc = map[domain.DocumentoID][]domain.Oferta{}
	}
	if m.byProduto == nil {
		m.byProduto = map[domain.ProdutoID]map[domain.DocumentoID]struct{}{}
	}
	prev := produtoIDsFromOfertas(m.byDoc[id])
	next := produtoIDsFromOfertas(ofertas)
	for pid := range prev {
		if _, ok := next[pid]; ok {
			continue
		}
		delete(m.byProduto[pid], id)
		if len(m.byProduto[pid]) == 0 {
			delete(m.byProduto, pid)
		}
	}
	for pid := range next {
		if m.byProduto[pid] == nil {
			m.byProduto[pid] = map[domain.DocumentoID]struct{}{}
		}
		m.byProduto[pid][id] = struct{}{}
	}
	m.byDoc[id] = ofertas
	return nil
}
func (m *memOfertas) ListByDocumento(_ context.Context, id domain.DocumentoID) ([]domain.Oferta, error) {
	return m.byDoc[id], nil
}
func (m *memOfertas) ListDocumentoIDsByProduto(_ context.Context, produtoID domain.ProdutoID) ([]domain.DocumentoID, error) {
	set := m.byProduto[produtoID]
	ids := make([]domain.DocumentoID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids, nil
}

func produtoIDsFromOfertas(ofertas []domain.Oferta) map[domain.ProdutoID]struct{} {
	out := make(map[domain.ProdutoID]struct{})
	for _, o := range ofertas {
		if o.ProdutoID == "" {
			continue
		}
		out[o.ProdutoID] = struct{}{}
	}
	return out
}

type memFalhas struct {
	byDoc map[domain.DocumentoID][]domain.FalhaExtracao
}

func (m *memFalhas) SaveAll(_ context.Context, id domain.DocumentoID, falhas []domain.FalhaExtracao) error {
	if m.byDoc == nil {
		m.byDoc = map[domain.DocumentoID][]domain.FalhaExtracao{}
	}
	m.byDoc[id] = falhas
	return nil
}

type memFonteHTTP struct {
	pdfs map[string][]domain.PDFDescoberto
	body []byte
}

func (m *memFonteHTTP) DiscoverPDFs(_ context.Context, f domain.Fonte) ([]domain.PDFDescoberto, error) {
	return m.pdfs[string(f.ID)], nil
}
func (m *memFonteHTTP) DownloadPDF(context.Context, string) ([]byte, error) {
	return m.body, nil
}

type memArtefatos struct{}

func (memArtefatos) AttemptPath(doc domain.Documento, tentativa string) string {
	return "artefatos/" + string(doc.FonteID) + "/" + doc.Dia + "/" + doc.Filename + "/" + tentativa
}
func (memArtefatos) SavePDF(context.Context, domain.Documento, string, []byte) error { return nil }
func (memArtefatos) SaveImages(context.Context, domain.Documento, string, []domain.PageImage) error {
	return nil
}
func (memArtefatos) SaveRawExtrator(context.Context, domain.Documento, string, []byte) error {
	return nil
}
func (memArtefatos) SaveUsoExtrator(context.Context, domain.Documento, string, domain.UsoExtrator) error {
	return nil
}
func (memArtefatos) SaveValidated(context.Context, domain.Documento, string, []domain.Oferta, []domain.FalhaExtracao) error {
	return nil
}

type memUsos struct {
	items []domain.UsoExtrator
}

func (m *memUsos) Save(_ context.Context, uso domain.UsoExtrator) error {
	m.items = append(m.items, uso)
	return nil
}

func TestRunDailyJob_PersistsValidOferta(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, loc)
	docs := newMemDocs()
	ofertas := &memOfertas{}
	var logBuf bytes.Buffer

	deps := application.RunDailyJobDeps{
		Fontes: &memFontes{items: []domain.Fonte{{
			ID: "f1", MercadoID: "m1", URL: "http://example",
		}}},
		Documentos: docs,
		Produtos:   &memProdutos{},
		Marcas:     &memMarcas{},
		Ofertas:    ofertas,
		Falhas:     &memFalhas{},
		FonteHTTP: &memFonteHTTP{
			pdfs: map[string][]domain.PDFDescoberto{
				"f1": {{Filename: "encarte.pdf", URL: "http://cdn/encarte.pdf"}},
			},
			body: []byte("%PDF"),
		},
		Raster: raster.Fixed{Pages: []domain.PageImage{{Page: 1, JPEG: []byte{0xff, 0xd8}}}},
		Extrator: extrator.Stub{Candidatos: []domain.CandidatoOferta{{
			Produto: "Arroz", Valor: 10, Quantidades: []float64{1000}, Medida: "g",
			DataInicio: "2026-07-18", DataExpiracao: "2026-07-25",
		}}},
		Artefatos: memArtefatos{},
		Dates:     filenamedate.Parser{},
		Clock:     fixedClock{t: now},
		Log:       log.New(&logBuf, "", 0),
		Location:  loc,
	}

	if err := application.RunDailyJob(context.Background(), deps); err != nil {
		t.Fatal(err)
	}

	doc, ok, _ := docs.GetByIdentity(context.Background(), "f1", "encarte.pdf", "2026-07-18")
	if !ok {
		t.Fatal("documento missing")
	}
	if doc.Estado != domain.EstadoConcluido {
		t.Fatalf("estado=%s ultimoErro=%s", doc.Estado, doc.UltimoErro)
	}
	got := ofertas.byDoc[doc.ID]
	if len(got) != 1 {
		t.Fatalf("ofertas=%v", got)
	}
	if got[0].DataInicio != "2026-07-18" || got[0].DataExpiracao != "2026-07-25" {
		t.Fatalf("vigencia=%s..%s", got[0].DataInicio, got[0].DataExpiracao)
	}
	if got[0].OrigemDataInicio != domain.OrigemExtrator || got[0].OrigemDataExpiracao != domain.OrigemExtrator {
		t.Fatalf("origens=%s/%s", got[0].OrigemDataInicio, got[0].OrigemDataExpiracao)
	}
}

func TestFiltrarPDFs_Regex(t *testing.T) {
	all := []domain.PDFDescoberto{
		{Filename: "atacado.pdf"},
		{Filename: "varejo.pdf"},
	}
	kept, rejected, err := application.FiltrarPDFs(domain.Fonte{FiltroNomeDocumento: "atacado"}, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0].Filename != "atacado.pdf" {
		t.Fatalf("kept=%v", kept)
	}
	if len(rejected) != 1 {
		t.Fatalf("rejected=%v", rejected)
	}
}

func TestRunDailyJob_SkipsConcluido(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, loc)
	docs := newMemDocs()
	_ = docs.Save(context.Background(), domain.Documento{
		ID: "d1", FonteID: "f1", MercadoID: "m1",
		Filename: "encarte.pdf", Dia: "2026-07-18", Estado: domain.EstadoConcluido,
	})
	calls := 0
	extr := extrator.Stub{Candidatos: nil}
	// wrap to count — use custom
	counting := &countingExtrator{inner: extr, n: &calls}

	deps := application.RunDailyJobDeps{
		Fontes:     &memFontes{items: []domain.Fonte{{ID: "f1", MercadoID: "m1"}}},
		Documentos: docs,
		Produtos:   &memProdutos{},
		Marcas:     &memMarcas{},
		Ofertas:    &memOfertas{},
		Falhas:     &memFalhas{},
		FonteHTTP: &memFonteHTTP{
			pdfs: map[string][]domain.PDFDescoberto{"f1": {{Filename: "encarte.pdf", URL: "u"}}},
			body: []byte("x"),
		},
		Raster:    raster.Fixed{Pages: []domain.PageImage{{Page: 1, JPEG: []byte{1}}}},
		Extrator:  counting,
		Artefatos: memArtefatos{},
		Dates:     filenamedate.Parser{},
		Clock:     fixedClock{t: now},
		Log:       log.New(&bytes.Buffer{}, "", 0),
		Location:  loc,
	}
	if err := application.RunDailyJob(context.Background(), deps); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("extrator called %d times", calls)
	}
}

type countingExtrator struct {
	inner domain.Extrator
	n     *int
}

func (c *countingExtrator) Extract(ctx context.Context, images []domain.PageImage) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	*c.n++
	return c.inner.Extract(ctx, images)
}

func TestRunDailyJob_OnlyFonteID(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, loc)
	docs := newMemDocs()
	calls := 0

	deps := application.RunDailyJobDeps{
		Fontes: &memFontes{items: []domain.Fonte{
			{ID: "f-skip", MercadoID: "m1"},
			{ID: "f-run", MercadoID: "m1"},
		}},
		Documentos: docs,
		Produtos:   &memProdutos{},
		Marcas:     &memMarcas{},
		Ofertas:    &memOfertas{},
		Falhas:     &memFalhas{},
		FonteHTTP: &memFonteHTTP{
			pdfs: map[string][]domain.PDFDescoberto{
				"f-skip": {{Filename: "skip.pdf", URL: "u1"}},
				"f-run":  {{Filename: "run.pdf", URL: "u2"}},
			},
			body: []byte("%PDF"),
		},
		Raster: raster.Fixed{Pages: []domain.PageImage{{Page: 1, JPEG: []byte{0xff, 0xd8}}}},
		Extrator: &countingExtrator{
			inner: extrator.Stub{Candidatos: []domain.CandidatoOferta{{
				Produto: "Feijão", Valor: 5, Quantidades: []float64{1}, Medida: "unidade",
				DataInicio: "2026-07-18", DataExpiracao: "2026-07-25",
			}}},
			n: &calls,
		},
		Artefatos:   memArtefatos{},
		Dates:       filenamedate.Parser{},
		Clock:       fixedClock{t: now},
		Log:         log.New(&bytes.Buffer{}, "", 0),
		Location:    loc,
		OnlyFonteID: "f-run",
	}

	if err := application.RunDailyJob(context.Background(), deps); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("extrator calls=%d want 1", calls)
	}
	if _, ok, _ := docs.GetByIdentity(context.Background(), "f-skip", "skip.pdf", "2026-07-18"); ok {
		t.Fatal("skipped fonte should not create documento")
	}
	doc, ok, _ := docs.GetByIdentity(context.Background(), "f-run", "run.pdf", "2026-07-18")
	if !ok || doc.Estado != domain.EstadoConcluido {
		t.Fatalf("run fonte doc ok=%v estado=%s", ok, doc.Estado)
	}
}

func TestRunDailyJob_MaxDocumentos(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, loc)
	docs := newMemDocs()
	calls := 0

	deps := application.RunDailyJobDeps{
		Fontes: &memFontes{items: []domain.Fonte{{ID: "f1", MercadoID: "m1"}}},
		Documentos: docs,
		Produtos:   &memProdutos{},
		Marcas:     &memMarcas{},
		Ofertas:    &memOfertas{},
		Falhas:     &memFalhas{},
		FonteHTTP: &memFonteHTTP{
			pdfs: map[string][]domain.PDFDescoberto{
				"f1": {
					{Filename: "a.pdf", URL: "u1"},
					{Filename: "b.pdf", URL: "u2"},
				},
			},
			body: []byte("%PDF"),
		},
		Raster: raster.Fixed{Pages: []domain.PageImage{{Page: 1, JPEG: []byte{0xff, 0xd8}}}},
		Extrator: &countingExtrator{
			inner: extrator.Stub{Candidatos: []domain.CandidatoOferta{{
				Produto: "Leite", Valor: 4, Quantidades: []float64{1000}, Medida: "ml",
				DataInicio: "2026-07-18", DataExpiracao: "2026-07-25",
			}}},
			n: &calls,
		},
		Artefatos:     memArtefatos{},
		Dates:         filenamedate.Parser{},
		Clock:         fixedClock{t: now},
		Log:           log.New(&bytes.Buffer{}, "", 0),
		Location:      loc,
		MaxDocumentos: 1,
	}

	if err := application.RunDailyJob(context.Background(), deps); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("extrator calls=%d want 1", calls)
	}
	if _, ok, _ := docs.GetByIdentity(context.Background(), "f1", "a.pdf", "2026-07-18"); !ok {
		t.Fatal("first pdf missing")
	}
	if _, ok, _ := docs.GetByIdentity(context.Background(), "f1", "b.pdf", "2026-07-18"); ok {
		t.Fatal("second pdf should not be processed")
	}
}

func TestRunDailyJob_SkipsParcial(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, loc)
	docs := newMemDocs()
	_ = docs.Save(context.Background(), domain.Documento{
		ID: "d1", FonteID: "f1", MercadoID: "m1",
		Filename: "encarte.pdf", Dia: "2026-07-18", Estado: domain.EstadoParcial,
	})
	calls := 0

	deps := application.RunDailyJobDeps{
		Fontes:     &memFontes{items: []domain.Fonte{{ID: "f1", MercadoID: "m1"}}},
		Documentos: docs,
		Produtos:   &memProdutos{},
		Marcas:     &memMarcas{},
		Ofertas:    &memOfertas{},
		Falhas:     &memFalhas{},
		FonteHTTP: &memFonteHTTP{
			pdfs: map[string][]domain.PDFDescoberto{"f1": {{Filename: "encarte.pdf", URL: "u"}}},
			body: []byte("x"),
		},
		Raster: raster.Fixed{Pages: []domain.PageImage{{Page: 1, JPEG: []byte{1}}}},
		Extrator: &countingExtrator{
			inner: extrator.Stub{Candidatos: []domain.CandidatoOferta{{
				Produto: "X", Valor: 1, Quantidades: []float64{1}, Medida: "unidade",
				DataInicio: "2026-07-18", DataExpiracao: "2026-07-25",
			}}},
			n: &calls,
		},
		Artefatos: memArtefatos{},
		Dates:     filenamedate.Parser{},
		Clock:     fixedClock{t: now},
		Log:       log.New(&bytes.Buffer{}, "", 0),
		Location:  loc,
	}
	if err := application.RunDailyJob(context.Background(), deps); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("parcial should not reprocess; calls=%d", calls)
	}
}
