package auth

import (
	"encoding/base32"
	"testing"
	"time"
)

func TestTOTPRoundTrip(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	if secret == "" {
		t.Fatal("expected a non-empty secret")
	}

	key, err := base32DecodeForTest(secret)
	if err != nil {
		t.Fatalf("secret is not valid base32: %v", err)
	}
	if len(key) != totpSecretLen {
		t.Fatalf("decoded secret length = %d, want %d", len(key), totpSecretLen)
	}

	step := time.Now().Unix() / int64(totpStep.Seconds())
	code := totpCodeAt(key, step)
	if len(code) != totpDigits {
		t.Fatalf("code length = %d, want %d", len(code), totpDigits)
	}

	if !VerifyTOTP(secret, code) {
		t.Error("VerifyTOTP should accept the code for the current time step")
	}
}

func TestVerifyTOTPRejectsWrongCode(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	if VerifyTOTP(secret, "000000") {

		key, _ := base32DecodeForTest(secret)
		step := time.Now().Unix() / int64(totpStep.Seconds())
		if totpCodeAt(key, step) == "000000" {
			t.Skip("code happened to be 000000, skipping")
		}
		t.Error("VerifyTOTP should reject an incorrect code")
	}
}

func TestVerifyTOTPRejectsMalformedInput(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	cases := []string{"", "12345", "1234567", "abcdef", "  123456  "}
	for _, c := range cases {
		if VerifyTOTP(secret, c) {
			t.Errorf("VerifyTOTP(%q) should be rejected", c)
		}
	}
}

func TestTOTPProvisioningURIContainsSecretAndIssuer(t *testing.T) {
	uri := TOTPProvisioningURI("Lyrahcle", "user@example.com", "JBSWY3DPEHPK3PXP")
	if !contains(uri, "otpauth://totp/") {
		t.Errorf("provisioning URI missing scheme/host: %s", uri)
	}
	if !contains(uri, "secret=JBSWY3DPEHPK3PXP") {
		t.Errorf("provisioning URI missing secret: %s", uri)
	}
	if !contains(uri, "issuer=Lyrahcle") {
		t.Errorf("provisioning URI missing issuer: %s", uri)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func base32DecodeForTest(s string) ([]byte, error) {
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
}
