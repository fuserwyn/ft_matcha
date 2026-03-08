COMPOSE=docker compose

.PHONY: up down rebuild run rm logs api-logs ps e2e test dev dev-api dev-infra lan-ip showdb sqli-test sqltest

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

# Full Docker cleanup: stop/remove containers, delete volumes and prune system data
rm:
	$(COMPOSE) down -v --remove-orphans --timeout 10
	docker system prune -f
	docker volume prune -f

rebuild:
	$(COMPOSE) up -d --build

run:
	$(COMPOSE) down --remove-orphans --timeout 10
	DOCKER_BUILDKIT=0 $(COMPOSE) build --no-cache api
	DOCKER_BUILDKIT=0 $(COMPOSE) build --no-cache frontend
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

# Run SQL injection protection tests against the running API
sqltest:
	@chmod +x scripts/test_sqli.sh
	@./scripts/test_sqli.sh

# Show all DB tables with data in readable format
showdb:
	@chmod +x scripts/showdb.sh
	@./scripts/showdb.sh

# Detect local IP and update .env for mobile/LAN access
lan-ip:
	@chmod +x scripts/set-lan-ip.sh 2>/dev/null || true
	@./scripts/set-lan-ip.sh
