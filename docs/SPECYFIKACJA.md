# UniQuest – specyfikacja techniczna i decyzje architektoniczne

| | |
|---|---|
| Projekt | **UniQuest** (Character Profiler) dla Politechniki Lubelskiej |
| Wersja dokumentu | 1.0 – 2026-10-17 |
| Wejście | `docs/BRIEF.md`, `docs/BAZA_WIEDZY.md` (fakty o PL ze źródłami) |
| Status | Decyzje podjęte. Warianty opisane tam, gdzie do celu prowadzi więcej niż jedna droga |

Konwencja: każda kluczowa decyzja ma format **Decyzja → Dlaczego → Wariant B (kiedy go wybrać)**.
Słowa „musi” i „powinien” znaczą odpowiednio wymaganie twarde i miękkie.

---

## 0. Decyzje w skrócie

| Obszar | Decyzja | Wariant B |
|---|---|---|
| Dystrybucja | **PWA** (Android + iOS + desktop) z jednej bazy kodu. Bez sklepów w MVP | Google Play przez TWA (Bubblewrap), gdy potrzebna obecność w sklepie; App Store przez Capacitor tylko na wyraźne żądanie klienta |
| Frontend | **SvelteKit + Svelte 5 + TypeScript (strict)**, `adapter-static`, własny service worker | React + Vite + `vite-plugin-pwa`, jeśli zespół zna tylko Reacta |
| UI | Własne komponenty na **tokenach Material Design 3** (kolor, typografia, kształt, elewacja, ruch). Bez ciężkiej biblioteki UI | Angular Material / MUI – odrzucone jako zbyt ciężkie |
| Animacje | Wbudowane `svelte/transition` i `svelte/motion` + **View Transitions API** + CSS. Tylko `transform`/`opacity` | – |
| Język | **TypeScript w 100% frontendu**, skryptów i narzędzi; typy API generowane z OpenAPI | – |
| Backend | **Go** (stdlib `net/http`, `pgx`, `sqlc`) – jeden statyczny plik binarny, który serwuje też frontend | Rust (axum) – odrzucony: przyrost wydajności bez znaczenia przy tej skali, dużo wolniejszy rozwój. Node/TS (Hono) – gdy zespół nie zna Go |
| Baza danych | **PostgreSQL 16+** | Supabase **wyłącznie jako hosting Postgresa**. Firebase odrzucony |
| Realtime | **Server-Sent Events** + Postgres `LISTEN/NOTIFY` (rozsyłanie między instancjami) | WebSocket (`coder/websocket`), gdyby pojawiła się gra wymagająca komunikacji klient→serwer w czasie rzeczywistym |
| Uwierzytelnianie | **Konto gościa od pierwszego uruchomienia** → opcjonalny upgrade: e-mail + 6-cyfrowy kod (bez haseł) → później Google / konto uczelniane Microsoft 365 | Passkeys (WebAuthn) w fazie 2 |
| Sesje | **Sesje po stronie serwera** (Postgres), cookie `__Host-` HttpOnly Secure SameSite=Lax. Bez JWT | – |
| QR obecności | **Rotujący kod HMAC** (zmiana co 30 s) wyświetlany przez stację. Statyczny wydruk jako tryb awaryjny | – |
| Skaner QR | Natywne **`BarcodeDetector`**, a gdy go brak (iOS, Firefox) – leniwie ładowany polyfill `barcode-detector` (zxing-wasm) | `jsQR` (mniejszy, wolniejszy) – wybór po pomiarze w M1 |
| Awatar | Filtr **pixel-art liczony na urządzeniu** (Canvas). Na serwer trafia tylko wynik ~64×64 px; zdjęcie nigdy nie opuszcza telefonu | Stylizacja AI po stronie serwera – poza MVP (koszt, RODO) |
| Baza wiedzy o PL | **Statyczne treści JSON w repozytorium** (wersjonowane, ze źródłem przy każdym fakcie), prerenderowane i dostępne offline | CMS headless, gdy treści ma edytować dział marketingu |
| Powiadomienia | **Web Push (VAPID)** + powiadomienia w aplikacji przez SSE | E-mail jako trzeci kanał |
| Hosting | **Jeden obraz Docker** (Go + wbudowany frontend) + Postgres. Pilot: VPS lub PaaS w UE. Produkcja: docelowo infrastruktura PL | Cloud Run (europe-central2) + Cloud SQL |
| Demo / PoC | **GitHub Pages w trybie demo** (cały frontend z backendem symulowanym w przeglądarce) + pełny stack przez `docker compose up` | Publiczny pełny stack na Fly.io (wymaga tokenu konta) |

---

## 1. Cel, użytkownicy, kontekst użycia

**Cel**: podczas wydarzeń na kampusie (Dzień Otwarty, Lubelski Festiwal Nauki, dni adaptacyjne) uczestnik
odwiedza stacje (wydziały, laboratoria, bibliotekę, halę sportową), potwierdza obecność kodem QR, ocenia,
jak bardzo mu się podobało, a aplikacja buduje z tego **kartę postaci RPG** i rekomenduje kierunki studiów
i koła naukowe PL. Poza wydarzeniami aplikacja jest **kompendium wiedzy o uczelni**.

| Persona | Potrzeba | Konsekwencja projektowa |
|---|---|---|
| Uczeń / kandydat (15–19 lat) | Zabawa, szybki start, wskazówka „co studiować” | Start bez rejestracji (gość), onboarding ≤ 60 s, minimalne dane osobowe (często niepełnoletni) |
| Student I roku | Orientacja na kampusie, koła naukowe, integracja | Mapa, gry zespołowe, kompendium |
| Gość / rodzic | Informacje o uczelni | Kompendium działa bez konta i offline |
| Obsługa stacji (staff) | Pokazać QR na koniec zajęć, zobaczyć frekwencję | Pełnoekranowy widok QR na tablecie lub rzutniku |
| Administrator (PL) | Konfiguracja wydarzeń i stacji, statystyki zainteresowania | Panel admina, eksport zanonimizowanych danych |

**Warunki brzegowe wydarzenia** (wpływają na architekturę):
- Tłum i przeciążone Wi-Fi lub LTE → **offline-first**: skan zapisuje się lokalnie i synchronizuje później, a treści są w cache.
- Szczyty ruchu: 100–300 osób skanuje ten sam kod w ciągu ~1 minuty po zakończeniu warsztatu.
- Szacunek obciążenia na wydarzenie: 2–5 tys. uczestników, do ~5 tys. równoczesnych połączeń SSE i ~300 skanów na minutę w szczycie. Go na 2 vCPU / 4 GB obsługuje to z dużym zapasem. Weryfikacja: test obciążeniowy k6 w M4.

---

## 2. Zakres

### MVP (M0–M3)
1. Onboarding: konto gościa, pseudonim, zgody, selfie → awatar pixel-art (do pominięcia), wybór motywu (opcjonalnie).
2. Panel bohatera: awatar, poziom/XP, 8 statystyk (na start wyzerowanych), postęp do karty (np. 3/5).
3. Skaner QR + ocena suwakiem + animacja przyrostu statystyk.
4. Silnik profilowania (archetypy, wagi, mnożniki ocen).
5. Karta postaci: odblokowanie po N skanach, opis procentowy, grafika PNG do pobrania i udostępnienia.
6. Rekomendacje: kierunki i koła naukowe PL dopasowane do profilu, z uzasadnieniem.
7. Zespoły: tworzenie, dołączanie przez QR lub kod, lista członków na żywo, stacje zespołowe, ranking.
8. Kompendium „Odkrywaj”: uczelnia, 6 wydziałów, ~30 kierunków, ~70 kół, kampus, życie studenckie, rekrutacja, wyszukiwarka.
9. Mapa kampusu (17 POI) i agenda wydarzenia (co, gdzie, kiedy).
10. Powiadomienia: przypomnienia o zapisanych zajęciach, zdarzenia zespołu, „karta gotowa”.
11. Panel stacji (staff) i podstawowy panel admina.
12. Upgrade konta gościa do e-maila (zachowanie postępu), eksport i usunięcie danych (RODO).

