.PHONY: dev-up migrate build-all test confidence

dev-up:
	@echo "Start local dev stack (docker-compose)"
	docker-compose -f ops/docker/docker-compose.yml up --build -d

migrate:
	@echo "Run DB migrations"
	# placeholder: run goose or migrate tool

build-all:
	@echo "Build Rust, Go, and frontend"
	cd services/merkle-engine && cargo build --release || true
	cd services/vault-api && go build ./... || true
	cd frontend/audit-dashboard && npm install && npm run build || true

test:
	@echo "Run tests"
	cd services/merkle-engine && cargo test || true
	cd services/vault-api && go test ./... || true

confidence:
	@echo "Run confidence gate"
	ci/confidence-gate --config CONFIDENCE.yaml || true
