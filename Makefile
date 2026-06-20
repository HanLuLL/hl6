.PHONY: dev dev-server dev-web db-up db-down

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

dev-server:
	cd server && go run ./cmd/server

dev-web:
	cd web && npm run dev

dev: db-up
	@trap 'kill 0' EXIT; \
	cd server && go run ./cmd/server & \
	cd web && npm run dev & \
	wait
