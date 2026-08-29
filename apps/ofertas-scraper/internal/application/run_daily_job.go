package application

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Clock abstracts "now" for discovery day and Artefato tentativa.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// RunDailyJobDeps wires ports for the morning job (AGENTS.md pipeline).
type RunDailyJobDeps struct {
	Fontes     domain.FonteRepository
	Documentos domain.DocumentoRepository
	Produtos   domain.ProdutoRepository
	Marcas     domain.MarcaRepository
	Ofertas    domain.OfertaRepository
	Falhas     domain.FalhaExtracaoRepository
	Usos       domain.UsoExtratorRepository // optional; when nil, usage is only written to Artefatos
	FonteHTTP  domain.FonteClient
	Raster     domain.Rasterizer
	Extrator   domain.Extrator
	Artefatos  domain.ArtefatoStore
	Dates      domain.FilenameDateParser
	Clock      Clock
	Log        *log.Logger
	Location   *time.Location // America/Sao_Paulo
	// ExtratorRetryBudget is the max wall-clock spent retrying Extrator outages (ADR 0022).
	ExtratorRetryBudget time.Duration
	// OnlyFonteID, when non-empty, processes only that Fonte (smoke tests).
	OnlyFonteID domain.FonteID
	// MaxDocumentos stops after N documentos are attempted (0 = no limit). Smoke tests use 1.
	MaxDocumentos int
}

func prepareRunDeps(d RunDailyJobDeps) (RunDailyJobDeps, error) {
	if d.Clock == nil {
		d.Clock = realClock{}
	}
	if d.Log == nil {
		d.Log = log.Default()
	}
	if d.Location == nil {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			return d, err
		}
		d.Location = loc
	}
	if d.ExtratorRetryBudget <= 0 {
		d.ExtratorRetryBudget = time.Hour
	}
	return d, nil
}

// RunDailyJob processes Fontes sequentially for the current discovery day.
func RunDailyJob(ctx context.Context, d RunDailyJobDeps) error {
	var err error
	d, err = prepareRunDeps(d)
	if err != nil {
		return err
	}

	now := d.Clock.Now().In(d.Location)
	dia := now.Format("2006-01-02")

	fontes, err := d.Fontes.List(ctx)
	if err != nil {
		return fmt.Errorf("list fontes: %w", err)
	}

	extratorBudgetLeft := d.ExtratorRetryBudget
	extratorKnownDown := false
	processed := 0

	for _, fonte := range fontes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.OnlyFonteID != "" && fonte.ID != d.OnlyFonteID {
			d.Log.Printf("fonte %s skipped (RUN_FONTE_ID=%s)", fonte.ID, d.OnlyFonteID)
			continue
		}
		if d.MaxDocumentos > 0 && processed >= d.MaxDocumentos {
			break
		}
		all, err := d.FonteHTTP.DiscoverPDFs(ctx, fonte)
		if err != nil {
			d.Log.Printf("fonte %s discover failed: %v", fonte.ID, err)
			continue
		}
		kept, rejected, err := FiltrarPDFs(fonte, all)
		if err != nil {
			d.Log.Printf("fonte %s filtro inválido: %v", fonte.ID, err)
			continue
		}
		d.Log.Printf("fonte %s pdfs descobertos=%d filtrados_out=%d mantidos=%d", fonte.ID, len(all), len(rejected), len(kept))
		for _, p := range all {
			d.Log.Printf("  descoberto: %s (%s)", p.Filename, p.URL)
		}
		for _, p := range rejected {
			d.Log.Printf("  rejeitado_filtro: %s", p.Filename)
		}
		for _, p := range kept {
			d.Log.Printf("  mantido: %s", p.Filename)
		}

		for _, pdf := range kept {
			if d.MaxDocumentos > 0 && processed >= d.MaxDocumentos {
				d.Log.Printf("RUN_MAX_DOCUMENTOS=%d reached; stopping", d.MaxDocumentos)
				return nil
			}
			if err := processDocumento(ctx, d, fonte, pdf, dia, &extratorBudgetLeft, &extratorKnownDown, processDocumentoOptions{}); err != nil {
				if errors.Is(err, errGlobalInfra) {
					return err
				}
				d.Log.Printf("documento %s/%s: %v", fonte.ID, pdf.Filename, err)
			}
			processed++
		}
	}
	return nil
}

var errGlobalInfra = errors.New("global infra failure")

type processDocumentoOptions struct {
	Force bool
}

