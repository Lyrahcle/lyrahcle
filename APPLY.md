# Bloque 5 — Caché de incidents / asset-alerts / RF

## Qué hace

Añade caché de respuesta (3s TTL, Redis, aislada por tenant) para los
endpoints que el dashboard sondea cada 8s y que hasta ahora golpeaban
Postgres en cada poll, sin caché:

- `GET /api/v1/incidents`
- `GET /api/v1/asset-alerts`
- `GET /api/v1/rf/readings`
- `GET /api/v1/rf/anomalies`

Con invalidación explícita de la vista por defecto (sin filtro, primera
página — la que pide el dashboard) tras:

- `POST /api/v1/incidents` (crear)
- `PATCH /api/v1/incidents/{id}` (actualizar estado/severidad/asignación)
- `POST /api/v1/asset-alerts/{id}/acknowledge`
- `POST /api/v1/rf/ingest`

Sigue exactamente el mismo patrón ya usado en el repo para satélites y
CVEs (`internal/respcache`).

## Ficheros

Modificados:
- `apps/backend/cmd/server/main.go` — caché + invalidación en los 4
  endpoints GET y sus 4 escrituras asociadas; nuevas funciones
  `incidentsCacheKey`, `incidentsCacheKeyDefault`, `assetAlertsCacheKey`,
  `assetAlertsCacheKeyDefault`, `rfReadingsCacheKey`,
  `rfAnomaliesCacheKey`
- `docs/improvement-roadmap.md`, `CHANGELOG.md`

Nuevos:
- `apps/backend/cmd/server/response_cache_v143_test.go` — verifica
  aislamiento por tenant, diferenciación por filtro/paginación, y que la
  clave "default" coincide con lo que pide el frontend (`PAGE_LIMIT=50`,
  offset 0)

## Aplicar en tu repo local

```bash
cd ~/ruta/a/lyrahcle
rsync -a /ruta/al/zip-extraido/ ./
git status
git diff --stat
```

## Verificar

```bash
cd apps/backend
go build ./...
go vet ./...
TEST_DATABASE_URL= TEST_REDIS_ADDR= go test ./...
TEST_DATABASE_URL= TEST_REDIS_ADDR= go test -race ./...
```

Ya lo corrí antes de empaquetar (tuve que instalar Go 1.26 en el
contenedor porque el módulo lo exige): build y vet limpios, 32 paquetes
en verde con `-race` incluido, y los 6 tests nuevos de contrato de caché
pasando.

## Limitación conocida (documentada también en el test)

`respcache.Delete` borra una clave exacta, no un patrón. Por eso solo se
invalida la vista "sin filtro, primera página" tras cada escritura — el
resto de combinaciones de filtro/offset dependen del TTL de 3s para
acotar la desactualización, igual que ya hace el caché de satélites
existente en este mismo repo.
