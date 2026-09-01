package keyprovider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrKeyNotFound          = errors.New("keyprovider: key not found")
	ErrKeyRevoked           = errors.New("keyprovider: key revoked")
	ErrKeyNotStaged         = errors.New("keyprovider: key is not staged")
	ErrRotationPending      = errors.New("keyprovider: a staged rotation already exists")
	ErrActiveKey            = errors.New("keyprovider: active key cannot be revoked; activate a replacement first")
	ErrLifecycleUnsupported = errors.New("keyprovider: lifecycle management is not supported by this provider")
)

type KeyStatus string

const (
	KeyStatusStaged  KeyStatus = "staged"
	KeyStatusActive  KeyStatus = "active"
	KeyStatusGrace   KeyStatus = "grace"
	KeyStatusRevoked KeyStatus = "revoked"
)

type SigningKey interface {
	KID() string
	Algorithm() string
	Sign(context.Context, []byte) ([]byte, error)
}

type Provider interface {
	ActiveSigningKey(context.Context) (SigningKey, error)
	SigningKey(context.Context, string) (SigningKey, error)
}

type KeyMetadata struct {
	KID            string     `json:"kid"`
	Algorithm      string     `json:"algorithm"`
	Status         KeyStatus  `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
	GraceExpiresAt *time.Time `json:"grace_expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type LifecycleProvider interface {
	Provider
	Keys(context.Context) ([]KeyMetadata, error)
	PrepareRotation(context.Context) (KeyMetadata, []byte, error)
	Activate(context.Context, string, time.Duration) (KeyMetadata, error)
	Revoke(context.Context, string) (KeyMetadata, error)
	PublicKeyPEM(context.Context, string) ([]byte, error)
	RotationPolicy(context.Context) (RotationPolicy, error)
	UpdateRotationPolicy(context.Context, RotationPolicy) (RotationPolicy, error)
}

type LocalProvider struct {
	activeKID string
	keys      map[string]SigningKey
}

type localEd25519Key struct {
	kid        string
	privateKey ed25519.PrivateKey
}

func NewLocalProvider(activePath string, additionalPaths ...string) (*LocalProvider, error) {
	activePath = strings.TrimSpace(activePath)
	if activePath == "" {
		return nil, errors.New("keyprovider: active key path is required")
	}
	provider := &LocalProvider{keys: make(map[string]SigningKey)}
	paths := append([]string{activePath}, additionalPaths...)
	for i, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		key, err := loadLocalEd25519Key(path)
		if err != nil {
			return nil, err
		}
		if existing, ok := provider.keys[key.kid]; ok && existing.Algorithm() != key.Algorithm() {
			return nil, fmt.Errorf("keyprovider: duplicate kid %q with different algorithms", key.kid)
		}
		provider.keys[key.kid] = key
		if i == 0 {
			provider.activeKID = key.kid
		}
	}
	if provider.activeKID == "" {
		return nil, errors.New("keyprovider: active key could not be loaded")
	}
	return provider, nil
}

