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

// RunDailyJob processes Fontes sequentially for the current discovery day.
func RunDailyJob(ctx context.Context, d RunDailyJobDeps) error {
	if d.Clock == nil {
		d.Clock = realClock{}
	}
	if d.Log == nil {
		d.Log = log.Default()
	}
	if d.Location == nil {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			return err
		}
		d.Location = loc
	}
	if d.ExtratorRetryBudget <= 0 {
		d.ExtratorRetryBudget = time.Hour
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
			if err := processDocumento(ctx, d, fonte, pdf, dia, &extratorBudgetLeft, &extratorKnownDown); err != nil {
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

func processDocumento(
	ctx context.Context,
	d RunDailyJobDeps,
	fonte domain.Fonte,
	pdf domain.PDFDescoberto,
	dia string,
	extratorBudgetLeft *time.Duration,
	extratorKnownDown *bool,
) error {
	existing, ok, err := d.Documentos.GetByIdentity(ctx, fonte.ID, pdf.Filename, dia)
	if err != nil {
		return fmt.Errorf("%w: get documento: %v", errGlobalInfra, err)
	}
	var doc domain.Documento
	if ok {
		if !domain.DeveReprocessar(existing.Estado) {
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
	doc.Atualizado = now
	if err := d.Documentos.Save(ctx, doc); err != nil {
		return fmt.Errorf("%w: save documento: %v", errGlobalInfra, err)
	}
	// Clear previous Ofertas/Falhas for this Documento (ADR 0017) before the new attempt.
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
	if err := d.Artefatos.SavePDF(ctx, doc, tentativa, pdfBytes); err != nil {
		return fail(fmt.Sprintf("artefato pdf: %v", err))
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
	if *extratorKnownDown {
		return fail("extrator indisponivel (orçamento de retry esgotado neste run)")
	}
	candidatos, raw, err = extractWithRetry(ctx, d, images, extratorBudgetLeft, extratorKnownDown)
	if err != nil {
		return fail(fmt.Sprintf("extrator: %v", err))
	}
	if err := d.Artefatos.SaveRawExtrator(ctx, doc, tentativa, raw); err != nil {
		return fail(fmt.Sprintf("artefato raw: %v", err))
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

	ofertas, err := PersistirOfertasValidas(ctx, d.Produtos, d.Marcas, doc, validas)
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

func extractWithRetry(	ctx context.Context,
	d RunDailyJobDeps,
	images []domain.PageImage,
	budgetLeft *time.Duration,
	knownDown *bool,
) ([]domain.CandidatoOferta, []byte, error) {
	backoff := time.Second
	const maxBackoff = 5 * time.Minute

	for {
		candidatos, raw, err := d.Extrator.Extract(ctx, images)
		if err == nil {
			return candidatos, raw, nil
		}
		if !errors.Is(err, domain.ErrExtratorIndisponivel) {
			return nil, nil, err
		}
		if *budgetLeft <= 0 {
			*knownDown = true
			return nil, nil, domain.ErrExtratorIndisponivel
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
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
		*budgetLeft -= wait
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
