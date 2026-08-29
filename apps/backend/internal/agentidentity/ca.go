package agentidentity

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrInvalidCSR = errors.New("agentidentity: invalid certificate signing request")

type CA struct {
	cert    *x509.Certificate
	signer  crypto.Signer
	certPEM []byte
}

type IssuedCertificate struct {
	PEM               string
	SerialNumber      string
	FingerprintSHA256 string
	NotBefore         time.Time
	NotAfter          time.Time
}

func Load(certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("agentidentity: read CA certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("agentidentity: read CA key: %w", err)
	}
	return Parse(certPEM, keyPEM)
}

func Parse(certPEM, keyPEM []byte) (*CA, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, errors.New("agentidentity: invalid CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("agentidentity: parse CA certificate: %w", err)
	}
	if !cert.IsCA || cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, errors.New("agentidentity: certificate is not a signing CA")
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("agentidentity: invalid CA key PEM")
	}
	signer, err := parseSigner(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	if !publicKeysEqual(cert.PublicKey, signer.Public()) {
		return nil, errors.New("agentidentity: CA certificate and key do not match")
	}
	return &CA{cert: cert, signer: signer, certPEM: append([]byte(nil), certPEM...)}, nil
}

func parseSigner(der []byte) (crypto.Signer, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if signer, ok := key.(crypto.Signer); ok {
			return signer, nil
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	return nil, errors.New("agentidentity: unsupported CA private key")
}

func publicKeysEqual(a, b any) bool {
	left, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false
	}
	right, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func (c *CA) CertificatePEM() string {
	return string(c.certPEM)
}

func (c *CA) SignCSR(agentID, csrPEM string, ttl time.Duration) (IssuedCertificate, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return IssuedCertificate{}, ErrInvalidCSR
	}
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return IssuedCertificate{}, ErrInvalidCSR
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || csr.CheckSignature() != nil || !acceptablePublicKey(csr.PublicKey) {
		return IssuedCertificate{}, ErrInvalidCSR
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return IssuedCertificate{}, fmt.Errorf("agentidentity: generate serial: %w", err)
	}
	if serial.Sign() == 0 {
		serial = big.NewInt(1)
	}

	now := time.Now().UTC()
	notBefore := now.Add(-2 * time.Minute)
	notAfter := now.Add(ttl)
	if notAfter.After(c.cert.NotAfter) {
		notAfter = c.cert.NotAfter
	}
	if !notAfter.After(now.Add(5 * time.Minute)) {
		return IssuedCertificate{}, errors.New("agentidentity: CA certificate is expired or too close to expiry")
	}
	uri, _ := url.Parse("spiffe://lyrahcle/agent/" + url.PathEscape(agentID))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   agentID,
			Organization: []string{"Lyrahcle SDR Agent"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{uri},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, c.cert, csr.PublicKey, c.signer)
	if err != nil {
		return IssuedCertificate{}, fmt.Errorf("agentidentity: sign certificate: %w", err)
	}
	fingerprint := sha256.Sum256(der)
	return IssuedCertificate{
		PEM:               string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		SerialNumber:      serial.Text(16),
		FingerprintSHA256: hex.EncodeToString(fingerprint[:]),
		NotBefore:         notBefore,
		NotAfter:          notAfter,
	}, nil
}

func acceptablePublicKey(key any) bool {
	switch k := key.(type) {
	case *ecdsa.PublicKey:
		return k.Curve.Params().BitSize >= 256
	case ed25519.PublicKey:
		return len(k) == ed25519.PublicKeySize
	case *rsa.PublicKey:
		return k.N.BitLen() >= 2048
	default:
		return false
	}
}

func Fingerprint(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}
