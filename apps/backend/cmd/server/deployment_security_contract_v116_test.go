package main

import (
	"os"
	"strings"
	"testing"
)

func TestFrontendCSPDoesNotRequireInlineScripts(t *testing.T) {
	index, err := os.ReadFile("../../../frontend/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(index)
	if strings.Contains(html, "<script>") {
		t.Fatal("frontend index must not contain inline scripts under strict script-src CSP")
	}
	if !strings.Contains(html, `src="/theme-init.js"`) {
		t.Fatal("frontend theme bootstrap must be an external same-origin script")
	}

	for _, path := range []string{"../../../../deploy/docker/nginx.frontend.conf", "../../../../deploy/docker/nginx.frontend.tls.conf"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if !strings.Contains(src, `script-src 'self'`) {
			t.Fatalf("%s must keep strict same-origin script-src", path)
		}
		if strings.Contains(src, `script-src 'self' 'unsafe-inline'`) {
			t.Fatalf("%s must not allow inline scripts", path)
		}
	}
}

func TestComposeChecksGrafanaPasswordInProduction(t *testing.T) {
	data, err := os.ReadFile("../../../../docker-compose.yml")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"LYRAHCLE_ENV=${ENV:-development}",
		"GRAFANA_ADMIN_PASSWORD",
		`"lyrahcle-dev"`,
		"FATAL: GRAFANA_ADMIN_PASSWORD",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("production Grafana secret contract missing %q", token)
		}
	}
}
