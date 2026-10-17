# UniQuest — wdrożenie i eksploatacja

## 1. Artefakt

Jeden obraz Docker (`deploy/Dockerfile`): binarka Go z wbudowaną aplikacją webową, uruchamiana jako użytkownik
bez uprawnień na obrazie distroless. Wymaga tylko PostgreSQL 16. Migracje wykonują się automatycznie przy starcie
(`serve`) albo ręcznie (`migrate`) — z blokadą doradczą, więc kilka instancji może startować równocześnie.

Podkomendy: `serve`, `migrate`, `seed [-date YYYY-MM-DD] [-activate] [-admin e-mail]`, `gen-vapid`, `cleanup`, `health`.

## 2. Konfiguracja (zmienne środowiskowe)

| Zmienna | Domyślnie | Opis |
|---|---|---|
| `DATABASE_URL` | `postgres://uniquest@127.0.0.1:5432/uniquest?sslmode=disable` | połączenie z PostgreSQL |
| `UQ_ADDR` | `:8080` | adres nasłuchu |
| `UQ_BASE_URL` | `http://localhost:8080` | publiczny adres (trafia do kodów QR i zaproszeń; używany do kontroli `Origin`) |
| `UQ_COOKIE_SECURE` | `true` | cookie `__Host-` + `Secure` + HSTS. `false` tylko dla lokalnego http |
| `UQ_TRUST_PROXY` | `false` | ufaj `X-Forwarded-For` (włącz za Caddy/nginx) |
| `UQ_SESSION_TTL` | `720h` | przesuwana ważność sesji |
| `UQ_SMTP_HOST`, `UQ_SMTP_PORT`, `UQ_SMTP_USER`, `UQ_SMTP_PASSWORD`, `UQ_SMTP_FROM` | — / 587 | wysyłka kodów logowania |
| `UQ_DEV_LOGIN_CODES` | `false` | **tylko dev/demo**: kod logowania wraca w odpowiedzi API zamiast e-maila |
| `UQ_VAPID_PUBLIC`, `UQ_VAPID_PRIVATE`, `UQ_VAPID_SUBJECT` | — | Web Push (`uniquest gen-vapid`); bez kluczy push jest wyłączony, powiadomienia działają w aplikacji |
| `UQ_OPEN_ALL_STATIONS` | `false` | **tylko testy**: wszystkie stacje przyjmują skany niezależnie od harmonogramu |
| `UQ_RATE_LIMITS` | `true` | **tylko testy obciążeniowe** mogą je wyłączyć |
| `UQ_LOG_LEVEL` | `info` | logi JSON na stdout (bez danych osobowych) |
| `UQ_VERSION` | `dev` | wersja w `/api/v1/healthz` |

## 3. Warianty hostingu (spec §14)

### A. Infrastruktura Politechniki Lubelskiej (docelowo)
VM z Dockerem w Centrum Informatycznym, domena np. `uniquest.pollub.pl`, certyfikat uczelniany lub Caddy.
Kroki jak w wariancie B; zmienne SMTP — serwer pocztowy uczelni.

### B. VPS w UE (pilot)
```bash
git clone … && cd PL-App
cat > deploy/.env <<'ENV'
UQ_DOMAIN=uniquest.example.pl
UQ_BASE_URL=https://uniquest.example.pl
UQ_COOKIE_SECURE=true
UQ_TRUST_PROXY=true
UQ_DEV_LOGIN_CODES=false
POSTGRES_PASSWORD=<losowe-hasło>
UQ_SMTP_HOST=… UQ_SMTP_USER=… UQ_SMTP_PASSWORD=… UQ_SMTP_FROM="UniQuest <uniquest@example.pl>"
UQ_VAPID_PUBLIC=… UQ_VAPID_PRIVATE=…
ENV
docker compose -f deploy/compose.yaml --env-file deploy/.env --profile https up -d --build
docker compose -f deploy/compose.yaml --env-file deploy/.env run --rm app seed -date 2027-04-01 -admin organizator@example.pl
```
Caddy sam pobiera certyfikat Let's Encrypt i nie buforuje SSE (`flush_interval -1`).

### C. PaaS (np. Fly.io w regionie Warszawa, Cloud Run europe-central2)
Ten sam obraz; zarządzany PostgreSQL (Fly Postgres, Cloud SQL, Supabase — tylko jako baza). Ustaw zmienne z §2,
health check na `/api/v1/readyz`. Przy wielu instancjach realtime działa przez `LISTEN/NOTIFY` bez dodatkowego brokera.

## 4. Wydarzenie

1. Utwórz wydarzenie: `uniquest seed -date RRRR-MM-DD -activate` (z `content/demo-event.json` — edytuj stacje, sesje i wagi
   razem z wydziałami; ponowny seed zachowuje identyfikatory i sekrety stacji, więc wydrukowane plakaty pozostają ważne).
2. Nadaj role: konto organizatora loguje się e-mailem, admin w `/admin` → *Uprawnienia* (rola *Obsługa stacji*), a w edycji
   stacji przypisuje e-maile obsługi (`staffEmails` w API).
3. Na każdej stacji: tablet/rzutnik z `/staff` → *Wyświetl kod QR* (rotacja co 30 s, blokada wygaszania ekranu, kod ręczny).
   Okno skanowania otwiera się automatycznie 10 min przed końcem sesji i trwa do 15 min po niej; obsługa może je otworzyć lub
   zamknąć ręcznie.
4. Po wydarzeniu: `/admin` → analityka i eksport CSV (zagregowany, anonimowy).

## 5. Checklista przed wydarzeniem
- [ ] Test obciążeniowy na środowisku docelowym: `k6 run tools/load/k6-event-peak.js` (z `UQ_RATE_LIMITS=false` tylko na czas testu).
- [ ] Skalowanie w górę na dzień wydarzenia (≥ 2 vCPU / 4 GB), zamrożenie wdrożeń.
- [ ] Kopia zapasowa bazy (`pg_dump` co noc + przed wydarzeniem), test odtworzenia.
- [ ] SMTP: test wysyłki kodu; VAPID: test powiadomienia na Androidzie i na zainstalowanej PWA na iPhonie.
- [ ] Zasięg Wi-Fi/LTE w salach; aplikacja działa offline (kolejka skanów), ale stacje potrzebują sieci do wyświetlania kodów.
- [ ] Plakaty z kodem statycznym dla stacji bez ekranu: `/staff` → *Plakat do druku* (API: `GET /api/v1/staff/stations/{id}/poster`); kod działa tylko w oknie skanowania.
- [ ] Deklaracja dostępności i polityka prywatności zatwierdzone przez PL i IOD.

## 6. Utrzymanie
- Retencja: serwer codziennie usuwa konta gości nieaktywne > 12 miesięcy oraz wygasłe sesje i kody (`cleanup` ręcznie).
- Logi: JSON na stdout; `/api/v1/readyz` pokazuje stan bazy i liczbę połączeń SSE.
- Treści bazy wiedzy: aktualizacja plików w `content/` (każdy rekord ma `sources` i `verifiedAt`) → PR → nowy build.
