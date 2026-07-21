package upstash

import store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"

// Client is re-exported from modules/ofertas-store so scraper and API share one
// Upstash Redis REST implementation.
type Client = store.Client

var NewClient = store.NewClient