func (p *LocalProvider) ActiveSigningKey(context.Context) (SigningKey, error) {
	if p == nil {
		return nil, ErrKeyNotFound
	}
	key, ok := p.keys[p.activeKID]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (p *LocalProvider) SigningKey(_ context.Context, kid string) (SigningKey, error) {
	if p == nil {
		return nil, ErrKeyNotFound
	}
	key, ok := p.keys[strings.TrimSpace(kid)]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (k *localEd25519Key) KID() string       { return k.kid }
func (k *localEd25519Key) Algorithm() string { return "Ed25519" }
func (k *localEd25519Key) Sign(_ context.Context, payload []byte) ([]byte, error) {
	if len(k.privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("keyprovider: invalid Ed25519 private key")
	}
	return ed25519.Sign(k.privateKey, payload), nil
}

type managedManifest struct {
	Version        int                         `json:"version"`
	Keys           map[string]managedKeyRecord `json:"keys"`
	RotationPolicy RotationPolicy              `json:"rotation_policy"`
}

type managedKeyRecord struct {
	FileName       string     `json:"file_name"`
	Status         KeyStatus  `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
	GraceExpiresAt *time.Time `json:"grace_expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type ManagedLocalProvider struct {
	mu           sync.RWMutex
	dir          string
	manifestPath string
	manifest     managedManifest
	keys         map[string]*localEd25519Key
}

func NewManagedLocalProvider(dir, bootstrapPath string) (*ManagedLocalProvider, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("keyprovider: managed key directory is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("keyprovider: create managed key directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: inspect managed key directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("keyprovider: managed key directory must be a real directory")
	}
	p := &ManagedLocalProvider{
		dir: dir, manifestPath: filepath.Join(dir, "manifest.json"), keys: map[string]*localEd25519Key{},
		manifest: managedManifest{Version: 1, Keys: map[string]managedKeyRecord{}, RotationPolicy: DefaultRotationPolicy()},
	}
	if _, err := os.Stat(p.manifestPath); err == nil {
		if err := p.loadManifest(); err != nil {
			return nil, err
		}
		return p, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("keyprovider: inspect manifest: %w", err)
	}

	if strings.TrimSpace(bootstrapPath) != "" {
		key, err := loadLocalEd25519Key(bootstrapPath)
		if err != nil {
			return nil, fmt.Errorf("keyprovider: import bootstrap key: %w", err)
		}
		fileName, err := p.persistPrivateKey(key)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		p.manifest.Keys[key.kid] = managedKeyRecord{FileName: fileName, Status: KeyStatusActive, CreatedAt: now, ActivatedAt: &now}
		p.keys[key.kid] = key
	} else {
		key, err := generateLocalEd25519Key()
		if err != nil {
			return nil, err
		}
		fileName, err := p.persistPrivateKey(key)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		p.manifest.Keys[key.kid] = managedKeyRecord{FileName: fileName, Status: KeyStatusActive, CreatedAt: now, ActivatedAt: &now}
		p.keys[key.kid] = key
	}
	if err := p.saveManifestLocked(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *ManagedLocalProvider) fileNameForKID(kid string) string {
	if rec, ok := p.manifest.Keys[kid]; ok && rec.FileName != "" {
		return rec.FileName
	}
	return randomKeyFileName()
}

func (p *ManagedLocalProvider) loadManifest() error {
	raw, err := os.ReadFile(p.manifestPath)
	if err != nil {
		return fmt.Errorf("keyprovider: read manifest: %w", err)
	}
	var manifest managedManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("keyprovider: parse manifest: %w", err)
	}
	if manifest.Version != 1 || len(manifest.Keys) == 0 {
		return errors.New("keyprovider: invalid managed key manifest")
	}
	policy, policyChanged := NormalizeRotationPolicy(manifest.RotationPolicy)
	manifest.RotationPolicy = policy
	p.manifest = manifest
	p.keys = map[string]*localEd25519Key{}
	activeCount := 0
	for kid, rec := range manifest.Keys {
		if err := ValidateKID(kid); err != nil {
			return fmt.Errorf("keyprovider: manifest kid %q: %w", kid, err)
		}
		if rec.FileName == "" || filepath.Base(rec.FileName) != rec.FileName {
			return errors.New("keyprovider: invalid managed key filename")
		}
		path := filepath.Join(p.dir, rec.FileName)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("keyprovider: inspect managed private key: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("keyprovider: managed private key must be a regular non-symlink file")
		}
		key, err := loadLocalEd25519Key(path)
		if err != nil {
			return err
		}
		if key.kid != kid {
			return errors.New("keyprovider: managed key fingerprint does not match manifest kid")
		}
		p.keys[kid] = key
		if rec.Status == KeyStatusActive {
			activeCount++
		}
	}
	if activeCount > 1 {
		return errors.New("keyprovider: managed manifest contains multiple active keys")
	}
	p.mu.Lock()
	changed := policyChanged || p.refreshExpiredLocked(time.Now().UTC())
	if changed {
		err = p.saveManifestLocked()
	}
	p.mu.Unlock()
	return err
}

func (p *ManagedLocalProvider) persistPrivateKey(key *localEd25519Key) (string, error) {
	fileName := p.fileNameForKID(key.kid)
	path := filepath.Join(p.dir, fileName)
	der, err := x509.MarshalPKCS8PrivateKey(key.privateKey)
	if err != nil {
		return "", fmt.Errorf("keyprovider: marshal private key: %w", err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("keyprovider: write managed private key: %w", err)
	}
	return fileName, nil
}

func (p *ManagedLocalProvider) saveManifestLocked() error {
	raw, err := json.MarshalIndent(p.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("keyprovider: marshal manifest: %w", err)
	}
	temp := p.manifestPath + ".tmp"
	if err := os.WriteFile(temp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("keyprovider: write manifest: %w", err)
	}
	if err := os.Rename(temp, p.manifestPath); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("keyprovider: replace manifest: %w", err)
	}
	return nil
}

func (p *ManagedLocalProvider) refreshExpiredLocked(now time.Time) bool {
	changed := false
	for kid, rec := range p.manifest.Keys {
		if rec.Status == KeyStatusGrace && rec.GraceExpiresAt != nil && !now.Before(*rec.GraceExpiresAt) {
			revoked := *rec.GraceExpiresAt
			rec.Status = KeyStatusRevoked
			rec.RevokedAt = &revoked
			p.manifest.Keys[kid] = rec
			changed = true
		}
	}
	return changed
}

func (p *ManagedLocalProvider) ActiveSigningKey(_ context.Context) (SigningKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refreshExpiredLocked(time.Now().UTC()) {
		if err := p.saveManifestLocked(); err != nil {
			return nil, err
		}
	}
	for kid, rec := range p.manifest.Keys {
		if rec.Status == KeyStatusActive {
			key := p.keys[kid]
			if key == nil {
				return nil, ErrKeyNotFound
			}
			return key, nil
		}
	}
	return nil, ErrKeyNotFound
}

func (p *ManagedLocalProvider) SigningKey(_ context.Context, kid string) (SigningKey, error) {
	kid = strings.TrimSpace(kid)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refreshExpiredLocked(time.Now().UTC()) {
		if err := p.saveManifestLocked(); err != nil {
			return nil, err
		}
	}
	rec, ok := p.manifest.Keys[kid]
	if !ok {
		return nil, ErrKeyNotFound
	}
	if rec.Status == KeyStatusRevoked {
		return nil, ErrKeyRevoked
	}
	if rec.Status != KeyStatusActive && rec.Status != KeyStatusGrace {
		return nil, ErrKeyNotFound
	}
	key := p.keys[kid]
	if key == nil {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (p *ManagedLocalProvider) Keys(_ context.Context) ([]KeyMetadata, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refreshExpiredLocked(time.Now().UTC()) {
		if err := p.saveManifestLocked(); err != nil {
			return nil, err
		}
	}
	out := make([]KeyMetadata, 0, len(p.manifest.Keys))
	for kid, rec := range p.manifest.Keys {
		out = append(out, KeyMetadata{KID: kid, Algorithm: "Ed25519", Status: rec.Status, CreatedAt: rec.CreatedAt, ActivatedAt: rec.ActivatedAt, GraceExpiresAt: rec.GraceExpiresAt, RevokedAt: rec.RevokedAt})
	}
	order := map[KeyStatus]int{KeyStatusActive: 0, KeyStatusStaged: 1, KeyStatusGrace: 2, KeyStatusRevoked: 3}
	sort.Slice(out, func(i, j int) bool {
		if order[out[i].Status] != order[out[j].Status] {
			return order[out[i].Status] < order[out[j].Status]
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (p *ManagedLocalProvider) PrepareRotation(_ context.Context) (KeyMetadata, []byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refreshExpiredLocked(time.Now().UTC()) {
		if err := p.saveManifestLocked(); err != nil {
			return KeyMetadata{}, nil, err
		}
	}
	for _, rec := range p.manifest.Keys {
		if rec.Status == KeyStatusStaged {
			return KeyMetadata{}, nil, ErrRotationPending
		}
	}
	key, err := generateLocalEd25519Key()
	if err != nil {
		return KeyMetadata{}, nil, err
	}
	fileName := randomKeyFileName()
	der, err := x509.MarshalPKCS8PrivateKey(key.privateKey)
	if err != nil {
		return KeyMetadata{}, nil, err
	}
	if err := os.WriteFile(filepath.Join(p.dir, fileName), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		return KeyMetadata{}, nil, fmt.Errorf("keyprovider: write staged private key: %w", err)
	}
	now := time.Now().UTC()
	p.manifest.Keys[key.kid] = managedKeyRecord{FileName: fileName, Status: KeyStatusStaged, CreatedAt: now}
	p.keys[key.kid] = key
	if err := p.saveManifestLocked(); err != nil {
		return KeyMetadata{}, nil, err
	}
	publicPEM, err := publicKeyPEM(key.privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		return KeyMetadata{}, nil, err
	}
	return KeyMetadata{KID: key.kid, Algorithm: "Ed25519", Status: KeyStatusStaged, CreatedAt: now}, publicPEM, nil
}

func (p *ManagedLocalProvider) Activate(_ context.Context, kid string, grace time.Duration) (KeyMetadata, error) {
	if grace < time.Hour || grace > 30*24*time.Hour {
		return KeyMetadata{}, errors.New("keyprovider: grace period must be between 1h and 720h")
	}
	kid = strings.TrimSpace(kid)
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now().UTC()
	p.refreshExpiredLocked(now)
	target, ok := p.manifest.Keys[kid]
	if !ok {
		return KeyMetadata{}, ErrKeyNotFound
	}
	if target.Status != KeyStatusStaged {
		return KeyMetadata{}, ErrKeyNotStaged
	}
	for existingKID, rec := range p.manifest.Keys {
		if rec.Status == KeyStatusActive {
			expires := now.Add(grace)
			rec.Status = KeyStatusGrace
			rec.GraceExpiresAt = &expires
			rec.RevokedAt = nil
			p.manifest.Keys[existingKID] = rec
		}
	}
	target.Status = KeyStatusActive
	target.ActivatedAt = &now
	target.GraceExpiresAt = nil
	target.RevokedAt = nil
	p.manifest.Keys[kid] = target
	if err := p.saveManifestLocked(); err != nil {
		return KeyMetadata{}, err
	}
	return KeyMetadata{KID: kid, Algorithm: "Ed25519", Status: KeyStatusActive, CreatedAt: target.CreatedAt, ActivatedAt: target.ActivatedAt}, nil
}

func (p *ManagedLocalProvider) Revoke(_ context.Context, kid string) (KeyMetadata, error) {
	kid = strings.TrimSpace(kid)
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now().UTC()
	p.refreshExpiredLocked(now)
	rec, ok := p.manifest.Keys[kid]
	if !ok {
		return KeyMetadata{}, ErrKeyNotFound
	}
	if rec.Status == KeyStatusActive {
		return KeyMetadata{}, ErrActiveKey
	}
	if rec.Status != KeyStatusRevoked {
		rec.Status = KeyStatusRevoked
		rec.GraceExpiresAt = nil
		rec.RevokedAt = &now
		p.manifest.Keys[kid] = rec
		if err := p.saveManifestLocked(); err != nil {
			return KeyMetadata{}, err
		}
	}
	return KeyMetadata{KID: kid, Algorithm: "Ed25519", Status: rec.Status, CreatedAt: rec.CreatedAt, ActivatedAt: rec.ActivatedAt, GraceExpiresAt: rec.GraceExpiresAt, RevokedAt: rec.RevokedAt}, nil
}

func (p *ManagedLocalProvider) PublicKeyPEM(_ context.Context, kid string) ([]byte, error) {
	kid = strings.TrimSpace(kid)
	p.mu.RLock()
	defer p.mu.RUnlock()
	key, ok := p.keys[kid]
	if !ok {
		return nil, ErrKeyNotFound
	}
	publicKey, ok := key.privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("keyprovider: invalid Ed25519 public key")
	}
	return publicKeyPEM(publicKey)
}

func randomKeyFileName() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("keyprovider: crypto/rand unavailable")
	}
	return "key-" + hex.EncodeToString(value[:]) + ".pem"
}

func loadLocalEd25519Key(path string) (*localEd25519Key, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: inspect private key: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("keyprovider: signing key must be a regular non-symlink file")
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: read private key: %w", err)
	}
	block, rest := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("keyprovider: signing key must contain exactly one PKCS#8 PRIVATE KEY PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: parse private key: %w", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("keyprovider: signing key must be Ed25519")
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("keyprovider: invalid Ed25519 public key")
	}
	kid, err := Ed25519KID(publicKey)
	if err != nil {
		return nil, err
	}
	return &localEd25519Key{kid: kid, privateKey: append(ed25519.PrivateKey(nil), privateKey...)}, nil
}

func generateLocalEd25519Key() (*localEd25519Key, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	kid, err := Ed25519KID(publicKey)
	if err != nil {
		return nil, err
	}
	return &localEd25519Key{kid: kid, privateKey: privateKey}, nil
}

func GenerateEd25519KeyPair() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	publicPEM, err := publicKeyPEM(publicKey)
	if err != nil {
		return nil, nil, err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	return privatePEM, publicPEM, nil
}

func publicKeyPEM(publicKey ed25519.PublicKey) ([]byte, error) {
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), nil
}

func ValidateKID(kid string) error {
	kid = strings.TrimSpace(kid)
	if len(kid) == 0 || len(kid) > 128 {
		return errors.New("keyprovider: kid must contain between 1 and 128 characters")
	}
	for _, r := range kid {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == ':' || r == '-' {
			continue
		}
		return errors.New("keyprovider: kid contains unsupported characters")
	}
	return nil
}

func Ed25519KID(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("keyprovider: invalid Ed25519 public key")
	}
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("keyprovider: marshal public key: %w", err)
	}
	digest := sha256.Sum256(der)
	return "ed25519-sha256-" + hex.EncodeToString(digest[:]), nil
}
