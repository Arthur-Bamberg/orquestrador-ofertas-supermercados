module github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper

go 1.26.0

require (
	github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2 v0.0.0
	github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/stretchr/testify v1.12.1
	golang.org/x/image v0.28.0
	google.golang.org/genai v1.64.0
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.9.3 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas v0.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fergusstrange/embedded-postgres v1.34.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas v0.0.0 => ../agente-ofertas

replace github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2 v0.0.0 => ../ofertas-scraper-v2

replace github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store v0.0.0 => ../../modules/ofertas-store