### Poza MVP (faza 2+)
Logowanie Google i Microsoft 365 (konto uczelniane) · passkeys · wersja EN (ponad 250 studentów zagranicznych) ·
publiczna strona karty `/c/{id}` z obrazkiem OG · stylizacja awatara AI · CMS dla marketingu · TWA w Google Play ·
odznaki sezonowe między wydarzeniami.

### Świadomie poza zakresem
Funkcje wirtualnego dziekanatu, mLegitymacji, planu zajęć – ma je istniejąca aplikacja **„Kampus Pollub”**.
UniQuest do niej linkuje i jej nie dubluje.

---

## 3. Platforma i dystrybucja

**Decyzja**: Progressive Web App instalowana z przeglądarki („Dodaj do ekranu głównego”), działająca też
bez instalacji jako zwykła strona.

**Dlaczego**:
- Na wydarzeniu liczą się sekundy: QR z plakatu otwiera aplikację **od razu**, bez sklepu i bez pobierania 50 MB przez przeciążone LTE.
- Jedna baza kodu, jedno wdrożenie, aktualizacje natychmiast (bez review w App Store).
- Wszystkie wymagane możliwości są dostępne w przeglądarce:

| Możliwość | Android (Chrome) | iOS/iPadOS (Safari ≥ 16.4) | Uwagi |
|---|---|---|---|
| Kamera (skaner) | tak (`getUserMedia`) | tak (`getUserMedia`, także w trybie standalone) | HTTPS wymagane. iOS potrafi ponownie pytać o zgodę po restarcie PWA (ryzyko UX, §16) |
| Dekodowanie QR | tak (`BarcodeDetector` natywnie) | brak → polyfill WASM | Polyfill ładowany leniwie i dodawany do cache |
| Selfie | tak (`<input type="file" accept="image/*" capture="user">`) | tak | Najprostsza i najbardziej niezawodna ścieżka (natywny aparat systemu) |
| Web Push | tak | tak, **tylko po instalacji na ekranie głównym** | Onboarding iOS pokazuje instrukcję instalacji |
| Instalacja | `beforeinstallprompt` (własny przycisk) | tylko ręcznie (Udostępnij → Do ekranu początkowego) | Własny ekran z instrukcją dla iOS |
| Udostępnianie karty | tak (Web Share API z plikami) | tak | Fallback: pobranie PNG |
| Offline | tak (Service Worker, Cache, IndexedDB) | tak | Background Sync nie działa w Safari → synchronizacja przy zdarzeniu `online` i przy starcie aplikacji |
| Wibracja (haptyka) | tak | nie | Tylko jako ozdobnik |

**Wariant B (sklepy)**:
- **Google Play**: ta sama PWA opakowana w **Trusted Web Activity** (Bubblewrap). Mały koszt, zero zmian w kodzie. Robić, jeśli klient chce ikony w sklepie.
- **App Store**: opakowanie Capacitorem. Ryzyko odrzucenia przez Apple za „cienki wrapper” (wytyczna 4.2), więc dodatkowe natywne funkcje byłyby konieczne. Rekomendacja: nie w MVP.

---

## 4. Architektura

```
             ┌─────────────────────── telefon uczestnika ───────────────────────┐
             │  PWA (SvelteKit, TS)                                              │
             │   ├─ UI (tokeny MD3, animacje)       ├─ IndexedDB: kolejka skanów │
             │   ├─ Service Worker: precache, offline, push                     │
             │   ├─ Skaner: BarcodeDetector | polyfill WASM                     │
             │   ├─ Awatar: Canvas pixel-art (lokalnie)                         │
             │   └─ ApiClient ── HttpApi (produkcja) | DemoApi (GitHub Pages)   │
             └───────────────┬───────────────────────────────▲──────────────────┘
                     HTTPS JSON (REST)                 SSE (/api/v1/stream)
                             │                               │
             ┌───────────────▼───────────────────────────────┴──────────────────┐
             │  uniquest (jeden plik binarny Go, jeden obraz Docker)              │
             │   ├─ /            statyczny frontend (go:embed)                   │
             │   ├─ /api/v1/...  handlery REST (net/http, wzorce routingu 1.22+) │
             │   ├─ auth/sesje · game (scoring) · qr (HMAC) · teams · push       │
             │   ├─ realtime hub (SSE) ◄── LISTEN/NOTIFY                         │
             │   └─ content (JSON wbudowany: kierunki, koła, wektory wag)        │
             └───────────────┬──────────────────────────────────────────────────┘
                             │ pgx
                      ┌──────▼──────┐        ┌──────────────────────────┐
                      │ PostgreSQL  │        │ SMTP (kody logowania)    │
                      └─────────────┘        │ Web Push (FCM/APNs/Moz.) │
                                             └──────────────────────────┘
```

Zasady:
1. **Jedno źródło (same-origin)**: frontend i API pod tą samą domeną (`/` i `/api`). Bez CORS, proste cookies, prosty CSP.
2. **Serwer jest autorytatywny** dla punktów, walidacji QR i zespołów. Klient liczy punkty tylko dla natychmiastowej animacji (optimistic UI).
3. **Logika gry to czyste funkcje** (bez I/O) w Go i w TS, sprawdzane tymi samymi wspólnymi przypadkami testowymi (golden fixtures, §12).
4. **Kontrakt najpierw**: `packages/contracts/openapi.yaml` jest źródłem prawdy. Typy TS generuje `openapi-typescript`, handlery Go waliduje test kontraktowy.

### Struktura repozytorium

```
apps/
  web/                 SvelteKit PWA (TypeScript)
    src/lib/design/    tokeny MD3 (CSS custom properties), komponenty bazowe
    src/lib/game/      scoring.ts, archetypes.ts (czyste funkcje)
    src/lib/api/       ApiClient, HttpApi, DemoApi
    src/lib/scanner/   kamera, BarcodeDetector + polyfill
    src/lib/avatar/    pixelizacja (Canvas)
    src/routes/        ekrany (§6)
    src/service-worker.ts
  api/                 Go module
    cmd/uniquest/      main: serve | migrate | seed | gen-vapid
    internal/{http,auth,game,qr,teams,push,realtime,store,content}
    migrations/        SQL (goose)
    queries/           SQL dla sqlc
packages/
  contracts/           openapi.yaml, wygenerowane typy TS, fixtures/scoring/*.json
content/               treści bazy wiedzy (JSON) + JSON Schema
tools/kb-refresh/      skrypt TS: odświeżanie treści przez Tavily → diff do przeglądu
deploy/                Dockerfile, compose.yaml, Caddyfile
docs/
```

Bez narzędzi monorepo (Nx, Turborepo). Wystarczą `npm workspaces` dla części TS i osobny moduł Go. Jedna komenda: `make dev`, `make test`, `make build`.

---

## 5. Frontend

### 5.1 Stack

**Decyzja**: SvelteKit (aktualna stabilna linia: Kit 3.x, Svelte 5.x, Vite 8, TypeScript; wersje przypięte w lockfile), `@sveltejs/adapter-static` z fallbackiem SPA i prerenderingiem stron kompendium.

**Dlaczego**:
- Svelte kompiluje komponenty do małego JS bez wirtualnego DOM, więc ma najmniejszy runtime z popularnych frameworków. To realnie wpływa na start aplikacji na średnim telefonie przez słabe LTE.
- **Przejścia i animacje sprężynowe są wbudowane** (`svelte/transition`, `svelte/motion`). Płynne animacje bez bibliotek typu GSAP czy Framer Motion.
- **Service worker jest wspierany natywnie** (`$service-worker`: lista plików builda do precache), więc Workbox nie jest potrzebny.
- Pełne wsparcie TypeScript (`svelte-check`), dojrzała dokumentacja, standardowe narzędzia (Vite, Vitest, Playwright).

