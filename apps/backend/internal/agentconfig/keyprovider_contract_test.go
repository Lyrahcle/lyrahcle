package agentconfig

import (
	"os"
	"strings"
	"testing"
)

func TestSignerDoesNotReadOrUsePrivateKeyMaterialDirectly(t *testing.T) {
	data, err := os.ReadFile("signing.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{
		"os.ReadFile(",
		"x509.ParsePKCS8PrivateKey(",
		"ed25519.Sign(",
		"privateKey ed25519.PrivateKey",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("agent config signer bypasses KeyProvider with %q", forbidden)
		}
	}
	for _, required := range []string{
		"provider keyprovider.Provider",
		"s.provider.ActiveSigningKey(ctx)",
		"key.Sign(ctx, payloadBytes)",
		"keyprovider.ValidateKID(key.KID())",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("agent config signer KeyProvider contract missing %q", required)
		}
	}
}
