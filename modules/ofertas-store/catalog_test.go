package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"
)

func TestChaveUnicaOfertaStable(t *testing.T) {
	got := ChaveUnicaOferta(Oferta{
		ProdutoID: "p1", MercadoID: "m1", Valor: 9.99,
		Quantidades: []float64{500}, Medida: MedidaG,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-10",
	})
	const want = "eab46759feb7d43630072fa68d815308e0820ed6017286bd3dda8076d837cfd5"
	if got != want {
		t.Fatalf("hash changed: got %s want %s", got, want)
	}
}

func TestDeleteGuards(t *testing.T) {
	catalog, closeServer := newTestCatalog(t)
	defer closeServer()
	ctx := context.Background()

	if err := catalog.SaveMercado(ctx, Mercado{ID: "m1", Nome: "Mercado"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFonte(ctx, Fonte{ID: "f1", MercadoID: "m1", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteMercado(ctx, "m1"); !errors.Is(err, ErrDeleteBlocked) {
		t.Fatalf("DeleteMercado err=%v", err)
	}

	if err := catalog.SaveDocumento(ctx, Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteFonte(ctx, "f1"); !errors.Is(err, ErrDeleteBlocked) {
		t.Fatalf("DeleteFonte err=%v", err)
	}

	if err := catalog.SaveProduto(ctx, Produto{ID: "p1", Nome: "Arroz", NomeNorm: "arroz"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveMarca(ctx, Marca{ID: "ma1", Nome: "Boa", NomeNorm: "boa"}); err != nil {
		t.Fatal(err)
	}
	marcaID := MarcaID("ma1")
	oferta := Oferta{
		ID: "o1", ProdutoID: "p1", MarcaID: &marcaID, MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: MedidaUnidade, DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	if err := catalog.SaveOferta(ctx, oferta, []DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteProduto(ctx, "p1"); !errors.Is(err, ErrDeleteBlocked) {
		t.Fatalf("DeleteProduto err=%v", err)
	}
	if err := catalog.DeleteMarca(ctx, "ma1"); !errors.Is(err, ErrDeleteBlocked) {
		t.Fatalf("DeleteMarca err=%v", err)
	}
}

func TestDeleteDocumentoUnlinksAndDeletesOrphanOferta(t *testing.T) {
	catalog, closeServer := newTestCatalog(t)
	defer closeServer()
	ctx := context.Background()

	if err := catalog.SaveDocumento(ctx, Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	oferta := Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: MedidaUnidade, DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	if err := catalog.SaveOferta(ctx, oferta, []DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFalhasDocumento(ctx, "d1", []FalhaExtracao{{Codigo: "x"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteDocumento(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := catalog.GetOferta(ctx, "o1"); err != nil || ok {
		t.Fatalf("orphan oferta ok=%v err=%v", ok, err)
	}
	if falhas, err := catalog.GetFalhasDocumento(ctx, "d1"); err != nil || len(falhas) != 0 {
		t.Fatalf("falhas=%v err=%v", falhas, err)
	}
}

func newTestCatalog(t *testing.T) (*Catalog, func()) {
	t.Helper()
	storeData := map[string]string{}
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
			storeData[cmd[1].(string)] = cmd[2].(string)
			writeRedisResult(w, "OK")
		case "GET":
			v, ok := storeData[cmd[1].(string)]
			if !ok {
				writeRedisRaw(w, "null")
				return
			}
			writeRedisResult(w, v)
		case "DEL":
			for i := 1; i < len(cmd); i++ {
				key := cmd[i].(string)
				delete(storeData, key)
				delete(sets, key)
			}
			writeRedisResult(w, 1)
		case "SADD":
			key := cmd[1].(string)
			if sets[key] == nil {
				sets[key] = map[string]struct{}{}
			}
			sets[key][cmd[2].(string)] = struct{}{}
			writeRedisResult(w, 1)
		case "SREM":
			delete(sets[cmd[1].(string)], cmd[2].(string))
			writeRedisResult(w, 1)
		case "SMEMBERS":
			var members []string
			for member := range sets[cmd[1].(string)] {
				members = append(members, member)
			}
			b, _ := json.Marshal(members)
			writeRedisRaw(w, string(b))
		case "KEYS":
			var keys []string
			pattern := cmd[1].(string)
			for key := range storeData {
				if ok, _ := path.Match(pattern, key); ok {
					keys = append(keys, key)
				}
			}
			for key := range sets {
				if ok, _ := path.Match(pattern, key); ok {
					keys = append(keys, key)
				}
			}
			b, _ := json.Marshal(keys)
			writeRedisRaw(w, string(b))
		default:
			t.Fatalf("unexpected op %s", cmd[0])
		}
	}))
	return NewCatalog(NewClient(srv.URL, "token", srv.Client())), srv.Close
}

func writeRedisResult(w http.ResponseWriter, v any) {
	b, _ := json.Marshal(map[string]any{"result": v})
	_, _ = w.Write(b)
}

func writeRedisRaw(w http.ResponseWriter, resultJSON string) {
	_, _ = w.Write([]byte(`{"result":` + resultJSON + `}`))
}
