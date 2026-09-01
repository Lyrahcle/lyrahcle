package remoteconfig

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var errVerificationKeyNotFound = errors.New("remote config: verification key not found")

type VerificationKeyProvider interface {
	VerificationKey(string) (ed25519.PublicKey, error)
	LegacyKey() (ed25519.PublicKey, error)
}

type localVerificationKeyProvider struct {
	keys   map[string]ed25519.PublicKey
	legacy ed25519.PublicKey
}

func loadVerificationKeyProvider(legacyPath, keyDir string) (VerificationKeyProvider, error) {
	provider := &localVerificationKeyProvider{keys: make(map[string]ed25519.PublicKey)}
	legacyPath = strings.TrimSpace(legacyPath)
	keyDir = strings.TrimSpace(keyDir)

	if legacyPath != "" {
		publicKey, err := readEd25519PublicKey(legacyPath)
		if err != nil {
			return nil, err
		}
		provider.legacy = append(ed25519.PublicKey(nil), publicKey...)
		kid, err := ed25519KID(publicKey)
		if err != nil {
			return nil, err
		}
		provider.keys[kid] = append(ed25519.PublicKey(nil), publicKey...)
	}

	if keyDir != "" {
		entries, err := os.ReadDir(keyDir)
		if err != nil {
			return nil, fmt.Errorf("remote config: read verification key directory: %w", err)
		}
		loaded := 0
		for _, entry := range entries {
			if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".pem" {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, errors.New("remote config: verification key directory must not contain PEM symlinks")
			}
			if loaded >= 64 {
				return nil, errors.New("remote config: verification key directory contains more than 64 PEM keys")
			}
			publicKey, err := readEd25519PublicKey(filepath.Join(keyDir, entry.Name()))
			if err != nil {
				return nil, err
			}
			kid, err := ed25519KID(publicKey)
			if err != nil {
				return nil, err
			}
			provider.keys[kid] = append(ed25519.PublicKey(nil), publicKey...)
			loaded++
		}
	}

	if len(provider.keys) == 0 {
		return nil, errors.New("remote config: at least one verification key is required")
	}
	return provider, nil
}

func (p *localVerificationKeyProvider) VerificationKey(kid string) (ed25519.PublicKey, error) {
	kid = strings.TrimSpace(kid)
	if kid == "" || len(kid) > 128 {
		return nil, errVerificationKeyNotFound
	}
	publicKey, ok := p.keys[kid]
	if !ok {
		return nil, errVerificationKeyNotFound
	}
	return append(ed25519.PublicKey(nil), publicKey...), nil
}

func (p *localVerificationKeyProvider) LegacyKey() (ed25519.PublicKey, error) {
	if len(p.legacy) != ed25519.PublicKeySize {
		return nil, errVerificationKeyNotFound
	}
	return append(ed25519.PublicKey(nil), p.legacy...), nil
}

func readEd25519PublicKey(path string) (ed25519.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("remote config: read verification key: %w", err)
	}
	block, rest := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("remote config: verification key must contain exactly one public key PEM block")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("remote config: parse verification key: %w", err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("remote config: verification key must be Ed25519")
	}
	return publicKey, nil
}

func ed25519KID(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("remote config: invalid Ed25519 verification key")
	}
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("remote config: marshal verification key: %w", err)
	}
	digest := sha256.Sum256(der)
	return "ed25519-sha256-" + hex.EncodeToString(digest[:]), nil
}
