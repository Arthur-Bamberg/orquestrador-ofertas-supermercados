package redismigrate_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/redismigrate"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

func TestCopyFromRedis_MercadoFonteDocumentoOferta(t *testing.T) {
	redis, closeRedis := newFakeRedis(t)
	defer closeRedis()
	ctx := context.Background()
	src := redismigrate.NewClient(redis.URL, "token", redis.Client())
	seedCatalogRedis(t, src)

	dst := storetest.New(t)
	if err := redismigrate.CopyFromRedis(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	got, ok, err := dst.GetMercado(ctx, "m1")
	if err != nil || !ok || got.Nome != "Fort" {
		t.Fatalf("mercado ok=%v err=%v got=%+v", ok, err, got)
	}
	list, err := dst.ListOfertasByDocumento(ctx, "d1")
	if err != nil || len(list) != 1 || list[0].ID != "o1" {
		t.Fatalf("ofertas=%v err=%v", list, err)
	}
}

func TestCopyFromRedis_OperacaoSemHistoryUsaKeys(t *testing.T) {
	redis, closeRedis := newFakeRedis(t)
	defer closeRedis()
	ctx := context.Background()
	src := redismigrate.NewClient(redis.URL, "token", redis.Client())
	now := time.Date(2026, 7, 21, 8, 0, 0, 0, time.UTC)
	op := store.OperacaoPipeline{ID: "op1", Kind: store.OperacaoRun, Status: store.OperacaoSucceeded, CreatedAt: now}
	raw, _ := json.Marshal(op)
	if err := src.Set(ctx, "ofertas-api:ops:op1", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := src.Set(ctx, "ofertas-api:ops:queue", `["op1"]`); err != nil {
		t.Fatal(err)
	}

	dst := storetest.New(t)
	if err := redismigrate.CopyFromRedis(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	got, ok, err := dst.GetOperacao(ctx, "op1")
	if err != nil || !ok || got.Status != store.OperacaoSucceeded {
		t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
	}
}

func TestCopyFromRedis_IdempotentePorPK(t *testing.T) {
	redis, closeRedis := newFakeRedis(t)
	defer closeRedis()
	ctx := context.Background()
	src := redismigrate.NewClient(redis.URL, "token", redis.Client())
	seedCatalogRedis(t, src)
	now := time.Date(2026, 7, 21, 8, 0, 0, 0, time.UTC)
	op := store.OperacaoPipeline{ID: "op1", Kind: store.OperacaoRun, Status: store.OperacaoFailed, CreatedAt: now, Error: "boom"}
	raw, _ := json.Marshal(op)
	if err := src.Set(ctx, "ofertas-api:ops:op1", string(raw)); err != nil {
		t.Fatal(err)
	}

	dst := storetest.New(t)
	if err := redismigrate.CopyFromRedis(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	if err := redismigrate.CopyFromRedis(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	got, ok, err := dst.GetOperacao(ctx, "op1")
	if err != nil || !ok || got.Status != store.OperacaoFailed || got.Error != "boom" {
		t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
	}
	list, err := dst.ListOfertasByDocumento(ctx, "d1")
	if err != nil || len(list) != 1 {
		t.Fatalf("ofertas=%v err=%v", list, err)
	}
}

func TestCopyFromRedis_OfertaSemDocumentoFalha(t *testing.T) {
	redis, closeRedis := newFakeRedis(t)
	defer closeRedis()
	ctx := context.Background()
	src := redismigrate.NewClient(redis.URL, "token", redis.Client())
	if err := src.Set(ctx, "mercado:m1", `{"id":"m1","nome":"Fort"}`); err != nil {
		t.Fatal(err)
	}
	if err := src.SAdd(ctx, "mercados", "m1"); err != nil {
		t.Fatal(err)
	}
	if err := src.Set(ctx, "produto:p1", `{"id":"p1","nome":"Arroz","nomeNorm":"arroz"}`); err != nil {
		t.Fatal(err)
	}
	if err := src.SAdd(ctx, "produtos", "p1"); err != nil {
		t.Fatal(err)
	}
	o := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	raw, _ := json.Marshal(o)
	if err := src.Set(ctx, "oferta:o1", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := src.SAdd(ctx, "ofertas", "o1"); err != nil {
		t.Fatal(err)
	}

	dst := storetest.New(t)
	err := redismigrate.CopyFromRedis(ctx, src, dst)
	if err == nil {
		t.Fatal("esperado erro ao copiar Oferta sem Documento")
	}
	if !strings.Contains(err.Error(), "o1") {
		t.Fatalf("err=%v", err)
	}
}

func TestCopyFromRedis_FalhasUsoCota(t *testing.T) {
	redis, closeRedis := newFakeRedis(t)
	defer closeRedis()
	ctx := context.Background()
	src := redismigrate.NewClient(redis.URL, "token", redis.Client())
	seedCatalogRedis(t, src)
	falhas := []store.FalhaExtracao{{Codigo: "sem_valor", Detalhe: "x"}}
	raw, _ := json.Marshal(falhas)
	if err := src.Set(ctx, "falhas:documento:d1", string(raw)); err != nil {
		t.Fatal(err)
	}
	uso := store.UsoExtrator{DocumentoID: "d1", Tentativa: "t1", Provider: "stub", PromptTokens: 9}
	raw, _ = json.Marshal(uso)
	if err := src.Set(ctx, "uso-extrator:d1:t1", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := src.Set(ctx, "ofertas-scraper:extrator:gemini:esgotado:2026-07-21", "1"); err != nil {
		t.Fatal(err)
	}

	dst := storetest.New(t)
	if err := redismigrate.CopyFromRedis(ctx, src, dst); err != nil {
		t.Fatal(err)
	}
	gotFalhas, err := dst.GetFalhasDocumento(ctx, "d1")
	if err != nil || len(gotFalhas) != 1 || gotFalhas[0].Codigo != "sem_valor" {
		t.Fatalf("falhas=%v err=%v", gotFalhas, err)
	}
	gotUso, ok, err := dst.GetUsoExtrator(ctx, "d1", "t1")
	if err != nil || !ok || gotUso.PromptTokens != 9 {
		t.Fatalf("uso ok=%v err=%v got=%+v", ok, err, gotUso)
	}
	ok, err = dst.Esgotado(ctx, "gemini", "2026-07-21")
	if err != nil || !ok {
		t.Fatalf("cota ok=%v err=%v", ok, err)
	}
}

func seedCatalogRedis(t *testing.T, src *redismigrate.Client) {
	t.Helper()
	ctx := context.Background()
	put := func(key, set, raw string) {
		t.Helper()
		if err := src.Set(ctx, key, raw); err != nil {
			t.Fatal(err)
		}
		if set != "" {
			id := strings.TrimPrefix(key, strings.Split(key, ":")[0]+":")
			if err := src.SAdd(ctx, set, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	put("mercado:m1", "mercados", `{"id":"m1","nome":"Fort"}`)
	put("fonte:f1", "fontes", `{"id":"f1","mercadoId":"m1","url":"https://example.com"}`)
	put("produto:p1", "produtos", `{"id":"p1","nome":"Arroz","nomeNorm":"arroz"}`)
	put("documento:d1", "documentos", `{"id":"d1","fonteId":"f1","mercadoId":"m1","filename":"a.pdf","dia":"2026-07-21"}`)
	o := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-21", DataExpiracao: "2026-07-22", DocumentoID: "d1",
	}
	raw, _ := json.Marshal(o)
	put("oferta:o1", "ofertas", string(raw))
	if err := src.SAdd(ctx, "oferta:documentos:o1", "d1"); err != nil {
		t.Fatal(err)
	}
}

func newFakeRedis(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	values := map[string]string{}
	sets := map[string]map[string]struct{}{}
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cmd []any
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		defer mu.Unlock()
		switch cmd[0].(string) {
		case "SET":
			values[cmd[1].(string)] = cmd[2].(string)
			writeRedisResult(w, "OK")
		case "GET":
			v, ok := values[cmd[1].(string)]
			if !ok {
				writeRedisRaw(w, "null")
				return
			}
			writeRedisResult(w, v)
		case "SADD":
			key := cmd[1].(string)
			if sets[key] == nil {
				sets[key] = map[string]struct{}{}
			}
			sets[key][cmd[2].(string)] = struct{}{}
			writeRedisResult(w, 1)
		case "SMEMBERS":
			var members []string
			for m := range sets[cmd[1].(string)] {
				members = append(members, m)
			}
			b, _ := json.Marshal(members)
			writeRedisRaw(w, string(b))
		case "KEYS":
			pat := cmd[1].(string)
			var keys []string
			for k := range values {
				if redisGlob(pat, k) {
					keys = append(keys, k)
				}
			}
			b, _ := json.Marshal(keys)
			writeRedisRaw(w, string(b))
		default:
			t.Fatalf("unexpected op %s", cmd[0])
		}
	}))
	return srv, srv.Close
}

func redisGlob(pattern, key string) bool {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, `.*`)
	re, err := regexp.Compile("^" + escaped + "$")
	if err != nil {
		return false
	}
	return re.MatchString(key)
}

func writeRedisResult(w http.ResponseWriter, v any) {
	b, _ := json.Marshal(map[string]any{"result": v})
	_, _ = w.Write(b)
}

func writeRedisRaw(w http.ResponseWriter, resultJSON string) {
	_, _ = w.Write([]byte(`{"result":` + resultJSON + `}`))
}