**Wariant B**: React 19 + Vite + `vite-plugin-pwa` – większa pula programistów, ale większy runtime i animacje wymagałyby dodatkowej biblioteki. Architektura (ApiClient, tokeny, SW, czyste funkcje gry) przenosi się 1:1.

### 5.2 TypeScript – zakres
- 100% kodu frontendu, service workera, skryptów narzędziowych i testów w TS (`strict: true`, `noUncheckedIndexedAccess: true`).
- Typy DTO API generowane z OpenAPI (zakaz ręcznego pisania typów odpowiedzi).
- Treści kompendium walidowane JSON Schemą podczas builda (z JSON Schemy powstają też typy TS).
- Backend **nie** jest w TS (patrz §7). Dzielimy kontrakt (OpenAPI) i przypadki testowe (fixtures), a nie kod.

### 5.3 Dozwolone zależności (allowlist)
Runtime (trafia do przeglądarki):
| Pakiet | Po co | Uwagi |
|---|---|---|
| `svelte`, `@sveltejs/kit` | framework | – |
| `barcode-detector` | polyfill standardowego `BarcodeDetector` (zxing-wasm) | ładowany dynamicznie **tylko**, gdy brak natywnego API |
| Generator QR Nayuki (`qrcodegen`, plik TS dołączony do repo) | rysowanie QR zaproszeń i stacji | referencyjna implementacja, bez zależności |

Dev: `typescript`, `vite`, `vitest`, `@playwright/test`, `svelte-check`, `eslint`, `prettier`, `openapi-typescript`.
Wszystko inne (IndexedDB, i18n, wyszukiwarka, router, ikony, stan) piszemy sami, bo kod jest mały i prosty.
Dodanie nowej zależności runtime wymaga uzasadnienia w PR.

### 5.4 Design system – „jasny, przyjazny, material z nutą RPG”
- **Tokeny MD3** jako CSS custom properties: role kolorów (`--md-sys-color-primary`, `on-primary`, `surface`, `surface-container-*`…), skala typografii (display/headline/title/body/label), kształty (corner 4/8/12/16/28 px), elewacja (1–5), stany (hover/focus/pressed jako warstwa).
- **Paleta** generowana raz (Material Theme Builder / `material-color-utilities` uruchomione jednorazowo, wynik zapisany jako CSS – nie jest zależnością runtime) z koloru bazowego z **księgi znaku PL** (do pozyskania od Centrum Komunikacji i Marketingu; roboczo granat uczelniany). Motyw **jasny jako domyślny**, ciemny dostępny.
- **Typografia**: Roboto Flex (UI) + font pixelowy **tylko** na nagłówki i liczby statystyk (kandydaci: Pixelify Sans, Silkscreen – warunek: pełne polskie znaki, licencja OFL). Fonty self-hosted, subset latin + latin-ext, `font-display: swap`.
- **Ikony**: Material Symbols (Apache 2.0) jako **sprite SVG tylko z użytymi ikonami** (nie cały font). 8 ikon atrybutów rysujemy sami w pixel-art (SVG).
- **Elementy RPG** dawkowane: segmentowane paski statystyk, ramka karty postaci, pixelowy awatar, „toast” z +XP. Reszta interfejsu to czytelny Material.
- **Komponenty bazowe** (własne, ok. 20): Button (filled/tonal/text), IconButton, FAB, Card, NavigationBar, TopAppBar, Slider, Chip, Dialog, BottomSheet, Snackbar, TextField, Switch, ProgressBar (liniowy, segmentowy), List, Tabs, Avatar, Badge, Skeleton.

### 5.5 Ruch i animacje
Zgodnie z MD3 motion:
| Token | Wartość | Użycie |
|---|---|---|
| `--motion-easing-standard` | `cubic-bezier(0.2, 0, 0, 1)` | większość przejść |
| `--motion-easing-emphasized-decelerate` | `cubic-bezier(0.05, 0.7, 0.1, 1)` | wejścia elementów |
| `--motion-easing-emphasized-accelerate` | `cubic-bezier(0.3, 0, 0.8, 0.15)` | wyjścia elementów |
| `--motion-duration-short` | 100–200 ms | stany, przyciski, chipy |
| `--motion-duration-medium` | 250–400 ms | przejścia ekranów, arkusze |
| `--motion-duration-long` | 450–600 ms | wyłącznie odsłonięcie karty i level-up |

Reguły: animujemy tylko `transform` i `opacity` (kompozytor GPU, brak reflow). Przejścia ekranów przez
**View Transitions API** (progresywnie – bez wsparcia po prostu brak animacji). Liczniki statystyk przez
`svelte/motion` `Tween`. **`prefers-reduced-motion: reduce` wyłącza wszystko poza zmianą opacity.**
Animacja nigdy nie blokuje interakcji. Budżet: 60 fps na telefonie klasy Moto G.

### 5.6 Budżety wydajności (sprawdzane w CI)
- JS początkowy (powłoka aplikacji) ≤ **80 KB gzip**, każda trasa ≤ 50 KB gzip. Polyfill skanera poza budżetem (ładowany warunkowo).
- Lighthouse mobile (profil Moto G Power / Slow 4G): Performance ≥ 95, Accessibility ≥ 95, Best Practices ≥ 95.
- LCP ≤ 2,0 s · INP ≤ 200 ms · CLS ≤ 0,05.
- Pixelizacja awatara ≤ 300 ms. Od wykrycia QR do ekranu oceny ≤ 300 ms (bez czekania na sieć – optymistycznie).

### 5.7 Offline i service worker
- **Precache**: powłoka aplikacji, fonty, sprite ikon, treści kompendium (JSON + prerenderowane strony), mapa SVG; polyfill skanera dociągany w tle po pierwszym uruchomieniu na przeglądarkach bez `BarcodeDetector`.
- **API GET** (profil, agenda): network-first z fallbackiem do cache.
- **Kolejka skanów**: skan offline trafia do IndexedDB `{payload, scannedAtClient, rating}` i jest wysyłany przy `online`, przy starcie aplikacji i co 30 s. UI pokazuje „Zapisano – wyślemy, gdy wróci zasięg”.
- Aktualizacja SW: nowa wersja czeka, a snackbar „Dostępna nowa wersja → Odśwież” pojawia się dopiero poza ekranem skanera.

### 5.8 Dostępność
Wymaganie twarde: **WCAG 2.1 AA** – PL jako podmiot publiczny podlega ustawie o dostępności cyfrowej (konieczna jest też deklaracja dostępności). W praktyce: kontrast ≥ 4.5:1, pełna obsługa czytnikiem ekranu (etykiety, role ARIA, kolejność fokusa), cele dotyku ≥ 48 px, suwak oceny obsługiwany klawiaturą i czytnikiem, alternatywa dla QR (ręczne wpisanie kodu), brak informacji przekazywanej wyłącznie kolorem, testy `axe` w Playwright.

---

## 6. Ekrany i nawigacja

Dolny **Navigation Bar (MD3)**: **Bohater** · **Zadania** · **[Skanuj]** (wyróżniony, środek) · **Drużyna** · **Odkrywaj**.

