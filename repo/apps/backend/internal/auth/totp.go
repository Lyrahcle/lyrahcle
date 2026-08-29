package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	totpDigits    = 6
	totpStep      = 30 * time.Second
	totpSkewSteps = 1 // accept the previous and next 30s window to absorb clock drift
	totpSecretLen = 20
)

// GenerateTOTPSecret returns a new random base32-encoded (RFC 4648, no
// padding) TOTP secret suitable for a Google-Authenticator-compatible app.
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, totpSecretLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// TOTPProvisioningURI builds an otpauth:// URI that authenticator apps can
// scan as a QR code to enroll the account.
func TOTPProvisioningURI(issuer, accountName, secret string) string {
	u := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   "/" + issuer + ":" + accountName,
	}
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", totpDigits))
	q.Set("period", fmt.Sprintf("%d", int(totpStep.Seconds())))
	u.RawQuery = q.Encode()
	return u.String()
}

// VerifyTOTP checks a 6-digit code against secret, allowing a small window
// around the current time step to tolerate clock drift between the server
// and the authenticator app.
func VerifyTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	now := time.Now().Unix() / int64(totpStep.Seconds())
	for skew := -totpSkewSteps; skew <= totpSkewSteps; skew++ {
		if subtle.ConstantTimeCompare([]byte(code), []byte(totpCodeAt(key, now+int64(skew)))) == 1 {
			return true
		}
	}
	return false
}

// CurrentTOTPCode returns the 6-digit code for secret at the current time
// step. Exposed primarily for tests and operational tooling — VerifyTOTP is
// what should be used to check a code supplied by a user.
func CurrentTOTPCode(secret string) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	step := time.Now().Unix() / int64(totpStep.Seconds())
	return totpCodeAt(key, step), nil
}

func totpCodeAt(key []byte, step int64) string {
	msg := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		msg[i] = byte(step & 0xff)
		step >>= 8
	}

	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])

	mod := uint32(math.Pow10(totpDigits))
	return fmt.Sprintf("%0*d", totpDigits, binCode%mod)
}
