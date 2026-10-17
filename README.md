# UniQuest — Politechnika Lubelska

Progressive Web App, która podczas wydarzeń na kampusie Politechniki Lubelskiej buduje profil kompetencyjny
uczestnika w formie karty postaci RPG i rekomenduje kierunki studiów oraz koła naukowe. Poza wydarzeniami jest
kompendium wiedzy o uczelni (6 wydziałów, 31 kierunków, 65 kół naukowych, mapa kampusu).

- [Brief](Brief%20Aplikacja.md) · [Specyfikacja i decyzje](docs/SPECYFIKACJA.md) · [Baza wiedzy (źródła)](docs/BAZA_WIEDZY.md) · [Wdrożenie](docs/WDROZENIE.md)

## Co jest w środku

| Część | Technologia | Gdzie |
|---|---|---|
| Aplikacja (PWA) | SvelteKit 3 + Svelte 5 + TypeScript, tokeny Material Design 3, własny service worker | `apps/web` |
| Backend | Go (stdlib `net/http`, `pgx`), PostgreSQL 16, SSE + `LISTEN/NOTIFY`, Web Push (VAPID) | `apps/api` |
| Kontrakt API | OpenAPI 3.1 → typy TS (`openapi-typescript`), test zgodności tras w Go | `packages/contracts` |
| Wspólne testy logiki | golden fixtures punktacji i wektory QR — sprawdzane w TS **i** w Go | `packages/contracts/fixtures` |
| Treści | JSON z bazą wiedzy (źródło przy każdym fakcie), wbudowane w aplikację i w binarkę | `content` |
| Wdrożenie | jeden obraz Docker (Go + wbudowany frontend), Compose, Caddy | `deploy` |

Frontend ma dwa „backendy” za wspólnym interfejsem `ApiClient`: **HttpApi** (prawdziwy serwer Go) i **DemoApi**
(cała logika serwera w przeglądarce, z prawdziwą weryfikacją HMAC kodów QR) — dzięki temu publiczne demo działa na
GitHub Pages bez serwera.

## Szybki start

Wymagania: Node 22.17+, Go (wersja z `go.mod`), PostgreSQL 16 (lokalnie lub `make dev-db` z Dockerem).

```bash
make install            # zależności frontendu
make dev-db             # (opcjonalnie) PostgreSQL w Dockerze
make dev-api            # API na :8080, wydarzenie demo na dziś, admin: admin@pollub.test
make dev-web            # aplikacja na http://localhost:5173 (proxy /api → :8080)
```

Samo demo bez backendu: `cd apps/web && PUBLIC_API_MODE=demo npm run dev`.

Pełny stack w kontenerach:

```bash
docker compose -f deploy/compose.yaml up --build -d
docker compose -f deploy/compose.yaml run --rm app seed -admin admin@pollub.test
# → http://localhost:8080 (kody logowania pokazują się w aplikacji: UQ_DEV_LOGIN_CODES=true)
```

Logowanie jako organizator: w aplikacji *Profil → Dodaj e-mail* → `admin@pollub.test` → kod. Panel stacji
(`/staff`) wyświetla rotujący kod QR, a w `/admin` jest analityka, edycja wag stacji i komunikaty.

## Testy

```bash
make test           # svelte-check + Vitest + go vet + testy Go (z PostgreSQL)
make e2e            # Playwright na buildzie demo: onboarding, skan przez symulowaną kamerę, karta, drużyny, axe (WCAG)
make e2e-fullstack  # Playwright na prawdziwym backendzie Go (panel stacji → kod → ocena; plakat; drużyna na żywo przez SSE)
```

CI (`.github/workflows/ci.yml`) uruchamia to wszystko oraz budżet rozmiaru JS, `govulncheck` i budowę obrazu Docker.
Demo publikuje się na GitHub Pages workflowem `pages.yml` (jednorazowo: *Settings → Pages → Source: GitHub Actions*).

## Struktura

```
apps/web/            SvelteKit PWA
  src/lib/game/      silnik punktacji (czyste funkcje, zgodne 1:1 z Go)
  src/lib/api/       ApiClient: http.ts (Go) | demo/ (w przeglądarce)
  src/lib/scanner/   kamera + BarcodeDetector / zxing-wasm
  src/lib/avatar/    selfie → pixel-art (na urządzeniu)
  src/lib/card/      karta postaci (Canvas) + Web Share
  src/routes/        ekrany (bohater, skaner, zadania, mapa, drużyna, kompendium, staff, admin)
  e2e/, e2e-fullstack/  testy Playwright
apps/api/            backend Go
  cmd/uniquest/      serve | migrate | seed | gen-vapid | cleanup | health
  internal/app/      przypadki użycia (konta, skany, drużyny, powiadomienia, staff, admin)
  internal/game, qr/ logika gry i kodów QR (testowana wspólnymi fixtures)
  internal/httpapi/  REST + SSE + serwowanie frontendu
  migrations/        schemat SQL
content/             baza wiedzy (JSON)
packages/contracts/  OpenAPI + fixtures
deploy/              Dockerfile, compose.yaml, Caddyfile
tools/load/          test obciążeniowy k6
```
