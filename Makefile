.PHONY: dev-up migrate build-all test confidence

ifeq ($(OS),Windows_NT)
SLEEP_CMD = powershell -Command "Start-Sleep -Seconds 5"
BASH_CMD = bash
ifneq ("$(wildcard C:/Program Files/Git/bin/bash.exe)","")
BASH_CMD = "C:/Program Files/Git/bin/bash.exe"
endif
else
SLEEP_CMD = sleep 5
BASH_CMD = bash
endif

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
	cd services/merkle-engine && cargo test
	cd services/vault-api && go test ./...


.PHONY: compose-up compose-down integration-test

compose-up:
	@echo "Bringing up docker-compose services"
	docker-compose -f ops/docker/docker-compose.yml up --build -d

compose-down:
	@echo "Tearing down docker-compose services"
	docker-compose -f ops/docker/docker-compose.yml down -v --remove-orphans || true

# integration-test: bring up services, wait briefly, run integration tests, then tear down
integration-test: compose-up
	@echo "Waiting for services to become ready..."
	@$(SLEEP_CMD)
	@echo "Running integration and e2e tests"
	@echo "Running Go integration and e2e tests"
	go test ./tests/integration -v || (RET=$$?; $(MAKE) compose-down; exit $$RET)
	go test ./tests/e2e -v || (RET=$$?; $(MAKE) compose-down; exit $$RET)
	$(MAKE) compose-down

confidence:
	@echo "Run confidence gate"
	ci/confidence-gate --config CONFIDENCE.yaml || true

.PHONY: proof-pack proof-pack-run pin-ci

DATE ?= $(shell date -u +%F)
SHA ?= $(shell git rev-parse HEAD)

proof-pack:
	@echo "Preparing proof-pack for $(DATE)"
	$(BASH_CMD) ./scripts/proof_pack.sh "$(DATE)"

proof-pack-run:
	@echo "Running full workflow for proof-pack $(DATE)"
	$(MAKE) compose-up
	$(MAKE) test
	go test ./tests/integration -v
	go test ./tests/e2e -v
	$(BASH_CMD) ./scripts/drill_restore.sh
	$(BASH_CMD) ./scripts/game_day_merkle_down.sh
	$(MAKE) compose-down
	$(MAKE) proof-pack DATE="$(DATE)"
	$(MAKE) pin-ci SHA="$(SHA)" DATE="$(DATE)"

pin-ci:
	@echo "Pinning CI run for SHA=$(SHA) DATE=$(DATE)"
	$(BASH_CMD) ./scripts/pin_ci.sh "$(SHA)" "$(DATE)"
	$(MAKE) proof-pack DATE="$(DATE)"
