package upstash_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/upstash"
)

func TestMercadoRepo_SaveGetList(t *testing.T) {
	store := map[string]string{}
	sets := map[string]map[string]struct{}{}
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cmd []any
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		defer mu.Unlock()
		op := cmd[0].(string)
		switch op {
		case "SET":
			store[cmd[1].(string)] = cmd[2].(string)
			writeResult(w, "OK")
		case "GET":
			v, ok := store[cmd[1].(string)]
			if !ok {
				writeRaw(w, "null")
				return
			}
			writeResult(w, v)
		case "SADD":
			key := cmd[1].(string)
			if sets[key] == nil {
				sets[key] = map[string]struct{}{}
			}
			sets[key][cmd[2].(string)] = struct{}{}
			writeResult(w, 1)
		case "SMEMBERS":
			key := cmd[1].(string)
			var members []string
			for m := range sets[key] {
				members = append(members, m)
			}
			b, _ := json.Marshal(members)
			writeRaw(w, string(b))
		default:
			t.Fatalf("unexpected op %s", op)
		}
	}))
	defer srv.Close()

	client := upstash.NewClient(srv.URL, "token", srv.Client())
	repo := upstash.NewMercadoRepo(client)
	ctx := context.Background()

	m := domain.Mercado{ID: "m1", Nome: "Atacadão"}
	if err := repo.Save(ctx, m); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Nome != "Atacadão" {
		t.Fatalf("got %+v", got)
	}
	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func TestDocumentoRepo_EarliestDia(t *testing.T) {
	store := map[string]string{}
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
			store[cmd[1].(string)] = cmd[2].(string)
			writeResult(w, "OK")
		case "SADD":
			key := cmd[1].(string)
			if sets[key] == nil {
				sets[key] = map[string]struct{}{}
			}
			sets[key][cmd[2].(string)] = struct{}{}
			writeResult(w, 1)
		case "SMEMBERS":
			key := cmd[1].(string)
			var members []string
			for m := range sets[key] {
				members = append(members, m)
			}
			b, _ := json.Marshal(members)
			writeRaw(w, string(b))
		default:
			t.Fatalf("unexpected op %s", cmd[0])
		}
	}))
	defer srv.Close()

	repo := upstash.NewDocumentoRepo(upstash.NewClient(srv.URL, "t", srv.Client()))
	ctx := context.Background()
	fonte := domain.FonteID("fonte-fort")
	file := "RS_Fort_FDS_Regional_18-e-19_JUL_26-Canoas-Final.pdf"

	if err := repo.Save(ctx, domain.Documento{
		ID: "d2", FonteID: fonte, Filename: file, Dia: "2026-07-18", Estado: domain.EstadoConcluido,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, domain.Documento{
		ID: "d1", FonteID: fonte, Filename: file, Dia: "2026-07-10", Estado: domain.EstadoFalhou,
	}); err != nil {
		t.Fatal(err)
	}

	dia, ok, err := repo.EarliestDia(ctx, fonte, file)
	if err != nil || !ok || dia != "2026-07-10" {
		t.Fatalf("earliest=%q ok=%v err=%v", dia, ok, err)
	}
}

func TestOfertaRepo_SaveAllReplacesIncludingEmpty(t *testing.T) {
	repo, _ := newOfertaRepoTestEnv(t)
	ctx := context.Background()
	docID := domain.DocumentoID("d1")

	err := repo.SaveAll(ctx, docID, []domain.Oferta{{ID: "o1", DocumentoID: docID, Valor: 1.5}})
	if err != nil {
		t.Fatal(err)
	}
	list, _ := repo.ListByDocumento(ctx, docID)
	if len(list) != 1 {
		t.Fatalf("want 1 got %d", len(list))
	}
	if err := repo.SaveAll(ctx, docID, nil); err != nil {
		t.Fatal(err)
	}
	list, _ = repo.ListByDocumento(ctx, docID)
	if len(list) != 0 {
		t.Fatalf("want empty after replace, got %d", len(list))
	}
}

