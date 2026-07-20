# Extrator: live recall against Artefatos

How we measure and iterate Extrator quality without re-running the full daily job. Use this when changing `prompts/extrator.txt`, a Gemini/Cursor adapter, or raster settings.

## Mandatory: manual validation after prompt/schema changes

**Every** change to `prompts/extrator.txt` or `schemas/oferta.json` / `schemas/extracao.json` must be followed by a **manual** live recall check before considering the change done — not only automated floors (`≥90` Ofertas, `≥30` promocao). Domain tests catch invalid candidates; they do **not** prove the model filled the right fields on the right cells.

Minimum manual checklist:

1. Re-run opt-in live extract (`LIVE_EXTRATOR=1`, see below).
2. Open `live-page-by-page/extrator-raw.json` next to `images/page-*.jpg`.
3. Spot-check the cells that motivated the change (and a sample of others for regressions).
4. Confirm counts stayed in the expected band and that new fields (e.g. `comparativo`) appear where the flyer shows them — without false second Ofertas.

Skipping this step after a prompt/schema edit is a process failure, even if unit tests are green.

## Why

Domain validation and Falhas de Extração only catch bad candidates. **Missing Ofertas never become Falhas** — they simply never appear in `extrator-raw.json`. Recall must be checked by comparing Extrator output to the page images (or a known count).

## Fixture

Prefer a retained Artefato attempt that already has the images the model saw:

```
apps/ofertas-scraper/.data/artefatos/{fonteId}/{dia}/{filename}/{tentativa}/
  images/page-001.jpg …
  extrator-raw.json      # previous run (baseline)
  validated.json
  original.pdf
```

Example used for ADR 0031 (Fort Canoas, 4 pages):

`fonte-fort/2026-07-20/RS_Fort_Semanal-4a-ED_Regional_20-a-24_JUL_26-Canoas-Final.pdf/20260720T034602`

Override with `LIVE_ARTEFATO_DIR` if you want another attempt.

## Opt-in live tests

From `apps/ofertas-scraper` (load `.env` so an Extrator key is set; force real Extrator):

```bash
set -a && source .env && set +a
LIVE_EXTRATOR=1 EXTRATOR_STUB=0 \
  go test ./internal/infra/extrator/ \
  -run TestLiveFortArtefato \
  -count=1 -timeout 15m -v
```

| Env | Role |
|-----|------|
| `LIVE_EXTRATOR=1` | Unskip the live test |
| `LIVE_ARTEFATO_DIR` | Optional path to an attempt directory with `images/` |
| `LIVE_PAGE1_JPEG` | Optional single JPEG for `TestLiveFortPage1Hires` (probe insets / small packs) |
| `CURSOR_API_KEY` | Preferred when set (`EXTRATOR_PROVIDER=auto`) |
| `CURSOR_MODEL` | Optional; default `composer-2.5` |
| `GEMINI_API_KEY` | Used when Cursor key absent or `EXTRATOR_PROVIDER=gemini` |
| `GEMINI_MODEL` | Optional; defaults as in Gemini adapter |

Cursor live runs need `python3` + `pip install cursor-sdk` (ADR 0034).

CI and normal `go test ./...` **skip** these tests (no `LIVE_EXTRATOR`).

### What “good” looked like (Fort)

| Step | Result |
|------|--------|
| Before ADR 0031 (all pages in one call) | ~20 Ofertas for the whole Documento |
| After one call per page + prompt exhaustiveness | ~95 Ofertas (`21+25+24+25` per page log lines) |

Page-1 grid alone is ~20 cells; Girando Sol 4kg @ R$ 24,90 with red “nesta embalagem 800 g sai por R$ 4,98” is **one** Oferta with `comparativo` (ADR 0035) — not a second Oferta. Assert in the live test is `≥90` as a regression floor, not a hard ground-truth total. Promo floor: `≥30` with `promocao` (ADR 0032).

## Iteration loop

1. **Baseline** — count `ofertas` in the attempt’s `extrator-raw.json`; open `images/page-*.jpg` and spot-check missing cells (especially first page).
2. **Change one lever** — prompt wording, per-page vs multi-page (ADR 0031), resolution (`RASTER_MAX_EDGE_PX`), or model. Prefer one change per live run (API cost/latency).
3. **Re-run live test** — watch logs `gemini/cursor extrator page=N ofertas=M`; fail if total collapses.
4. **Compare to images** — duplicates (same produto/valor twice), wrong `quantidades`, cartão/clube as second Oferta instead of `promocao`, “nesta embalagem … sai por” as second Oferta instead of `comparativo`, missed real dual-SKU insets, missing `promocao` on VuonCard/Clube/leve-pague cells.
5. **Optional side folder** — `TestLiveFortArtefato` writes `…/{filename}/live-page-by-page/extrator-raw.json`, `uso-extrator.json` (token counts), and symlink `images/` → attempt images. Not a formal Artefato tentativa; just for human review. Redis Uso do Extrator is only on the real job (`run`).

## Prompt pitfalls already seen

- **Sampling** — one vision call with many dense pages → model returns a handful of Ofertas from each page. Fix: one call per page (ADR 0031).
- **Duplicates** — “extract everything” without “don’t duplicate” → same Nhoque twice; cartão/clube as extra Ofertas. Fix: promo in `promocao`; comparativo badge in `comparativo`; second Oferta only for distinct pack with its own shelf price.
- **Insets** — real secondary pack with its own shelf price needs an explicit second Oferta; “nesta embalagem X sai por Y” is `comparativo` on the main pack (ADR 0035). Higher-res probe when unsure (`TestLiveFortPage1Hires` + `pdftoppm -r 200`).

## Related

- Code: `internal/infra/extrator/gemini_live_test.go`, `gemini_live_page1_test.go`, `cursor.go`
- ADR: [`docs/adr/0031-extrator-one-call-per-page.md`](./adr/0031-extrator-one-call-per-page.md), [`docs/adr/0034-extrator-cursor-agent-sdk.md`](./adr/0034-extrator-cursor-agent-sdk.md)
- Prompt: [`../prompts/extrator.txt`](../prompts/extrator.txt)
