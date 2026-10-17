# UniQuest — common tasks. Requires Node 22, Go (see go.mod) and PostgreSQL 16 (or Docker).
WEB := apps/web
DIST := apps/api/internal/webui/dist
BIN := bin/uniquest
export DATABASE_URL ?= postgres://uniquest@127.0.0.1:5432/uniquest?sslmode=disable

.PHONY: help install dev-db dev-api dev-web build-web build-demo build test test-web test-api e2e e2e-fullstack docker gen

help:
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  %-16s %s\n", $$1, $$2}'

install: ## Install web dependencies
	cd $(WEB) && npm ci

dev-db: ## Start PostgreSQL 16 in Docker for development
	docker run -d --name uniquest-db -e POSTGRES_USER=uniquest -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=uniquest -p 5432:5432 postgres:16-alpine

dev-api: ## Run the Go API on :8080 (dev settings, demo event seeded for today)
	go run ./apps/api/cmd/uniquest seed -admin admin@pollub.test
	UQ_COOKIE_SECURE=false UQ_DEV_LOGIN_CODES=true UQ_BASE_URL=http://localhost:5173 go run ./apps/api/cmd/uniquest serve

dev-web: ## Run the web app with hot reload on :5173 (proxies /api to :8080)
	cd $(WEB) && npm run dev

build-web: ## Build the web app for the Go server
	cd $(WEB) && PUBLIC_API_MODE=http npm run build

build-demo: ## Build the standalone demo (in-browser backend), e.g. for GitHub Pages
	cd $(WEB) && PUBLIC_API_MODE=demo npm run build

build: build-web ## Build the single binary with the web app embedded
	mkdir -p $(DIST) && cp -r $(WEB)/build/. $(DIST)/
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./apps/api/cmd/uniquest

gen: ## Regenerate TypeScript API types and the icon set
	cd $(WEB) && npm run gen:api && npm run gen:icons

test: test-web test-api ## Unit + integration tests

test-web:
	cd $(WEB) && npm run check && npm test

test-api: ## Go tests (needs PostgreSQL; database uniquest_test)
	go vet ./... && UQ_TEST_DATABASE_URL=$${UQ_TEST_DATABASE_URL:-postgres://uniquest@127.0.0.1:5432/uniquest_test?sslmode=disable} go test -race -p 1 ./...

e2e: build-demo ## Browser E2E against the demo build (incl. fake-camera QR scanning)
	cd $(WEB) && npx playwright test

e2e-fullstack: build ## Browser E2E against the Go backend (server must not be running)
	$(BIN) seed -admin admin@pollub.test
	UQ_COOKIE_SECURE=false UQ_DEV_LOGIN_CODES=true UQ_OPEN_ALL_STATIONS=true UQ_BASE_URL=http://localhost:8080 $(BIN) serve & \
		PID=$$!; sleep 2; cd $(WEB) && npx playwright test -c playwright.fullstack.config.ts; S=$$?; kill $$PID; exit $$S

docker: ## Build the production image
	docker build -f deploy/Dockerfile -t uniquest:local .
