VERSION ?= 1.4.0
INSTALL_METHOD ?= manual
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.installMethod=$(INSTALL_METHOD) -X main.buildCommit=$(BUILD_COMMIT) -X main.buildTime=$(BUILD_TIME)

OPENAPI_SRC := docs/api/openapi.yaml
OPENAPI_DST := server/internal/docs/openapi.yaml
UI_I18N_SRC := android/ui/src/lib/i18n
UI_I18N_DST := server/ui_locales

.PHONY: dev dev-server dev-web build build-arm web-build category-icons-json copy-static copy-openapi check-openapi-examples openapi-check copy-ui-i18n ui-i18n-check inline-sql-check server-build server-build-arm fix-build-perms test test-unit test-e2e test-e2e-web test-coverage lint lint-go prepare prepare-go prepare-web prepare-android prepare-gen prepare-sql-check docker-build act-push act-release tag-release migrate ci sqlc sqlc-check download-bank-logos download-marketplace-logos clear version android-icons android-ui-build android-sync android-apk android-apk-release android-install android-install-release android-ensure-sdk

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

check-openapi-examples:
	@python3 scripts/check_openapi_error_examples.py $(OPENAPI_SRC)

# CI: sync + validate examples + require committed artifacts.
openapi-check: copy-openapi check-openapi-examples
	@git diff --exit-code -- $(OPENAPI_SRC) $(OPENAPI_DST) || \
		(echo "openapi: run 'make copy-openapi' and commit $(OPENAPI_SRC) and $(OPENAPI_DST)" && exit 1)

copy-ui-i18n:
	cp $(UI_I18N_SRC)/ru.json $(UI_I18N_DST)/ru.json
	cp $(UI_I18N_SRC)/en.json $(UI_I18N_DST)/en.json

# CI: sync + require committed server/ui_locales.
ui-i18n-check: copy-ui-i18n
	@git diff --exit-code -- $(UI_I18N_DST)/ru.json $(UI_I18N_DST)/en.json || \
		(echo "ui-i18n: run 'make copy-ui-i18n' and commit $(UI_I18N_DST)/{ru,en}.json" && exit 1)

build: fix-build-perms category-icons-json web-build copy-static copy-openapi copy-ui-i18n server-build

# linux/arm64 (aarch64) — Raspberry Pi 3/4/5 и другие ARM64-хосты.
# Бинарник: bin/buhgalter-linux-arm64 (не перезаписывает нативный bin/buhgalter).
build-arm: fix-build-perms category-icons-json web-build copy-static copy-openapi copy-ui-i18n server-build-arm

category-icons-json:
	python3 scripts/build_category_icons_json.py

web-build: category-icons-json
	cd web && npm run build

