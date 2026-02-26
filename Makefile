COMPOSE=docker compose

.PHONY: up down rebuild logs api-logs ps e2e test

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

rebuild:
	$(COMPOSE) up -d --build

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
