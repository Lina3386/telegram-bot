.PHONY: install-deps docker-up docker-down local-migration-up local-migration-down local-migration-status

MIGRATIONDIR := $(CURDIR)/migrations

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

local-migration-status:
	goose -dir $(MIGRATIONDIR) postgres "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" status -v

local-migration-up:
	goose -dir $(MIGRATIONDIR) postgres "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up -v

local-migration-down:
	goose -dir $(MIGRATIONDIR) postgres "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down -v