copy-static:
	mkdir -p server/internal/static/dist
	rm -rf server/internal/static/dist/*
	cp -r web/build/* server/internal/static/dist/

server-build: copy-openapi copy-ui-i18n
	mkdir -p bin
	cd server && go build -tags=embedstatic -ldflags "$(LDFLAGS)" -o ../bin/buhgalter ./cmd/buhgalter
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true
	@cp -f bin/buhgalter buhgalter 2>/dev/null || true

server-build-arm: copy-openapi copy-ui-i18n
	mkdir -p bin
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=embedstatic -ldflags "$(LDFLAGS)" -o ../bin/buhgalter-linux-arm64 ./cmd/buhgalter
	@echo "ARM64 binary: bin/buhgalter-linux-arm64"

test: test-unit test-e2e

test-unit: copy-openapi copy-ui-i18n
	cd server && go test ./...
	cd web && npm run check
	cd android/ui && npm run check
	cd android/ui && npm run test:unit

test-coverage:
	cd server && go test ./internal/money/... ./internal/transaction/... ./internal/credit/... ./internal/importexport/... ./internal/auth/... ./internal/notify/... -cover

test-e2e: build test-e2e-web

test-e2e-web:
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
	cd android/ui && npm run lint

# Автофикс форматирования, линтера и сгенерированных артефактов перед коммитом / make ci.
# prepare-gen синхронизирует артефакты (copy-*), check-цели с git diff — только в make ci.
prepare: prepare-go prepare-web prepare-android prepare-gen prepare-sql-check

prepare-go:
	cd server && golangci-lint fmt ./...
	cd server && golangci-lint run --fix ./...

prepare-web:
	cd web && npm run lint:fix

prepare-android:
	cd android/ui && npm run lint:fix

prepare-gen: sqlc category-icons-json copy-openapi check-openapi-examples copy-ui-i18n

prepare-sql-check: sqlc-check inline-sql-check

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

ci: lint-go sqlc-check inline-sql-check openapi-check ui-i18n-check category-icons-json
	cd web && npm ci && npm run lint
	cd android/ui && npm ci && npm run lint
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

inline-sql-check:
	python3 scripts/check_inline_sql.py

download-bank-logos:
	python3 scripts/download_bank_logos.py

download-marketplace-logos:
	python3 scripts/download_marketplace_logos.py

generate-category-icons:
	python3 scripts/generate_category_icons.py

dev:
	@echo "Use two terminals: make dev-server  |  make dev-web"

dev-server: copy-openapi copy-ui-i18n
	cd server && BUHGALTER_STATIC_EMBED=false BUHGALTER_ENV_FILE=$(CURDIR)/.env go run ./cmd/buhgalter

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

android-icons:
	python3 scripts/android-icons.py

ANDROID_APK := android/app/build/outputs/apk/debug/app-debug.apk
ANDROID_APK_DEBUG_DIR := android/app/build/outputs/apk/debug
ANDROID_APK_RELEASE_DIR := android/app/build/outputs/apk/release
ANDROID_APK_RELEASE := $(ANDROID_APK_RELEASE_DIR)/app-release.apk
ANDROID_APK_RELEASE_UNIVERSAL := $(ANDROID_APK_RELEASE_DIR)/app-universal-release.apk
ANDROID_APK_RELEASE_ARM64 := $(ANDROID_APK_RELEASE_DIR)/app-arm64-v8a-release.apk
ANDROID_APK_RELEASE_ARMV7 := $(ANDROID_APK_RELEASE_DIR)/app-armeabi-v7a-release.apk
ANDROID_APK_RELEASE_X86_64 := $(ANDROID_APK_RELEASE_DIR)/app-x86_64-release.apk

android-ui-build: category-icons-json
	cd android/ui && npm install && npm run build

android-sync: android-ui-build android-icons
	cd android && npm install
	cd android && npx cap sync android

android-apk: android-sync
	@$(MAKE) --no-print-directory android-ensure-sdk
	set -e; \
	if [ -f scripts/android-env.sh ]; then . scripts/android-env.sh; fi; \
	if [ -z "$$JAVA_HOME" ] || [ ! -x "$$JAVA_HOME/bin/java" ]; then \
		echo "ERROR: JAVA_HOME is unset or invalid: '$$JAVA_HOME'"; \
		echo "  CI: actions/setup-java; local: source scripts/android-env.sh"; \
		exit 1; \
	fi; \
	export PATH="$$JAVA_HOME/bin:$$PATH"; \
	echo "Gradle with JAVA_HOME=$$JAVA_HOME"; \
	cd android && ./gradlew assembleDebug --stacktrace --no-daemon
	@echo "APK: $(ANDROID_APK)"

android-apk-release: android-sync
	@$(MAKE) --no-print-directory android-ensure-sdk
	set -e; \
	if [ -f scripts/android-env.sh ]; then . scripts/android-env.sh; fi; \
	if [ -z "$$JAVA_HOME" ] || [ ! -x "$$JAVA_HOME/bin/java" ]; then \
		echo "ERROR: JAVA_HOME is unset or invalid: '$$JAVA_HOME'"; \
		echo "  CI: actions/setup-java; local: source scripts/android-env.sh"; \
		exit 1; \
	fi; \
	export PATH="$$JAVA_HOME/bin:$$PATH"; \
	echo "Gradle with JAVA_HOME=$$JAVA_HOME"; \
	cd android && ./gradlew assembleRelease --stacktrace --no-daemon
	@test -f "$(ANDROID_APK_RELEASE_UNIVERSAL)" || (echo "ERROR: missing $(ANDROID_APK_RELEASE_UNIVERSAL)"; ls -la "$(ANDROID_APK_RELEASE_DIR)" || true; exit 1)
	cp -f "$(ANDROID_APK_RELEASE_UNIVERSAL)" "$(ANDROID_APK_RELEASE)"
	@echo "Release APKs:"
	@ls -lh "$(ANDROID_APK_RELEASE)" \
		"$(ANDROID_APK_RELEASE_ARM64)" \
		"$(ANDROID_APK_RELEASE_ARMV7)" \
		"$(ANDROID_APK_RELEASE_X86_64)"
	@bash scripts/verify-android-release-apks.sh "$(ANDROID_APK_RELEASE_DIR)"

# Ensure android/local.properties exists (CI checkout has none; sdk.dir from ANDROID_*).
android-ensure-sdk:
	@if [ ! -f android/local.properties ]; then \
		SDK="$${ANDROID_SDK_ROOT:-$${ANDROID_HOME:-}}"; \
		if [ -z "$$SDK" ] || [ ! -d "$$SDK" ]; then \
			echo "ERROR: android/local.properties missing and ANDROID_SDK_ROOT/ANDROID_HOME unset."; \
			echo "  Run scripts/setup-android-dev.sh or set ANDROID_SDK_ROOT."; \
			exit 1; \
		fi; \
		printf 'sdk.dir=%s\n' "$$SDK" > android/local.properties; \
		echo "Wrote android/local.properties (sdk.dir=$$SDK)"; \
	fi
	@echo "android SDK: $$(grep '^sdk.dir=' android/local.properties)"
	@echo "JAVA_HOME=$${JAVA_HOME:-<unset>}"
	@if [ -n "$$JAVA_HOME" ] && [ -x "$$JAVA_HOME/bin/java" ]; then \
		"$$JAVA_HOME/bin/java" -version; \
	elif command -v java >/dev/null 2>&1; then \
		java -version; \
	else \
		echo "ERROR: java not found. Set JAVA_HOME (CI: setup-java; local: source scripts/android-env.sh)."; \
		exit 1; \
	fi
android-install: android-apk
	@apk="$(ANDROID_APK)"; \
	if [ ! -f "$$apk" ]; then \
		if [ -f "$(ANDROID_APK_DEBUG_DIR)/app-universal-debug.apk" ]; then \
			apk="$(ANDROID_APK_DEBUG_DIR)/app-universal-debug.apk"; \
		else \
			abi=$$(adb shell getprop ro.product.cpu.abi 2>/dev/null | tr -d '\r'); \
			case "$$abi" in \
				arm64-v8a|armeabi-v7a|x86_64) \
					candidate="$(ANDROID_APK_DEBUG_DIR)/app-$$abi-debug.apk"; \
					if [ -f "$$candidate" ]; then apk="$$candidate"; fi ;; \
			esac; \
		fi; \
	fi; \
	if [ ! -f "$$apk" ]; then \
		echo "APK not found (expected $(ANDROID_APK))"; \
		ls -1 "$(ANDROID_APK_DEBUG_DIR)"/*.apk 2>/dev/null || true; \
		exit 1; \
	fi; \
	echo "Installing $$apk"; \
	adb install -r "$$apk"

android-install-release: android-apk-release
	@apk="$(ANDROID_APK_RELEASE_UNIVERSAL)"; \
	if [ ! -f "$$apk" ]; then \
		echo "APK not found (expected $$apk)"; \
		ls -1 "$(ANDROID_APK_RELEASE_DIR)"/*.apk 2>/dev/null || true; \
		exit 1; \
	fi; \
	echo "Installing $$apk"; \
	adb install -r "$$apk"
