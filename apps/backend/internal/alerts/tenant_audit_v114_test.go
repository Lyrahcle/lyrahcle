package alerts

import (
	"os"
	"strings"
	"testing"
)

func TestCVEAlertWorkerAuditsMatchedAssetTenant(t *testing.T) {
	data, err := os.ReadFile("worker.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		`"organization_id": a.OrganizationID.String()`,
		"InsertIdempotentInOrganization(ctx, a.OrganizationID",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("CVE alert worker tenant audit contract missing %q", token)
		}
	}
}
