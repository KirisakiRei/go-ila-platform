# ILA Platform — perintah pengembangan & operasional

BACKEND_DIR := backend
FRONTEND_DIR := frontend
COMPOSE := docker compose -f deploy/docker-compose.yml

.PHONY: help
help:
	@echo "Target yang tersedia:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# --- Backend ----------------------------------------------------------------

.PHONY: build-backend
build-backend: ## Kompilasi semua binary backend
	cd $(BACKEND_DIR) && go build -o ../bin/backend-api ./cmd/backend-api
	cd $(BACKEND_DIR) && go build -o ../bin/runner ./cmd/runner
	cd $(BACKEND_DIR) && go build -o ../bin/migrate ./cmd/migrate

.PHONY: run-api
run-api: ## Jalankan backend-api lokal
	cd $(BACKEND_DIR) && go run ./cmd/backend-api

.PHONY: run-runner
run-runner: ## Jalankan runner lokal
	cd $(BACKEND_DIR) && go run ./cmd/runner

.PHONY: migrate
migrate: ## Jalankan migrasi tabel
	cd $(BACKEND_DIR) && go run ./cmd/migrate

.PHONY: test-backend
test-backend: ## Uji backend dengan race detector
	cd $(BACKEND_DIR) && go test -race ./...

.PHONY: vet
vet: ## Analisis statis backend
	cd $(BACKEND_DIR) && go vet ./...

.PHONY: tidy
tidy: ## Rapikan dependensi backend
	cd $(BACKEND_DIR) && go mod tidy

# --- Frontend ---------------------------------------------------------------

.PHONY: install-frontend
install-frontend: ## Pasang dependensi frontend
	cd $(FRONTEND_DIR) && npm install

.PHONY: dev-frontend
dev-frontend: ## Jalankan dev server frontend
	cd $(FRONTEND_DIR) && npm run dev

.PHONY: build-frontend
build-frontend: ## Build frontend ke dist/
	cd $(FRONTEND_DIR) && npm run build

.PHONY: check-frontend
check-frontend: ## Type-check frontend tanpa emit
	cd $(FRONTEND_DIR) && npm run lint

# --- Docker -----------------------------------------------------------------

.PHONY: up
up: ## Nyalakan semua container
	$(COMPOSE) up -d --build

.PHONY: down
down: ## Hentikan semua container
	$(COMPOSE) down

.PHONY: logs
logs: ## Lihat log container (follow)
	$(COMPOSE) logs -f --tail=100

.PHONY: ps
ps: ## Status container
	$(COMPOSE) ps

# --- Gabungan ---------------------------------------------------------------

.PHONY: check
check: vet test-backend check-frontend ## Semua pemeriksaan sebelum commit