| Trasa | Ekran | Kluczowe elementy |
|---|---|---|
| `/` | Start | Gość → onboarding; zalogowany → `/hero` |
| `/onboarding` | 4 kroki | 1) Witaj (czym jest UniQuest) 2) Pseudonim + zgody 3) Selfie → podgląd pixel-art (lub gotowy awatar) 4) „Twoje statystyki są na zero – ruszaj na kampus!” + instrukcja instalacji (iOS/Android) |
| `/hero` | Panel bohatera | Awatar, poziom i pasek XP, 8 statystyk (paski, wartości), „Do karty postaci: 3/5”, ostatnie aktywności, oczekujące oceny |
| `/scan` | Skaner | Pełny ekran kamery, ramka celownika, latarka (jeśli dostępna), „Wpisz kod ręcznie”, obsługa braku zgody na kamerę |
| `/scan/rate/[scanId]` | Ocena | Nazwa stacji, suwak 1–5 z etykietami, „Zatwierdź” → animacja przyrostu statystyk, +XP, ewentualny level-up |
| `/quests` | Zadania i agenda | „Teraz / Za chwilę / Później”, filtr wydziału, oznaczenie odwiedzonych, „Przypomnij mi” (push), przejście do mapy |
| `/map` | Mapa kampusu | Własna ilustrowana mapa SVG (17 POI), stan stacji, „Nawiguj” (link `geo:` / Mapy Google / Apple) |
| `/team` | Drużyna | Utwórz / Dołącz (skan lub kod) / Moja drużyna: członkowie na żywo, QR zaproszenia, ranking |
| `/card` | Karta postaci | Zablokowana do N skanów (licznik), odsłonięcie z animacją, opis procentowy, „Pobierz PNG” / „Udostępnij” |
| `/recommendations` | Rekomendacje | Top 3 kierunki + top 3 koła naukowe, każdy z „Dlaczego?” (wkład statystyk) i linkiem do kompendium oraz rekrutacji |
| `/explore/...` | Kompendium | Uczelnia · Wydziały · Kierunki · Koła naukowe · Kampus · Życie studenckie · Rekrutacja · „Czy wiesz, że…” · wyszukiwarka |
| `/settings` | Ustawienia | Upgrade konta (e-mail), powiadomienia, motyw, ograniczenie ruchu, eksport i usunięcie danych, informacje prawne |
| `/staff/...` | Obsługa stacji | Lista moich stacji, **widok QR na pełny ekran** (rotacja co 30 s, licznik skanów na żywo), otwórz/zamknij okno skanowania |
| `/admin/...` | Administracja | Wydarzenia, stacje (wagi atrybutów, tryb zespołowy), sesje/agenda, role, statystyki, eksport CSV |

---

## 7. Backend

### 7.1 Język: Go

**Decyzja**: Go (aktualna wersja stabilna; środowisko ma 1.24+).

**Dlaczego Go, a nie Rust**:
- Profil obciążenia (setki zapytań na sekundę, tysiące połączeń SSE) to dla Go trywialny poziom. Rust dałby lepszą latencję w mikrosekundach, której nikt nie zauważy, kosztem wolniejszego rozwoju, dłuższej kompilacji i stromej krzywej nauki.
- **Biblioteka standardowa wystarcza na prawie wszystko**: HTTP/2, routing z metodami i parametrami (`net/http` 1.22+), `crypto/hmac`, `crypto/rand`, `log/slog`, `embed`, `html/template`, `encoding/json`, `net/smtp`. To spełnia wymóg „minimum zewnętrznych bibliotek”.
- Goroutine na połączenie SSE to idiomatyczny i tani model.
- Jeden statyczny plik binarny (z wbudowanym frontendem) w obrazie distroless ~20 MB, start < 100 ms, łatwe wdrożenie gdziekolwiek.

**Wariant B**: Node.js + Hono + TypeScript – jedyny argument to wspólny język z frontendem. Wybrać, jeśli w zespole nie ma nikogo, kto czuje się pewnie w Go.

### 7.2 Dozwolone zależności Go
| Moduł | Po co |
|---|---|
| `github.com/jackc/pgx/v5` | sterownik PostgreSQL (standard de facto), pula połączeń, LISTEN/NOTIFY |
| `sqlc` (narzędzie, generuje kod) | typowane zapytania z czystego SQL – bez ORM |
| `github.com/pressly/goose/v3` | migracje SQL (wbudowane przez `embed`) |
| `github.com/SherClockHolmes/webpush-go` | szyfrowanie i wysyłka Web Push (RFC 8291/8292) |
| `golang.org/x/time/rate` | limity zapytań (token bucket) |
| *(faza 2)* `golang.org/x/oauth2`, `github.com/coreos/go-oidc/v3` | logowanie Google / Microsoft |

### 7.3 Uwierzytelnianie i sesje
1. **Gość**: pierwsze uruchomienie → `POST /api/v1/auth/guest` tworzy użytkownika `kind=guest` z losowym pseudonimem (np. „Dzielny Konstruktor 4821”) i sesję. Zero danych osobowych.
2. **Upgrade e-mail** (opcjonalny – „zapisz postęp, odbierz kartę na maila”): `POST /auth/email/start {email}` → 6-cyfrowy kod (ważny 10 min, maks. 5 prób, hash SHA-256 w bazie) → `POST /auth/email/verify` → e-mail dopięty do **tego samego** użytkownika, więc postęp zostaje. Logowanie na nowym urządzeniu tą samą ścieżką.
3. **Social login** (faza 2): Google (OIDC); **konto uczelniane Microsoft 365** dla obecnych studentów – wymaga rejestracji aplikacji przez Centrum Informatyczne PL. Facebook pomijamy (kosztowny review aplikacji Meta, niska wartość).
4. **Sesja**: token 32 losowe bajty → cookie `__Host-uq_session` (HttpOnly, Secure, SameSite=Lax, Path=/). W bazie trzymamy tylko `SHA-256(token)`. Wygaśnięcie przesuwane: 30 dni, odnawiane przy aktywności (zapis maks. raz na godzinę). Wylogowanie i „wyloguj wszędzie” kasują rekordy.
5. **CSRF**: SameSite=Lax + kontrola nagłówka `Origin` na wszystkich mutacjach + wymagany nagłówek `X-Requested-With: uniquest`.
6. **Role**: `participant`, `staff` (przypisany do stacji), `admin`. Rolę nadaje admin, a konto staff musi mieć zweryfikowany e-mail.

### 7.4 Kody QR

**A. QR obecności (stacja)** – anty-oszustwo bez uciążliwości:
- Ekran stacji (`/staff/stations/{id}/display`) co 30 s pobiera z serwera token i rysuje QR. Tablet nie zna sekretu.
- Treść QR to URL (działa też ze zwykłego aparatu telefonu):
  `https://{host}/q/1.{stationId}.{window}.{mac}`
  gdzie `window = floor(unix_time / 30)`, `mac = base64url(HMAC-SHA256(stationSecret, "1|" + stationId + "|" + window))[:12]`.
- Akceptujemy `window ∈ {teraz, teraz−1}` (≤ 60 s). Skany z kolejki offline: do **10 min** wstecz (konfigurowalne per wydarzenie, z flagą `queued`).
- Dodatkowo: stacja musi mieć **otwarte okno skanowania** (staff klika „Otwórz skanowanie” na koniec zajęć lub działa harmonogram sesji), a para **(użytkownik, stacja) jest unikalna** (ograniczenie w bazie).
- Porównanie MAC w czasie stałym (`hmac.Equal`).
- **Tryb awaryjny „static”** (wydruk na plakacie bez tabletu): `https://{host}/q/s.{stationId}.{sessionId}.{mac}`. Ważny tylko w oknie sesji i oznaczany niższym poziomem zaufania. Gdyby za ranking były nagrody – tryb statyczny wyłączyć.
- Ręczne wpisanie: 6-znakowy kod pod QR (alfabet bez 0/O/1/I) mapowany na bieżące okno.

**B. QR zaproszenia do drużyny**:
- Kapitan generuje zaproszenie: `https://{host}/t/{code}`, `code` = 8 znaków (~40 bitów), ważny 15 min, maks. liczba użyć = wolne miejsca. Pod QR ten sam kod do przepisania.
- Dołączenie: skan w aplikacji (lub aparatem → link otwiera PWA lub przeglądarkę → po zalogowaniu jako gość automatycznie dołącza).

