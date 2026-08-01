APP_NAME=chamie-backend

.PHONY: run migrate-up migrate-down

run: migrate-up
	go run ./cmd/chamie

migrate-up:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go run ./cmd/migrateup

migrate-down:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go run ./cmd/migrateup -down -steps $${STEPS:-1}