// DiscoverDocumentos creates today's Documento records for one Fonte without processing them.
func DiscoverDocumentos(ctx context.Context, d RunDailyJobDeps, fonteID domain.FonteID) error {
	var err error
	d, err = prepareRunDeps(d)
	if err != nil {
		return err
	}
	fonte, err := findFonte(ctx, d.Fontes, fonteID)
	if err != nil {
		return err
	}
	now := d.Clock.Now().In(d.Location)
	dia := now.Format("2006-01-02")
	all, err := d.FonteHTTP.DiscoverPDFs(ctx, fonte)
	if err != nil {
		return err
	}
	kept, rejected, err := FiltrarPDFs(fonte, all)
	if err != nil {
		return err
	}
	d.Log.Printf("fonte %s pdfs descobertos=%d filtrados_out=%d mantidos=%d", fonte.ID, len(all), len(rejected), len(kept))
	for _, pdf := range kept {
		if _, ok, err := d.Documentos.GetByIdentity(ctx, fonte.ID, pdf.Filename, dia); err != nil {
			return fmt.Errorf("%w: get documento: %v", errGlobalInfra, err)
		} else if ok {
			d.Log.Printf("documento já existe hoje: %s", pdf.Filename)
			continue
		}
		doc := domain.Documento{
			ID:         domain.DocumentoID(NewID()),
			FonteID:    fonte.ID,
			MercadoID:  fonte.MercadoID,
			Filename:   pdf.Filename,
			Dia:        dia,
			Estado:     domain.EstadoProcessando,
			UltimoErro: "descoberto sem processamento",
			Atualizado: d.Clock.Now().UTC(),
		}
		if err := d.Documentos.Save(ctx, doc); err != nil {
			return fmt.Errorf("%w: save documento: %v", errGlobalInfra, err)
		}
		d.Log.Printf("documento descoberto: %s (%s)", doc.ID, doc.Filename)
	}
	return nil
}

