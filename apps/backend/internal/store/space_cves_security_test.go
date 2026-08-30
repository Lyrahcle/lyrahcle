package store

import (
	"strings"
	"testing"
)

func TestSpaceCVESortClauseUsesWhitelist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sortBy   string
		desc     bool
		wantExpr string
		wantDir  string
	}{
		{name: "published", sortBy: "published", wantExpr: "published", wantDir: "ASC"},
		{name: "score desc", sortBy: "score", desc: true, wantExpr: "score", wantDir: "DESC"},
		{name: "unknown fallback", sortBy: "does_not_exist", wantExpr: "published", wantDir: "ASC"},
		{name: "sql injection fallback", sortBy: "published DESC; DROP TABLE users; --", wantExpr: "published", wantDir: "ASC"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, dir := spaceCVESortClause(tc.sortBy, tc.desc)
			if expr != tc.wantExpr || dir != tc.wantDir {
				t.Fatalf("spaceCVESortClause(%q, %v) = (%q, %q), want (%q, %q)", tc.sortBy, tc.desc, expr, dir, tc.wantExpr, tc.wantDir)
			}
			if strings.Contains(expr, tc.sortBy) && strings.ContainsAny(tc.sortBy, ";-'") {
				t.Fatalf("untrusted sort input leaked into SQL expression: %q", expr)
			}
		})
	}
}

func TestSpaceCVESeveritySortIsStatic(t *testing.T) {
	t.Parallel()
	expr, dir := spaceCVESortClause("severity", false)
	if dir != "ASC" {
		t.Fatalf("unexpected direction %q", dir)
	}
	for _, fragment := range []string{"CASE lower(severity)", "WHEN 'critical' THEN 4", "ELSE 0"} {
		if !strings.Contains(expr, fragment) {
			t.Fatalf("severity expression missing %q: %q", fragment, expr)
		}
	}
}
