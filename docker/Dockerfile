FROM node:22-alpine AS web
RUN apk add --no-cache python3
WORKDIR /app
COPY scripts/build_category_icons_json.py scripts/
RUN mkdir -p data && python3 scripts/build_category_icons_json.py
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS server
WORKDIR /app
RUN apk add --no-cache git
ARG VERSION=1.1.1
ARG INSTALL_METHOD=docker
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY docs/api/openapi.yaml ./internal/docs/openapi.yaml
COPY server/ ./
COPY server/locales ./locales
COPY --from=web /app/web/build ./internal/static/dist/
RUN CGO_ENABLED=0 go build -tags=embedstatic -ldflags "-s -w -X main.version=${VERSION} -X main.installMethod=${INSTALL_METHOD} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME}" -o /buhgalter ./cmd/buhgalter

FROM alpine:3.21
# Сертификаты НУЦ Минцифры для MAX API — vendored в docker/certs/ (обновление: scripts/vendor_russian_ca_certs.sh)
COPY docker/certs/russian_trusted_root_ca_pem.crt docker/certs/russian_trusted_sub_ca_2024_pem.crt /tmp/certs/
RUN apk add --no-cache ca-certificates tzdata wget && \
	cp /tmp/certs/russian_trusted_root_ca_pem.crt /usr/local/share/ca-certificates/russian_trusted_root_ca.crt && \
	cp /tmp/certs/russian_trusted_sub_ca_2024_pem.crt /usr/local/share/ca-certificates/russian_trusted_sub_ca_2024.crt && \
	update-ca-certificates && \
	rm -rf /tmp/certs
WORKDIR /app
RUN adduser -D -u 1000 buhgalter
COPY --from=server /buhgalter /app/buhgalter
COPY --from=server /app/locales /app/locales
ARG VERSION=1.1.1
ARG BUILD_COMMIT=unknown
LABEL org.opencontainers.image.source="https://github.com/kai-zer-ru/buhgalter" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${BUILD_COMMIT}"
RUN mkdir -p /app/data /app/logs /app/backups && chown -R buhgalter:buhgalter /app
USER buhgalter
ENV BUHGALTER_ADDR=:8765 \
    BUHGALTER_DB_PATH=/app/data/buhgalter.db \
    BUHGALTER_DATA_DIR=/app/data \
    BUHGALTER_LOG_DIR=/app/logs \
    BUHGALTER_LOCALES_DIR=/app/locales \
    BUHGALTER_ENV_FILE=/app/.env
EXPOSE 8765
VOLUME ["/app/data", "/app/backups", "/app/locales"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD wget -qO- http://127.0.0.1:8765/api/v1/health || exit 1
ENTRYPOINT ["/app/buhgalter"]
