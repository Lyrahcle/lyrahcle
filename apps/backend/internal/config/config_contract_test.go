package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

var weakSecretLiterals = map[string]struct{}{
	"password": {},
	"changeme": {},
	"secret":   {},
	"admin":    {},
	"admin123": {},
	"lyrahcle": {},
	"test":     {},
	"default":  {},
}

func productionConfigFiles(t *testing.T) []*ast.File {
	t.Helper()

	entries, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse config package: %v", err)
	}

	pkg, ok := entries["config"]
	if !ok {
		t.Fatalf("expected package config in current directory")
	}

	files := make([]*ast.File, 0, len(pkg.Files))
	for _, file := range pkg.Files {
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatalf("expected at least one production Go file in config package")
	}
	return files
}

func sensitiveConfigIdentifier(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "password") ||
		strings.Contains(n, "secret") ||
		strings.Contains(n, "privatekey") ||
		strings.Contains(n, "private_key") ||
		strings.Contains(n, "accesstoken") ||
		strings.Contains(n, "access_token") ||
		strings.Contains(n, "refreshtoken") ||
		strings.Contains(n, "refresh_token") ||
		strings.Contains(n, "jwtkey") ||
		strings.Contains(n, "jwt_key")
}

func isWeakSecretLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	_, weak := weakSecretLiterals[strings.ToLower(strings.TrimSpace(value))]
	return value, weak
}

func TestConfigProductionCodeDoesNotMutateProcessEnvironment(t *testing.T) {
	for _, file := range productionConfigFiles(t) {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			switch selector.Sel.Name {
			case "Setenv", "Unsetenv", "Clearenv":
				t.Errorf("config production code must not mutate process environment via os.%s", selector.Sel.Name)
			}
			return true
		})
	}
}

func TestConfigDoesNotHardcodeKnownWeakSecrets(t *testing.T) {
	for _, file := range productionConfigFiles(t) {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.ValueSpec:
				for i, name := range n.Names {
					if !sensitiveConfigIdentifier(name.Name) || i >= len(n.Values) {
						continue
					}
					if value, weak := isWeakSecretLiteral(n.Values[i]); weak {
						t.Errorf("sensitive config identifier %s hardcodes weak secret %q", name.Name, value)
					}
				}
			case *ast.KeyValueExpr:
				key, ok := n.Key.(*ast.Ident)
				if !ok || !sensitiveConfigIdentifier(key.Name) {
					return true
				}
				if value, weak := isWeakSecretLiteral(n.Value); weak {
					t.Errorf("sensitive config field %s hardcodes weak secret %q", key.Name, value)
				}
			}
			return true
		})
	}
}
