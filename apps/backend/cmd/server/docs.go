package main

import (
	_ "embed"
	"net/http"
)

var openAPISpec []byte

const swaggerUIPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Lyrahcle API docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>body { margin: 0; background: #0a0a0c; }</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis],
        })
      }
    </script>
  </body>
</html>`

func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openAPISpec)
}

func serveAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; script-src 'unsafe-inline' https://unpkg.com; "+
			"style-src 'unsafe-inline' https://unpkg.com; connect-src 'self'; "+
			"img-src 'self' data: https://unpkg.com; frame-ancestors 'none'",
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIPage))
}
