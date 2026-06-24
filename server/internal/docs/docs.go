package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

func OpenAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openAPISpec)
	}
}

func RedocHandler() http.HandlerFunc {
	const page = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="theme-color" content="#059669">
  <title>Бухгалтер API</title>
  <link rel="icon" href="/favicon.ico" sizes="any">
  <link rel="icon" href="/icon-192.png" sizes="192x192" type="image/png">
  <link rel="apple-touch-icon" href="/icon-192.png" sizes="192x192">
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url="/docs/openapi.yaml"></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@2.4.0/bundles/redoc.standalone.js"></script>
</body>
</html>`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(page))
	}
}
