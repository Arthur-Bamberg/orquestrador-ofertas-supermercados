package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestMercadoHandlers(t *testing.T) {
	catalog, closeServer := newTestCatalog(t)
	defer closeServer()
	handler := New(catalog, config.Config{CORSOrigin: "http://localhost:5173"})

	req := httptest.NewRequest(http.MethodPost, "/api/mercados", strings.NewReader(`{"id":"m1","nome":"Mercado Um"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("POST status=%d body=%s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/mercados/m1", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", res.Code, res.Body.String())
	}
	var got store.Mercado
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "m1" || got.Nome != "Mercado Um" {
		t.Fatalf("got %+v", got)
	}
}

func newTestCatalog(t *testing.T) (*store.Catalog, func()) {
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
			for member := range sets[cmd[1].(string)] {
				members = append(members, member)
			}
			b, _ := json.Marshal(members)
			writeRedisRaw(w, string(b))
		case "KEYS":
			var keys []string
			pattern := cmd[1].(string)
			for key := range values {
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
	return store.NewCatalog(store.NewClient(srv.URL, "token", srv.Client())), srv.Close
}

func writeRedisResult(w http.ResponseWriter, v any) {
	b, _ := json.Marshal(map[string]any{"result": v})
	_, _ = w.Write(b)
}

func writeRedisRaw(w http.ResponseWriter, resultJSON string) {
	_, _ = w.Write([]byte(`{"result":` + resultJSON + `}`))
}
