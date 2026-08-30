package agentconfig

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

type Settings struct {
	FrequencyHz            int64 `json:"frequency_hz"`
	CaptureIntervalSeconds int   `json:"capture_interval_seconds"`
}

type Payload struct {
	AgentID  string    `json:"agent_id"`
	Version  int64     `json:"version"`
	IssuedAt time.Time `json:"issued_at"`
	Config   Settings  `json:"config"`
}

type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	Checksum  string `json:"checksum_sha256"`
}

type Signer struct {
	privateKey ed25519.PrivateKey
}

func LoadSigner(path string) (*Signer, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agent config signing key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("agent config signing key is not PKCS#8 PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("agent config signing key: %w", err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("agent config signing key must be Ed25519")
	}
	return &Signer{privateKey: privateKey}, nil
}

func GenerateKeyPair() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	return privatePEM, publicPEM, nil
}

func (s *Signer) Sign(agentID string, version int64, settings Settings, issuedAt time.Time) (Envelope, []byte, []byte, error) {
	payloadBytes, err := json.Marshal(Payload{
		AgentID:  agentID,
		Version:  version,
		IssuedAt: issuedAt.UTC(),
		Config:   settings,
	})
	if err != nil {
		return Envelope{}, nil, nil, err
	}
	signature := ed25519.Sign(s.privateKey, payloadBytes)
	checksum := sha256.Sum256(payloadBytes)
	envelope := Envelope{
		Payload:   base64.RawURLEncoding.EncodeToString(payloadBytes),
		Signature: base64.RawURLEncoding.EncodeToString(signature),
		Checksum:  hex.EncodeToString(checksum[:]),
	}
	return envelope, payloadBytes, signature, nil
}
