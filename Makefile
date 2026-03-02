COMPOSE=docker compose

.PHONY: up down rebuild run logs api-logs ps e2e test dev dev-api dev-infra

# Development: hot reload without rebuilding Docker
dev-infra:
	$(COMPOSE) up -d postgres redis elasticsearch minio mailhog

dev-api: dev-infra
	@echo "Starting API with hot reload (air). Install: go install github.com/air-verse/air@latest"
	cd api && air

dev: dev-infra
	@echo "Infra up. Run in separate terminals:"
	@echo "  cd api && air          # API with hot reload"
	@echo "  cd frontend && npm run dev   # Frontend with HMR"

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

rebuild:
	$(COMPOSE) up -d --build

run:
	$(COMPOSE) down --remove-orphans --timeout 10
	BUILDX_NO_DEFAULT_ATTESTATIONS=1 $(COMPOSE) build --no-cache api
	BUILDX_NO_DEFAULT_ATTESTATIONS=1 $(COMPOSE) build --no-cache frontend
	$(COMPOSE) up -d --force-recreate

logs:
	$(COMPOSE) logs -f --tail=200

api-logs:
	$(COMPOSE) logs -f --tail=200 api

ps:
	$(COMPOSE) ps

e2e:
	cd api && RUN_E2E=1 E2E_API_BASE=http://localhost:8080 E2E_MAILHOG_BASE=http://localhost:8025 go test -v ./e2e

test:
	cd api && go test ./...
