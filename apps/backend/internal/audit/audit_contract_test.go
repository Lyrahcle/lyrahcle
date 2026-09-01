package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

var highRiskAuditFieldNames = map[string]struct{}{
	"password":      {},
	"passwordhash":  {},
	"secret":        {},
	"secretkey":     {},
	"privatekey":    {},
	"accesstoken":   {},
	"refreshtoken":  {},
	"authorization": {},
	"cookie":        {},
	"sessioncookie": {},
	"jwt":           {},
	"bearer":        {},
}

func normalizedAuditName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", ".", "")
	return replacer.Replace(value)
}

func productionAuditFiles(t *testing.T) []*ast.File {
	t.Helper()

	entries, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse audit package: %v", err)
	}

	pkg, ok := entries["audit"]
	if !ok {
		t.Fatalf("expected package audit in current directory")
	}

	files := make([]*ast.File, 0, len(pkg.Files))
	for _, file := range pkg.Files {
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatalf("expected at least one production Go file in audit package")
	}
	return files
}

func TestAuditModelsDoNotExposeHighRiskSecretFields(t *testing.T) {
	for _, file := range productionAuditFiles(t) {
		ast.Inspect(file, func(node ast.Node) bool {
			structType, ok := node.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					if _, forbidden := highRiskAuditFieldNames[normalizedAuditName(name.Name)]; forbidden {
						t.Errorf("audit struct field %q exposes high-risk secret material", name.Name)
					}
				}

				if field.Tag == nil {
					continue
				}
				tag, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					continue
				}
				for _, part := range strings.Fields(tag) {
					if !strings.HasPrefix(part, `json:"`) {
						continue
					}
					raw := strings.TrimPrefix(part, `json:"`)
					raw = strings.TrimSuffix(raw, `"`)
					jsonName := strings.Split(raw, ",")[0]
					if jsonName == "" || jsonName == "-" {
						continue
					}
					if _, forbidden := highRiskAuditFieldNames[normalizedAuditName(jsonName)]; forbidden {
						t.Errorf("audit JSON field %q exposes high-risk secret material", jsonName)
					}
				}
			}
			return true
		})
	}
}

func TestAuditActionConstantsAreNeverEmpty(t *testing.T) {
	found := 0

	for _, file := range productionAuditFiles(t) {
		ast.Inspect(file, func(node ast.Node) bool {
			valueSpec, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, name := range valueSpec.Names {
				if !strings.Contains(strings.ToLower(name.Name), "action") {
					continue
				}
				if i >= len(valueSpec.Values) {
					continue
				}
				literal, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				found++
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Errorf("audit action %s has invalid string literal: %v", name.Name, err)
					continue
				}
				if strings.TrimSpace(value) == "" {
					t.Errorf("audit action %s must not be empty", name.Name)
				}
			}
			return true
		})
	}

	if found == 0 {
		t.Log("no string audit action constants found; field-safety contract still enforced")
	}
}