**Skaner w kliencie**: `getUserMedia({video:{facingMode:'environment'}})` → klatki do `BarcodeDetector` (natywnie lub polyfill) w pętli `requestVideoFrameCallback` z limitem ~10 analiz na sekundę. Rozpoznaje tylko nasze prefiksy URL (`/q/`, `/t/`), a inne kody odrzuca z komunikatem.

### 7.5 API (REST, JSON, prefiks `/api/v1`)

| Metoda i ścieżka | Opis |
|---|---|
| `POST /auth/guest` | utwórz gościa + sesję |
| `POST /auth/email/start` · `POST /auth/email/verify` | kod jednorazowy |
| `POST /auth/logout` · `POST /auth/logout-all` | – |
| `GET /me` · `PATCH /me` | profil (pseudonim, zgody, preferencje) |
| `PUT /me/avatar` · `DELETE /me/avatar` | PNG ≤ 16 KB, ≤ 128×128 |
| `GET /me/export` · `DELETE /me` | RODO: eksport JSON, usunięcie konta |
| `GET /events/active` | aktywne wydarzenie + konfiguracja (min. skanów, skala ocen) |
| `GET /events/{id}/stations` · `GET /events/{id}/schedule` | stacje, agenda |
| `POST /scans` `{payload, scannedAt?, queued?}` | walidacja QR → `{scanId, station, needsRating}` (idempotentne dla tej samej pary) |
| `PUT /scans/{id}/rating` `{rating: 1..5}` | zapis oceny → zwraca nowe statystyki i delty |
| `GET /me/progress` | statystyki, XP, poziom, lista skanów, oczekujące oceny |
| `GET /me/character` | archetypy, procenty, tytuł, rekomendacje, `unlocked` |
| `POST /teams` · `GET /teams/{id}` | utwórz / pobierz drużynę |
| `POST /teams/{id}/invites` | nowe zaproszenie (kod, URL, ważność) |
| `POST /teams/join` `{code}` · `POST /teams/{id}/leave` · `DELETE /teams/{id}/members/{uid}` | członkostwo |
| `GET /events/{id}/leaderboard?type=team\|solo` | ranking (cache 5 s) |
| `GET /push/vapid-public-key` · `POST /push/subscriptions` · `DELETE /push/subscriptions` | Web Push |
| `POST /schedule/{sessionId}/remind` · `DELETE …` | przypomnienia o zajęciach |
| `GET /stream` | **SSE**: `team.member_joined`, `team.member_left`, `team.score`, `notification`, `scan.confirmed` |
| `GET /staff/stations` · `GET /staff/stations/{id}/qr` · `POST /staff/stations/{id}/window` · `GET /staff/stations/{id}/stats` | panel stacji |
| `/admin/...` | CRUD wydarzeń, stacji, sesji, ról; `GET /admin/events/{id}/analytics.csv` |

Błędy: `application/problem+json` (RFC 9457) z kodem domenowym (`qr_expired`, `already_scanned`, `scan_window_closed`, `team_full`…), mapowanym w UI na przyjazne komunikaty.

### 7.6 Realtime
- `GET /api/v1/stream` (SSE, `text/event-stream`). Hub w pamięci: mapa `userID → kanały`. Heartbeat co 25 s, `Last-Event-ID` przy wznowieniu.
- Zdarzenia z bazy (dołączenie do drużyny, skan członka) → `pg_notify('uq_events', json)`. Każda instancja słucha `LISTEN` i rozsyła do swoich klientów. Działa przy skalowaniu poziomym bez Redisa.

### 7.7 Powiadomienia
- **W aplikacji**: przez SSE + lista w `/hero`.
- **Web Push**: klucze VAPID (`uniquest gen-vapid`). Prośba o zgodę **dopiero po akcji użytkownika** („Przypomnij mi” przy zajęciach), nigdy przy starcie. Rodzaje: przypomnienie 10 min przed zapisaną sesją (worker co minutę), „karta postaci odblokowana”, „ktoś dołączył do drużyny”, komunikat organizatora (broadcast z panelu admina, z limitem).
- Wygasłe subskrypcje (HTTP 404/410) usuwamy automatycznie.

### 7.8 Bezpieczeństwo – lista kontrolna
CSP: `default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'` · HSTS · `Permissions-Policy: camera=(self)` ·
limity zapytań: `auth/email/start` 3/min/IP i 10/h/e-mail, `scans` 20/min/użytkownik, globalnie per IP · limity rozmiaru body · walidacja wejścia na granicy (pseudonimy: długość, filtr wulgaryzmów, Unicode NFC) ·
sekrety stacji 32 B losowe, rotowalne · zapytania tylko parametryzowane (sqlc) · logi bez danych osobowych (`slog`, ID zamiast e-maili) · `govulncheck` i `npm audit` w CI · przegląd bezpieczeństwa przed pilotem.

---

## 8. Mechanika profilowania

### 8.1 Atrybuty i archetypy (8)

| Kod | Statystyka w UI | Archetyp | Typowe źródła punktów (stacje) |
|---|---|---|---|
| `STR` | Siła | **Konstruktor** | Wydział Mechaniczny, CEN-TECH |
| `INT` | Inteligencja | **Analityk** | WMiIT (matematyka, dane), WZ (finanse) |
| `MAN` | Mana | **Technomag** | WEiI (informatyka, elektrotechnika), WMiIT (cyber) |
| `KRE` | Kreatywność | **Architekt** | WBiA, Wzornictwo przemysłowe, multimedia |
| `ENE` | Energia | **Strażnik Energii** | WIŚiE (energetyka, OZE, środowisko) |
| `CHA` | Charyzma | **Strateg** | Wydział Zarządzania, Rektorat/administracja |
| `WIS` | Mądrość | **Badacz** | Biblioteka (CIZT), laboratoria badawcze |
| `VIT` | Witalność (HP) | **Atleta** | Centrum Sportowe, gry zespołowe |

To realizuje mapowanie z briefu (Mechaniczny → mechaniczny/inżynieryjny, Informatyka → technologiczny/logiczny,
Biblioteka/Administracja → research/korporacyjny, Hala → witalność/praca zespołowa) w 8 osiach, które dają czytelny
wykres radarowy (ośmiokąt). Lista atrybutów jest **konfiguracją**, nie kodem.

### 8.2 Konfiguracja stacji
Każda stacja ma `base_points` (domyślnie 100) i **wektor wag** `w` (suma = 1), np.:
- Warsztat „Druk 3D i obróbka CNC” (WM): `STR 0.7, INT 0.2, KRE 0.1`
- Pokaz „Cyberatak na żywo” (WMiIT): `MAN 0.6, INT 0.4`
- Biblioteka CIZT: `WIS 0.6, CHA 0.4`
- Centrum Sportowe (zespołowa): `VIT 0.7, CHA 0.3`

Wagi ustala admin razem z wydziałem. Panel admina pokazuje podgląd „jak zmieni się profil przy ocenie 1…5”.

### 8.3 Wzór
Ocena `r ∈ {1,2,3,4,5}` („Na ile to było dla Ciebie fajne?”: *Nie dla mnie / Tak sobie / Ciekawe / Bardzo fajne / Chcę to robić!*).

```
f(r) = M[r],  domyślnie M = [0.25, 0.5, 1.0, 1.5, 2.0]   (konfigurowalne per wydarzenie)
Δ_k  = base_points × w_k × f(r) × team_bonus             (team_bonus = 1.2 dla skanu zespołowego, inaczej 1)
S_k  = Σ Δ_k po wszystkich ocenionych skanach            (statystyka k)
XP   = Σ_k S_k ;   poziom = 1 + floor( sqrt(XP / 25) )
```