func TestOfertaRepo_SaveAllIndexesByProduto(t *testing.T) {
	repo, _ := newOfertaRepoTestEnv(t)
	ctx := context.Background()
	docID := domain.DocumentoID("d1")

	err := repo.SaveAll(ctx, docID, []domain.Oferta{
		{ID: "o1", DocumentoID: docID, ProdutoID: "p1", Valor: 1},
		{ID: "o2", DocumentoID: docID, ProdutoID: "p2", Valor: 2},
		{ID: "o3", DocumentoID: docID, ProdutoID: "p1", Valor: 3}, // same produto twice
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, pid := range []domain.ProdutoID{"p1", "p2"} {
		ids, err := repo.ListDocumentoIDsByProduto(ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != docID {
			t.Fatalf("produto %s: want [%s], got %v", pid, docID, ids)
		}
	}
}

func TestOfertaRepo_SaveAllUpdatesIndexOnResave(t *testing.T) {
	repo, _ := newOfertaRepoTestEnv(t)
	ctx := context.Background()
	docID := domain.DocumentoID("d1")

	if err := repo.SaveAll(ctx, docID, []domain.Oferta{
		{ID: "o1", DocumentoID: docID, ProdutoID: "p1", Valor: 1},
		{ID: "o2", DocumentoID: docID, ProdutoID: "p2", Valor: 2},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAll(ctx, docID, []domain.Oferta{
		{ID: "o3", DocumentoID: docID, ProdutoID: "p2", Valor: 3},
		{ID: "o4", DocumentoID: docID, ProdutoID: "p3", Valor: 4},
	}); err != nil {
		t.Fatal(err)
	}

	assertDocIDs(t, repo, "p1", nil)
	assertDocIDs(t, repo, "p2", []domain.DocumentoID{docID})
	assertDocIDs(t, repo, "p3", []domain.DocumentoID{docID})
}

func TestOfertaRepo_EmptySaveAllClearsIndex(t *testing.T) {
	repo, _ := newOfertaRepoTestEnv(t)
	ctx := context.Background()
	docID := domain.DocumentoID("d1")

	if err := repo.SaveAll(ctx, docID, []domain.Oferta{
		{ID: "o1", DocumentoID: docID, ProdutoID: "p1", Valor: 1},
		{ID: "o2", DocumentoID: docID, ProdutoID: "p2", Valor: 2},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAll(ctx, docID, nil); err != nil {
		t.Fatal(err)
	}

	assertDocIDs(t, repo, "p1", nil)
	assertDocIDs(t, repo, "p2", nil)
	list, err := repo.ListByDocumento(ctx, docID)
	if err != nil || len(list) != 0 {
		t.Fatalf("blob want empty, got %v err=%v", list, err)
	}
}

func newOfertaRepoTestEnv(t *testing.T) (*upstash.OfertaRepo, *httptest.Server) {
	t.Helper()
	store := map[string]string{}
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
			store[cmd[1].(string)] = cmd[2].(string)
			writeResult(w, "OK")
		case "GET":
			v, ok := store[cmd[1].(string)]
			if !ok {
				writeRaw(w, "null")
				return
			}
			writeResult(w, v)
		case "SADD":
			key := cmd[1].(string)
			if sets[key] == nil {
				sets[key] = map[string]struct{}{}
			}
			sets[key][cmd[2].(string)] = struct{}{}
			writeResult(w, 1)
		case "SREM":
			key := cmd[1].(string)
			delete(sets[key], cmd[2].(string))
			writeResult(w, 1)
		case "SMEMBERS":
			key := cmd[1].(string)
			var members []string
			for m := range sets[key] {
				members = append(members, m)
			}
			b, _ := json.Marshal(members)
			writeRaw(w, string(b))
		default:
			t.Fatalf("unexpected op %s", cmd[0])
		}
	}))
	t.Cleanup(srv.Close)
	return upstash.NewOfertaRepo(upstash.NewClient(srv.URL, "t", srv.Client())), srv
}

func assertDocIDs(t *testing.T, repo *upstash.OfertaRepo, produtoID domain.ProdutoID, want []domain.DocumentoID) {
	t.Helper()
	got, err := repo.ListDocumentoIDsByProduto(context.Background(), produtoID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("produto %s: want %v, got %v", produtoID, want, got)
	}
	set := map[domain.DocumentoID]struct{}{}
	for _, id := range got {
		set[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := set[id]; !ok {
			t.Fatalf("produto %s: missing %s in %v", produtoID, id, got)
		}
	}
}

func writeResult(w http.ResponseWriter, v any) {
	b, _ := json.Marshal(map[string]any{"result": v})
	w.Write(b)
}

func writeRaw(w http.ResponseWriter, resultJSON string) {
	w.Write([]byte(`{"result":` + resultJSON + `}`))
}
