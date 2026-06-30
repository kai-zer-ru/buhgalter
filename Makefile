VERSION ?= 1.3.0
INSTALL_METHOD ?= manual
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.installMethod=$(INSTALL_METHOD) -X main.buildCommit=$(BUILD_COMMIT) -X main.buildTime=$(BUILD_TIME)

OPENAPI_SRC := docs/api/openapi.yaml
OPENAPI_DST := server/internal/docs/openapi.yaml

.PHONY: dev dev-server dev-web build web-build category-icons-json copy-static copy-openapi openapi-check server-build fix-build-perms test test-unit test-e2e test-coverage lint lint-go prepare prepare-go prepare-web prepare-gen docker-build act-push act-release tag-release migrate ci sqlc sqlc-check download-bank-logos download-marketplace-logos clear version

DOCKER_COMPOSE := docker compose -f docker/docker-compose.yml

# act/docker CI may leave root-owned files in these dirs → EACCES on local `make build`
BUILD_ARTIFACT_DIRS := web/build web/.svelte-kit/output server/internal/static/dist
UID_GID := $(shell id -u):$(shell id -g)

fix-build-perms:
	@for dir in $(BUILD_ARTIFACT_DIRS); do \
		if [ -d "$$dir" ] && find "$$dir" -user root 2>/dev/null | grep -q .; then \
			echo "fix-build-perms: $$dir (root → $(UID_GID))"; \
			docker run --rm -v "$(CURDIR)/$$dir:/p" alpine chown -R $(UID_GID) /p; \
		fi; \
	done

copy-openapi:
	cp $(OPENAPI_SRC) $(OPENAPI_DST)

openapi-check: copy-openapi
	@python3 scripts/check_openapi_error_examples.py $(OPENAPI_SRC)
	@git diff --exit-code -- $(OPENAPI_DST) || \
		(echo "openapi: run 'make copy-openapi' and commit $(OPENAPI_DST)" && exit 1)

build: fix-build-perms category-icons-json web-build copy-static copy-openapi server-build

category-icons-json:
	python3 scripts/build_category_icons_json.py

web-build: category-icons-json
	cd web && npm run build

copy-static:
	mkdir -p server/internal/static/dist
	rm -rf server/internal/static/dist/*
	cp -r web/build/* server/internal/static/dist/

server-build: copy-openapi
	mkdir -p bin
	cd server && go build -tags=embedstatic -ldflags "$(LDFLAGS)" -o ../bin/buhgalter ./cmd/buhgalter
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true

test: test-unit test-e2e

test-unit: copy-openapi
	cd server && go test ./...
	cd web && npm run check

test-coverage:
	cd server && go test ./internal/money/... ./internal/transaction/... ./internal/credit/... ./internal/importexport/... ./internal/auth/... ./internal/notify/... -cover

test-e2e: build
	cd web && npm exec playwright install chromium
	cd web && npm exec playwright test

lint-go:
	cd server && golangci-lint run ./...

ACT_PLATFORM := -P ubuntu-latest=catthehacker/ubuntu:full-latest

act-push:
	@git rev-parse HEAD >/dev/null 2>&1 || (echo "act-push: нужен хотя бы один git commit (без него checkout в act удаляет исходники)" && exit 1)
	act push $(ACT_PLATFORM) -W .github/workflows/ci.yml

# Локальная проверка release.yml (нужен GITHUB_TOKEN; публикация на GitHub — только с реальным токеном).
# Пример: GITHUB_TOKEN=ghp_... make act-release
act-release:
	@git rev-parse HEAD >/dev/null 2>&1 || (echo "act-release: нужен хотя бы один git commit" && exit 1)
	@test -n "$$GITHUB_TOKEN" || (echo "act-release: задайте GITHUB_TOKEN (Personal Access Token с repo)" && exit 1)
	act push $(ACT_PLATFORM) -W .github/workflows/release.yml -e .github/act/tag-push.json -s GITHUB_TOKEN=$$GITHUB_TOKEN

# Сборка текущей версии из VERSION и публикация аннотированного тега vX.Y.Z на origin.
tag-release:
	@chmod +x scripts/tag_release.sh
	@scripts/tag_release.sh

lint: lint-go
	cd web && npm run lint

# Автофикс форматирования, линтера и сгенерированных артефактов перед коммитом / make ci.
prepare: prepare-go prepare-web prepare-gen

prepare-go:
	cd server && golangci-lint fmt ./...
	cd server && golangci-lint run --fix ./...

prepare-web:
	cd web && npm run lint:fix

prepare-gen: sqlc copy-openapi category-icons-json

DOCKER_IMAGE_TAG ?= buhgalter:local

docker-build:
	docker build \
		-f docker/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg INSTALL_METHOD=docker \
		--build-arg BUILD_COMMIT=$(BUILD_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE_TAG) \
		.

ci: lint-go sqlc-check openapi-check category-icons-json
	cd web && npm ci && npm run lint
	$(MAKE) test
	$(MAKE) docker-build

migrate:
	@echo "Migrations apply automatically on server start."

SQLC ?= $(HOME)/go/bin/sqlc

sqlc:
	cd server && $(SQLC) generate

sqlc-check: sqlc
	@git diff --exit-code -- server/internal/db/sqlc server/schema.sql server/queries || \
		(echo "sqlc: run 'make sqlc' and commit generated files" && exit 1)

download-bank-logos:
	python3 scripts/download_bank_logos.py

download-marketplace-logos:
	python3 scripts/download_marketplace_logos.py

generate-category-icons:
	python3 scripts/generate_category_icons.py

dev:
	@echo "Use two terminals: make dev-server  |  make dev-web"

dev-server: copy-openapi
	cd server && BUHGALTER_STATIC_EMBED=false go run ./cmd/buhgalter

dev-web:
	cd web && npm run dev -- --host

# Полная очистка runtime-данных и артефактов сборки (без node_modules и ассетов в data/).
# Перед clear остановите dev-server / buhgalter — иначе процесс держит удалённый файл БД открытым.
clear:
	@echo "clear: остановка Docker и удаление томов..."
	-@$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null
	@echo "clear: база данных, бэкапы и маркер setup..."
	@for dir in data server/data; do \
		[ -d "$$dir" ] || continue; \
		rm -f "$$dir"/buhgalter.db "$$dir"/buhgalter.db-wal "$$dir"/buhgalter.db-shm "$$dir"/.configured; \
		rm -rf "$$dir"/backups; \
	done
	@rm -rf backups
	@find . -maxdepth 3 -name '*.test.db' -delete 2>/dev/null || true
	@echo "clear: логи..."
	@for dir in logs server/logs; do \
		[ -d "$$dir" ] || continue; \
		rm -rf "$$dir"/*; \
	done
	@echo "clear: артефакты сборки..."
	@rm -rf bin buhgalter server/buhgalter web/build web/.svelte-kit
	@rm -rf server/internal/static/dist/*
	@echo "clear: готово. С нуля: make build && ./buhgalter  (dev: make dev-server + make dev-web)"

# Usage: make version vX.Y.Z
ifneq (,$(filter version,$(MAKECMDGOALS)))
  VERSION_GOAL := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  ifneq ($(VERSION_GOAL),)
    $(foreach v,$(VERSION_GOAL),$(eval $(v):;@:))
  endif
endif

version:
	@if [ -z "$(VERSION_GOAL)" ]; then \
		echo "Usage: make version vX.Y.Z"; \
		exit 1; \
	fi
	@chmod +x scripts/set_version.sh
	@scripts/set_version.sh $(VERSION_GOAL)