// ReprocessDocumento forces a Documento through processing even when its state would normally skip.
func ReprocessDocumento(ctx context.Context, d RunDailyJobDeps, documentoID domain.DocumentoID) error {
	var err error
	d, err = prepareRunDeps(d)
	if err != nil {
		return err
	}
	doc, ok, err := d.Documentos.Get(ctx, documentoID)
	if err != nil {
		return fmt.Errorf("%w: get documento: %v", errGlobalInfra, err)
	}
	if !ok {
		return fmt.Errorf("documento %s not found", documentoID)
	}
	fonte, err := findFonte(ctx, d.Fontes, doc.FonteID)
	if err != nil {
		return err
	}
	all, err := d.FonteHTTP.DiscoverPDFs(ctx, fonte)
	if err != nil {
		return err
	}
	kept, _, err := FiltrarPDFs(fonte, all)
	if err != nil {
		return err
	}
	var pdf domain.PDFDescoberto
	found := false
	for _, candidate := range kept {
		if candidate.Filename == doc.Filename {
			pdf = candidate
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("documento %s filename %q não encontrado na Fonte %s", documentoID, doc.Filename, fonte.ID)
	}
	extratorBudgetLeft := d.ExtratorRetryBudget
	extratorKnownDown := false
	return processDocumento(ctx, d, fonte, pdf, doc.Dia, &extratorBudgetLeft, &extratorKnownDown, processDocumentoOptions{Force: true})
}

func findFonte(ctx context.Context, repo domain.FonteRepository, fonteID domain.FonteID) (domain.Fonte, error) {
	fontes, err := repo.List(ctx)
	if err != nil {
		return domain.Fonte{}, err
	}
	for _, fonte := range fontes {
		if fonte.ID == fonteID {
			return fonte, nil
		}
	}
	return domain.Fonte{}, fmt.Errorf("fonte %s not found", fonteID)
}

func processDocumento(
	ctx context.Context,
	d RunDailyJobDeps,
	fonte domain.Fonte,
	pdf domain.PDFDescoberto,
	dia string,
	extratorBudgetLeft *time.Duration,
	extratorKnownDown *bool,
	opts processDocumentoOptions,
) error {
	existing, ok, err := d.Documentos.GetByIdentity(ctx, fonte.ID, pdf.Filename, dia)
	if err != nil {
		return fmt.Errorf("%w: get documento: %v", errGlobalInfra, err)
	}
	var doc domain.Documento
	if ok {
		if !opts.Force && !domain.DeveReprocessar(existing.Estado) {
			d.Log.Printf("skip %s (%s)", pdf.Filename, existing.Estado)
			return nil
		}
		doc = existing
	} else {
		doc = domain.Documento{
			ID:        domain.DocumentoID(NewID()),
			FonteID:   fonte.ID,
			MercadoID: fonte.MercadoID,
			Filename:  pdf.Filename,
			Dia:       dia,
		}
	}

	now := d.Clock.Now().UTC()
	tentativa := now.Format("20060102T150405")
	doc.Estado = domain.EstadoProcessando
	doc.UltimoErro = ""
	doc.ConteudoIdenticoA = nil
	doc.Atualizado = now
	if err := d.Documentos.Save(ctx, doc); err != nil {
		return fmt.Errorf("%w: save documento: %v", errGlobalInfra, err)
	}
	// Clear previous Ofertas/Falhas for this Documento (ADR 0017/0036) before the new attempt.
	if err := d.Ofertas.SaveAll(ctx, doc.ID, nil); err != nil {
		return fmt.Errorf("%w: clear ofertas: %v", errGlobalInfra, err)
	}
	if err := d.Falhas.SaveAll(ctx, doc.ID, nil); err != nil {
		return fmt.Errorf("%w: clear falhas: %v", errGlobalInfra, err)
	}

	fail := func(msg string) error {
		doc.Estado = domain.EstadoFalhou
		doc.UltimoErro = msg
		doc.Atualizado = d.Clock.Now().UTC()
		if err := d.Documentos.Save(ctx, doc); err != nil {
			return fmt.Errorf("%w: save falhou: %v", errGlobalInfra, err)
		}
		d.Log.Printf("%s: %s", pdf.Filename, msg)
		return nil
	}

	pdfBytes, err := d.FonteHTTP.DownloadPDF(ctx, pdf.URL)
	if err != nil {
		return fail(fmt.Sprintf("download: %v", err))
	}
	fp := domain.FingerprintPDF(pdfBytes)
	doc.Fingerprint = fp
	if err := d.Documentos.Save(ctx, doc); err != nil {
		return fmt.Errorf("%w: save fingerprint: %v", errGlobalInfra, err)
	}
	if err := d.Artefatos.SavePDF(ctx, doc, tentativa, pdfBytes); err != nil {
		return fail(fmt.Sprintf("artefato pdf: %v", err))
	}

	if prior, ok, err := findConteudoIdentico(ctx, d, fonte.ID, pdf.Filename, dia, fp); err != nil {
		return fmt.Errorf("%w: conteudo identico: %v", errGlobalInfra, err)
	} else if ok {
		return concludeConteudoIdentico(ctx, d, doc, tentativa, prior, pdf.Filename)
	}

	images, err := d.Raster.Rasterize(ctx, pdfBytes)
	if err != nil {
		return fail(fmt.Sprintf("raster: %v", err))
	}
	if err := d.Artefatos.SaveImages(ctx, doc, tentativa, images); err != nil {
		return fail(fmt.Sprintf("artefato images: %v", err))
	}

	var candidatos []domain.CandidatoOferta
	var raw []byte
	var uso *domain.UsoExtrator
	if *extratorKnownDown {
		return fail("extrator indisponivel (orçamento de retry esgotado neste run)")
	}
	candidatos, raw, uso, err = extractWithRetry(ctx, d, images, extratorBudgetLeft, extratorKnownDown)
	if err != nil {
		return fail(fmt.Sprintf("extrator: %v", err))
	}
	if err := d.Artefatos.SaveRawExtrator(ctx, doc, tentativa, raw); err != nil {
		return fail(fmt.Sprintf("artefato raw: %v", err))
	}
	if uso != nil {
		uso.DocumentoID = doc.ID
		uso.Tentativa = tentativa
		uso.ArtefatoPath = d.Artefatos.AttemptPath(doc, tentativa)
		if err := d.Artefatos.SaveUsoExtrator(ctx, doc, tentativa, *uso); err != nil {
			return fail(fmt.Sprintf("artefato uso: %v", err))
		}
		if d.Usos != nil {
			if err := d.Usos.Save(ctx, *uso); err != nil {
				return fail(fmt.Sprintf("persist uso: %v", err))
			}
		}
		d.Log.Printf("%s uso-extrator provider=%s model=%s prompt=%d cache=%d output=%d path=%s",
			pdf.Filename, uso.Provider, uso.Model, uso.PromptTokens, uso.CacheTokens, uso.OutputTokens, uso.ArtefatoPath)
	}

	primeiroDia := dia
	if earliest, ok, err := d.Documentos.EarliestDia(ctx, fonte.ID, pdf.Filename); err != nil {
		return fmt.Errorf("%w: earliest dia: %v", errGlobalInfra, err)
	} else if ok && earliest != "" {
		primeiroDia = earliest
	}
	resolvidos := make([]CandidatoResolvido, len(candidatos))
	for i, c := range candidatos {
		resolvidos[i] = ResolverVigencia(c, pdf.Filename, d.Dates, primeiroDia)
	}
	validas, falhas, estado := ValidarExtracaoResolvida(resolvidos)

	ofertas, err := PersistirOfertasValidas(ctx, d.Produtos, d.Marcas, d.Ofertas, doc, validas)
	if err != nil {
		return fail(fmt.Sprintf("match-or-create: %v", err))
	}
	if err := d.Ofertas.SaveAll(ctx, doc.ID, ofertas); err != nil {
		return fmt.Errorf("%w: save ofertas: %v", errGlobalInfra, err)
	}
	if err := d.Falhas.SaveAll(ctx, doc.ID, falhas); err != nil {
		return fmt.Errorf("%w: save falhas: %v", errGlobalInfra, err)
	}
	if err := d.Artefatos.SaveValidated(ctx, doc, tentativa, ofertas, falhas); err != nil {
		return fail(fmt.Sprintf("artefato validated: %v", err))
	}

	doc.Estado = estado
	doc.UltimoErro = ""
	doc.Atualizado = d.Clock.Now().UTC()
	if err := d.Documentos.Save(ctx, doc); err != nil {
		return fmt.Errorf("%w: save documento final: %v", errGlobalInfra, err)
	}
	d.Log.Printf("%s → %s (ofertas=%d falhas=%d)", pdf.Filename, estado, len(ofertas), len(falhas))
	return nil
}

func findConteudoIdentico(
	ctx context.Context,
	d RunDailyJobDeps,
	fonteID domain.FonteID,
	filename, dia, fingerprint string,
) (domain.Documento, bool, error) {
	dias, err := d.Documentos.ListDias(ctx, fonteID, filename)
	if err != nil {
		return domain.Documento{}, false, err
	}
	var best domain.Documento
	found := false
	for _, otherDia := range dias {
		if otherDia == dia {
			continue
		}
		prior, ok, err := d.Documentos.GetByIdentity(ctx, fonteID, filename, otherDia)
		if err != nil {
			return domain.Documento{}, false, err
		}
		if !ok || prior.Estado != domain.EstadoConcluido || prior.Fingerprint == "" {
			continue
		}
		if prior.Fingerprint != fingerprint {
			continue
		}
		if !found || prior.Dia > best.Dia {
			best = prior
			found = true
		}
	}
	return best, found, nil
}

func concludeConteudoIdentico(
	ctx context.Context,
	d RunDailyJobDeps,
	doc domain.Documento,
	tentativa string,
	prior domain.Documento,
	filename string,
) error {
	ofertas, err := d.Ofertas.ListByDocumento(ctx, prior.ID)
	if err != nil {
		return fmt.Errorf("%w: list ofertas prior: %v", errGlobalInfra, err)
	}
	if err := d.Ofertas.SaveAll(ctx, doc.ID, ofertas); err != nil {
		return fmt.Errorf("%w: associate ofertas: %v", errGlobalInfra, err)
	}
	priorID := prior.ID
	doc.ConteudoIdenticoA = &priorID
	doc.Estado = domain.EstadoConcluido
	doc.UltimoErro = ""
	doc.Atualizado = d.Clock.Now().UTC()
	if err := d.Artefatos.SaveConteudoIdentico(ctx, doc, tentativa, prior.ID); err != nil {
		return fmt.Errorf("%w: artefato conteudo identico: %v", errGlobalInfra, err)
	}
	if err := d.Documentos.Save(ctx, doc); err != nil {
		return fmt.Errorf("%w: save documento identico: %v", errGlobalInfra, err)
	}
	d.Log.Printf("%s → concluido (conteudo identico a %s; ofertas=%d)", filename, prior.ID, len(ofertas))
	return nil
}

func extractWithRetry(
	ctx context.Context,
	d RunDailyJobDeps,
	images []domain.PageImage,
	budgetLeft *time.Duration,
	knownDown *bool,
) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	backoff := 5 * time.Minute
	const maxBackoff = 30 * time.Minute

	for {
		candidatos, raw, uso, err := d.Extrator.Extract(ctx, images)
		if err == nil {
			return candidatos, raw, uso, nil
		}
		if errors.Is(err, domain.ErrExtratorCota) {
			return nil, nil, nil, err
		}
		if !errors.Is(err, domain.ErrExtratorIndisponivel) {
			return nil, nil, nil, err
		}
		if *budgetLeft <= 0 {
			*knownDown = true
			return nil, nil, nil, domain.ErrExtratorIndisponivel
		}
		wait := backoff
		if wait > *budgetLeft {
			wait = *budgetLeft
		}
		d.Log.Printf("extrator indisponivel; retry em %s (budget restante %s)", wait, *budgetLeft)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, nil, ctx.Err()
		case <-timer.C:
		}
		*budgetLeft -= wait
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