To realizuje brief („suma punktów z lokalizacji pomnożona przez subiektywną ocenę”), przy czym tablica `M`
zamiast samego `r` daje 8-krotną różnicę między „nie dla mnie” a „chcę to robić”. Słabo oceniona wizyta
prawie nie przesuwa profilu, ale nadal liczy się do progu karty. Wariant: `M = [1,2,3,4,5]` (ściśle liniowo) – jedna zmiana konfiguracji.

Skan bez oceny nie daje punktów (UI przypomina „Oceń 2 aktywności”). Ocenę można zmienić w ciągu 15 min.

### 8.4 Wynik – opis procentowy
1. `p_k = S_k / Σ S`.
2. Wybierz archetypy z `p_k ≥ 15%`, maks. 3, min. 1 (najwyższy).
3. Renormalizuj do 100% i zaokrąglij do pełnych 5% metodą największych reszt (suma = 100).
4. Tytuł hybrydowy z pary top-2 (tabela w `content/archetypes.json`, np. Konstruktor+Technomag = **„Mechatronik”**, Konstruktor+Architekt = **„Mistrz Formy”**, Analityk+Strateg = **„Strateg Liczb”**, Technomag+Strażnik Energii = **„Inżynier Sieci Przyszłości”**). Brak pary w tabeli → „Konstruktor-Analityk”.
5. Opis: *„Twoja ścieżka to w 70% Konstruktor, w 30% Analityk”* + 2–3 zdania z szablonów archetypów.

**Przykład** (5 skanów): WM r=5 (`STR .7 INT .2 KRE .1`), WEiI r=3 (`MAN .6 INT .4`), Hala r=4 (`VIT .7 CHA .3`, skan solo – bez bonusu),
Biblioteka r=2 (`WIS .6 CHA .4`), WBiA r=4 (`KRE .7 STR .3`)
→ STR 185, KRE 125, VIT 105, INT 80, CHA 65, MAN 60, WIS 30, ENE 0 (Σ 650, XP 650, poziom 6)
→ ≥15%: STR 28,5%, KRE 19,2%, VIT 16,2% → **45% Konstruktor, 30% Architekt, 25% Atleta**, tytuł „Mistrz Formy”.
(Ten przypadek jest pierwszym golden fixture.)

### 8.5 Rekomendacje
Każdy kierunek i każde koło naukowe w `content/` ma wektor atrybutów `v` (np. Mechatronika: `STR .45 MAN .35 INT .2`;
Architektura: `KRE .7 STR .15 ENE .15`; Cyberbezpieczeństwo: `MAN .55 INT .45`; Finanse i rachunkowość: `INT .55 CHA .45`).
Dopasowanie = **podobieństwo cosinusowe** `cos(S, v)`. Pokazujemy top 3 kierunki i top 3 koła z procentem dopasowania
i wyjaśnieniem „Dlaczego?” (dwa atrybuty o największym wkładzie `S_k·v_k`). Wektory kierunków ustala się z wydziałami
(punkt wyjścia: przedmioty maturalne i opisy „Co po studiach?” z `rekrutacja.pollub.pl`).

### 8.6 Karta postaci
- Odblokowanie: `min_scans_for_card` ocenionych skanów (domyślnie **5**).
- Grafika rysowana na `<canvas>` w kliencie, format 1080×1350 (pion, social media): awatar, pseudonim, tytuł, poziom, radar 8 statystyk, opis %, logo PL i UniQuest, data i wydarzenie.
- `Udostępnij` → `navigator.share({files:[png]})`, fallback `Pobierz PNG`. Faza 2: publiczny link z obrazkiem OG.

### 8.7 Gry zespołowe
- Drużyna: 2–5 osób (konfigurowalne), nazwa + emblemat, kapitan. Jedna drużyna na użytkownika na wydarzenie.
- **Stacja zespołowa** (`team_mode = true`, `team_min_members = k`): skan członka jest „wstrzymany”, dopóki co najmniej `k` członków nie zeskanuje tej stacji w oknie sesji. Wtedy wszyscy dostają `team_bonus = 1.2` i drużyna dostaje odznakę. SSE pokazuje na żywo „2/3 członków zeskanowało”.
- Wynik drużyny = suma XP członków + bonusy. Ranking drużyn i solo w `/team` (opcjonalnie wyświetlany na ekranie w holu).

### 8.8 Odznaki (lekkie, konfigurowalne)
„Odkrywca wydziałów” (wszystkie 6) · „Mól książkowy” (Biblioteka) · „Duch drużyny” (3 stacje zespołowe) · „Ranny ptaszek” (pierwsza sesja dnia) · „Kartograf” (wszystkie POI na mapie).

---

## 9. Baza wiedzy o PL (moduł „Odkrywaj”)

**Decyzja**: treści jako **pliki JSON w repo** (`content/`), walidowane JSON Schemą, wbudowane w aplikację
(prerender + precache, czyli działa offline i zero obciążenia backendu) oraz w binarkę Go (wektory do rekomendacji).

Struktura:
```
content/
  university.json      fakty, historia, władze, wyróżniki („Czy wiesz, że…”)
  faculties/*.json     6 wydziałów: nazwa, skrót, adres, POI, opis, kontakt rekrutacyjny, lista kierunków i kół
  programs/*.json      ~30 kierunków: opis, poziomy/tryby, matura, „Co po studiach”, wektor v, link rekrutacji
  clubs/*.json         ~70 kół naukowych: nazwa, wydział, tematyka, tagi, wektor v (bez nazwisk opiekunów)
  campus.json          17 POI: współrzędne na mapie SVG, adres, typ
  student-life.json    akademiki, organizacje, kultura, sport
  recruitment.json     ERK, terminy, linki
  archetypes.json      atrybuty, archetypy, tytuły hybrydowe, szablony opisów
```
Każdy rekord ma pola `sources: [{url, title}]` i `verifiedAt`. UI pokazuje „Źródło: pollub.pl · zweryfikowano 2026-10”.
Materiał wyjściowy (zweryfikowany): `docs/BAZA_WIEDZY.md`.

**Wyszukiwarka**: indeks budowany podczas builda (tokeny znormalizowane: małe litery, bez polskich znaków diakrytycznych, prefiksy).
Przy ~150 dokumentach własna implementacja to ~80 linii, bez biblioteki.

**Odświeżanie treści** (`tools/kb-refresh`, TS): Tavily Map/Extract ograniczone do domen `*.pollub.pl` →
porównanie z `content/` → raport różnic → **PR do przeglądu przez człowieka** (nigdy automatyczna publikacja).
Klucz API tylko w sekretach CI lub lokalnie. Zgodnie z dobrymi praktykami Tavily: `include_domains`, ekstrakcja ukierunkowana
zapytaniem (`query` + chunki), zapis źródła i daty przy każdym fakcie, surowe wyniki trzymane poza repo.

**Wariant B**: headless CMS (np. Directus na tym samym Postgresie), gdy treści mają edytować osoby z marketingu bez Gita.

---

## 10. Awatar (selfie → pixel-art)

Na urządzeniu, w `src/lib/avatar/` (Canvas 2D, bez bibliotek):
1. Zdjęcie z `<input capture="user">` → `createImageBitmap` → przycięcie do kwadratu (środek, opcjonalnie ręczne przesunięcie).
2. Zmniejszenie do 48×48 (wygładzanie włączone) → korekta kontrastu i nasycenia.
3. Kwantyzacja do **stałej palety 16–24 kolorów** (spójny styl wszystkich awatarów, odcienie skóry w palecie) + opcjonalny dithering Floyda–Steinberga (przełącznik).
4. Kontur 1 px (wykrywanie krawędzi alfa po usunięciu tła – w MVP tło zastępowane jednolitym kolorem klasy przez prosty próg luminancji; segmentacja AI poza MVP).
5. Powiększenie ×N metodą najbliższego sąsiada (`imageSmoothingEnabled = false`) → PNG ~2–6 KB.
6. Ramka klasy (wybór 1 z 8, zmienia się po odblokowaniu karty – „ewolucja postaci”).

Na serwer trafia **tylko** wynikowy PNG (≤ 16 KB, trzymany w Postgresie jako `bytea`, bez S3). Oryginał jest odrzucany.
Alternatywa bez selfie: kreator z gotowych elementów (fryzura, kolor, akcesoria) – ważne dla osób, które nie chcą zdjęcia.

---

## 11. Model danych (PostgreSQL)

```sql
users(id uuid pk, kind text check (kind in ('guest','registered')), nickname text, email citext unique null,
      email_verified_at timestamptz, role text default 'participant', avatar_png bytea, avatar_frame text,
      consents jsonb, prefs jsonb, created_at, last_active_at, deleted_at)
sessions(token_hash bytea pk, user_id fk, created_at, last_seen_at, expires_at, user_agent text)
login_codes(id pk, email citext, code_hash bytea, expires_at, attempts int, consumed_at)
events(id pk, slug unique, name, starts_at, ends_at, status, min_scans_for_card int default 5,
       rating_multipliers numeric[] default '{0.25,0.5,1,1.5,2}', queued_scan_grace interval default '10 min',
       team_min int default 2, team_max int default 5)
stations(id pk, event_id fk, slug, name, description, faculty_code text null, poi_id text, kind text,
         base_points int default 100, weights jsonb, team_mode bool, team_min_members int,
         qr_secret bytea, qr_mode text check (qr_mode in ('rotating','static')), staff_user_ids uuid[])
station_sessions(id pk, station_id fk, title, room, starts_at, ends_at, scan_opens_at, scan_closes_at)
scans(id pk, event_id, user_id, station_id, session_id null, scanned_at, received_at, queued bool,
      trust text, rating smallint null check (rating between 1 and 5), rated_at, deltas jsonb,
      team_id null, team_status text null, unique(user_id, station_id))
teams(id pk, event_id, name, emblem, captain_id, locked bool, created_at)
team_members(team_id fk, user_id fk, event_id, joined_at, primary key(team_id,user_id), unique(event_id,user_id))
team_invites(code pk, team_id fk, created_by, expires_at, max_uses, uses)
push_subscriptions(id pk, user_id fk, endpoint text unique, p256dh text, auth text, created_at, last_ok_at)
session_reminders(user_id, session_id, notify_at, sent_at, primary key(user_id, session_id))
notifications(id pk, user_id null, event_id, kind, payload jsonb, created_at, read_at)
badges_awarded(user_id, event_id, badge, awarded_at, primary key(user_id, event_id, badge))
audit_log(id pk, actor_id, action, target, at, meta jsonb)       -- działania staff/admin
```
Statystyki i postać **liczone na żądanie** z `scans.deltas` (to tanie: ≤ ~30 skanów na osobę). Rankingi cache'owane w pamięci przez 5 s.
Indeksy: `scans(event_id, station_id)`, `scans(user_id)`, `team_members(event_id, user_id)`, `sessions(user_id)`.

---

## 12. Jakość i testy

| Poziom | Narzędzie | Zakres |
|---|---|---|
| Logika gry | Go `testing` + Vitest | **te same golden fixtures** `packages/contracts/fixtures/scoring/*.json` (wejście: skany i konfiguracja → oczekiwane S, %, tytuł, rekomendacje). Gwarancja zgodności Go↔TS |
| Backend | Go `testing` + prawdziwy Postgres (lokalnie `initdb`, w CI usługa `postgres`) | handlery, QR (okna czasu, replay, sfałszowany MAC), zespoły, sesje, limity |
| Kontrakt | test porównujący trasy Go z `openapi.yaml` | brak dryfu API |
| Frontend | Vitest | pixelizacja, kolejka offline, wyszukiwarka, i18n |
| E2E | Playwright (Chromium, emulacja Pixel/iPhone) | onboarding → skan → ocena → karta; drużyna z 2 kontekstów przeglądarki; **skan z „fałszywej kamery”** (`--use-fake-device-for-media-stream` + wideo z QR); `axe` dla a11y |
| Wydajność | Lighthouse CI, rozmiar bundla | budżety z §5.6 |
| Obciążenie | k6 | 5 tys. SSE + 300 skanów/min, p95 < 200 ms |
| Urządzenia | ręcznie | Safari iOS (kamera, instalacja, push), Chrome Android, słaba sieć |

CI (GitHub Actions): lint → typecheck → testy Go + TS → build → e2e → Lighthouse → obraz Docker → deploy demo (Pages).

---

## 13. Prywatność (RODO) i kwestie prawne

- **Minimalizacja**: gość = losowe ID + pseudonim + skany. E-mail opcjonalny. Selfie nie opuszcza urządzenia.
- **Niepełnoletni**: uczniowie mogą mieć < 16 lat. Tryb gościa nie wymaga danych identyfikujących. Podanie e-maila poprzedza oświadczenie o wieku ≥ 16 lat lub zgoda opiekuna. **Do potwierdzenia z Inspektorem Ochrony Danych PL**.
- **Role prawne**: administratorem danych jest PL; zespół deweloperski jako podmiot przetwarzający (umowa powierzenia). Hosting w UE.
- **Retencja**: konta gości nieaktywne 12 miesięcy są usuwane. Analityka dla uczelni tylko zagregowana (np. średnia ocena i liczba wizyt per stacja, rozkład archetypów).
- **Prawa użytkownika**: eksport (`GET /me/export`) i usunięcie (`DELETE /me`) samoobsługowo.
- Polityka prywatności, regulamin gry i deklaracja dostępności jako strony w aplikacji. Brak trackerów zewnętrznych; analityka produktowa (jeśli potrzebna) tylko first-party i zagregowana.

---

## 14. Hosting i wdrożenie

**Artefakt**: jeden obraz Docker (multi-stage: `node` buduje frontend → `golang` kompiluje binarkę z `go:embed` → `distroless/static`).
Konfiguracja przez zmienne środowiskowe (`DATABASE_URL`, `BASE_URL`, `SMTP_*`, `VAPID_*`, `SESSION_*`). Endpointy `/healthz` i `/readyz`.

| Wariant | Opis | Zalety | Wady | Kiedy |
|---|---|---|---|---|
| **A. Infrastruktura PL** | VM w Centrum Informatycznym, Docker Compose (app + Postgres + Caddy), domena np. `uniquest.pollub.pl` | dane u administratora, marka uczelni, brak kosztów chmury | zależność od terminów IT uczelni | **produkcja docelowa** |
| **B. VPS w UE** | 2 vCPU / 4 GB, Docker Compose + Caddy (automatyczny HTTPS), backup `pg_dump` co noc + kopia poza serwerem | tanio (rząd kilku–kilkunastu EUR/mies.), pełna kontrola | administracja po naszej stronie | **pilot**, jeśli A nie zdąży |
| **C. PaaS** | Fly.io (region Warszawa) z Postgresem zarządzanym albo Cloud Run (europe-central2) + Cloud SQL | wdrożenie z GitHub Actions jednym poleceniem, autoskalowanie | koszt rośnie z ruchem, vendor | publiczne demo full-stack, pilot |

Rekomendacja: **pilot na B lub C → produkcja na A**. Ten sam obraz działa wszędzie. Przed wydarzeniem: test k6, skalowanie w górę na dzień wydarzenia, wyłączenie deployów (freeze) w dniu wydarzenia.
Supabase może pełnić rolę zarządzanego Postgresa w wariantach B/C (używamy wtedy tylko bazy, bez Supabase Auth i Realtime).

---

## 15. Demo / proof of concept

Frontend ma interfejs `ApiClient` z dwiema implementacjami wybieranymi podczas builda (`PUBLIC_API_MODE`):
- `HttpApi` – prawdziwy backend Go.
- `DemoApi` – „backend w przeglądarce”: IndexedDB + te same czyste funkcje gry (`src/lib/game`) + dane z `content/`. Symuluje stacje, okna QR i boty-członków drużyny.

Dzięki temu udostępniamy:
1. **Publiczne klikalne demo na GitHub Pages**: `https://git-maks.github.io/PL-App/`. Pełna ścieżka: onboarding → awatar → skan → ocena → karta → rekomendacje → drużyna → kompendium. Działa na telefonie (HTTPS, kamera). Zawiera stronę **„Arkusz stacji testowych”** z QR do wydruku lub wyświetlenia na drugim ekranie. Wymaga jednorazowo: *Settings → Pages → Source: GitHub Actions* w repozytorium.
2. **Pełny stack lokalnie**: `docker compose up` → `http://localhost:8080` (Postgres + API + frontend, dane seed, „skrzynka mailowa” deweloperska zamiast SMTP).
3. **Publiczny pełny stack** (opcjonalnie): workflow deploy na Fly.io po dodaniu sekretu `FLY_API_TOKEN` (lub na VPS przez SSH).

---

## 16. Ryzyka i otwarte pytania

| Ryzyko | Wpływ | Mitygacja |
|---|---|---|
| iOS: ponowne pytanie o zgodę na kamerę w PWA, push tylko po instalacji | tarcie UX | ręczne wpisanie kodu, instrukcja instalacji, push jako dodatek, nie podstawa |
| Przeciążona sieć na wydarzeniu | utrata skanów | offline-first, kolejka, mały bundle, precache |
| Oszustwa (zdjęcie QR wysłane znajomym) | zafałszowany ranking | rotacja 30 s, okna skanowania, unikalność; nagrody tylko za rzeczy niewymagające twardej weryfikacji |
| Rozmiar polyfilla WASM na iOS | wolny pierwszy skan | precache w tle po onboardingu; wariant jsQR po pomiarze |
| Niepełnoletni i RODO | prawne | tryb gościa, konsultacja z IOD |
| Treści się dezaktualizują | błędne informacje | `verifiedAt`, `kb-refresh` + przegląd człowieka, właściciel treści po stronie PL |
| Kalibracja wag | „dziwne” wyniki | podgląd w adminie, golden fixtures, test na grupie pilotażowej |

**Pytania do klienta (PL)**:
1. Księga znaku i zgoda na użycie logo; kolor bazowy.
2. Domena i hosting: czy Centrum Informatyczne udostępni VM (wariant A)?
3. Docelowe wydarzenie pilotażowe, lista stacji, kto ustala wagi z wydziałami.
4. Czy przewidziane są nagrody? (decyduje o poziomie zabezpieczeń anty-oszustwo).
5. Logowanie kontem uczelnianym (Microsoft 365) dla studentów – czy IT PL zarejestruje aplikację?
6. Wersja angielska w MVP czy w fazie 2?
7. Kto po stronie PL jest właścicielem treści kompendium?

---

## 17. Plan realizacji (kamienie milowe)

| Etap | Zakres | Kryterium ukończenia |
|---|---|---|
| **M0 Fundamenty** | szkielet repo, CI, OpenAPI v0, tokeny MD3 i komponenty bazowe, powłoka PWA (manifest, SW, ikony), deploy demo na Pages | Lighthouse ≥ 95, instalowalna PWA na Androidzie i iOS |
| **M1 Pętla gry (demo)** | onboarding, awatar, panel bohatera, skaner + ocena, scoring + golden fixtures, karta + share, rekomendacje, kompendium z pełnymi treściami, mapa, agenda – wszystko na `DemoApi` | pełna ścieżka klikalna na telefonie z publicznego linku |
| **M2 Backend** | Go API, Postgres, migracje, sesje, gość i e-mail, QR HMAC + panel stacji, kolejka offline, `HttpApi`, Docker Compose | e2e na prawdziwym backendzie zielone, zgodność fixtures Go↔TS |
| **M3 Zespoły i realtime** | drużyny, zaproszenia QR, SSE, stacje zespołowe, ranking, Web Push, przypomnienia | e2e drużyny w 2 przeglądarkach, push na Androidzie i zainstalowanym iOS |
| **M4 Admin i gotowość produkcyjna** | panel admina, eksport analityki, RODO (eksport/usunięcie), audyt a11y, test k6, przegląd bezpieczeństwa, instrukcja wdrożenia | gotowość do pilota |
| **Pilot** | wydarzenie na kampusie | zebrane metryki i feedback → faza 2 |

---

## 18. Definicja „gotowe” (dla każdej funkcji)
Kod w TS/Go zgodny z lintem i typami · testy jednostkowe i (dla ścieżek użytkownika) e2e · zgodność z OpenAPI ·
dostępność (axe bez błędów, obsługa klawiaturą) · budżet wydajności zachowany · działa offline tam, gdzie to ma sens ·
teksty po polsku bez literówek · zaktualizowana dokumentacja.

---

## 19. Stan realizacji (2026-10-17) i odstępstwa od pierwotnego planu

Zrealizowane etapy M0–M3 oraz większość M4 (szczegóły w `README.md` i `docs/WDROZENIE.md`):

| Obszar | Stan |
|---|---|
| PWA (SvelteKit 3, Svelte 5, TS strict) | wszystkie ekrany z §6, MD3 + animacje, offline, instalacja, service worker |
| Backend Go + PostgreSQL | całe API z OpenAPI (test zgodności tras), sesje, gość + e-mail z łączeniem kont, QR HMAC, okna skanowania, drużyny, SSE, Web Push, przypomnienia, analityka, CSV, RODO |
| Logika gry | identyczna w TS i Go – 7 golden fixtures (statystyki, poziomy, archetypy, tytuły, rekomendacje) + wektory QR |
| Testy | Vitest (25), Go z PostgreSQL i `-race`, Playwright demo (8, w tym kamera i axe/WCAG) i full-stack (2) |
| Demo | tryb `DemoApi` + workflow GitHub Pages |
| Wdrożenie | Dockerfile (distroless), Compose + Caddy, Makefile, CI, k6 |

Świadome odstępstwa (uzasadnione prostotą i minimalną liczbą zależności):

1. **Zapytania SQL pisane jawnie w `pgx`** zamiast generowanych przez `sqlc` – przy ~60 zapytaniach skupionych w `internal/app`
   koszt dodatkowego narzędzia (generator z kompilacją C) przewyższał korzyść. `sqlc` można dołożyć bez zmian schematu.
2. **Własny migrator (~60 linii)** zamiast `goose` – wbudowane pliki SQL, transakcja na migrację, blokada doradcza.
3. **Bonus drużynowy**: punkty za stację zespołową naliczane są od razu, a bonus ×1,2 dopisywany całej drużynie w chwili
   osiągnięcia kworum (zamiast „wstrzymania” skanu) – gracz od razu widzi postęp, a wynik końcowy jest ten sam.
4. **SvelteKit 3**: aliasy `#lib/*` (subpath imports) zamiast `$lib`, moduły `$app/manifest`/`$app/env` w service workerze,
   CSP oparta na hashach generowanych przez SvelteKit + nagłówki bezpieczeństwa z Go.
5. **Mapa kampusu**: realne współrzędne budynków z OpenStreetMap/Nominatim rzutowane na ilustrowaną mapę SVG; położenie kilku
   obiektów bez adresu w OSM (hala sportowa, stołówka, Spichlerz) jest przybliżone i oznaczone w danych (`approximate: true`).
6. **Kolor marki**: roboczy granat `#1f5aa6` (księga znaku PL nie jest publicznie dostępna) – zmiana w jednym pliku tokenów.

Do zrobienia po stronie uczelni / w kolejnej fazie: logowanie Google i Microsoft 365, wersja EN, publiczna strona karty z
obrazkiem OG, testy na fizycznych urządzeniach iOS/Android, konsultacja RODO z IOD, właściwe wagi stacji z wydziałami.
