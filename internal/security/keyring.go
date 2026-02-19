package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "opendan"
	vaultFile      = "vault.enc"
)

// KeyStore manages secure storage of API keys.
// Primary: OS Keychain. Fallback: encrypted file.
type KeyStore struct {
	encryptionKey []byte // derived from master password
	vaultPath     string
}

// NewKeyStore creates a key store.
// masterKey is the AES key derived from master password (may be nil if using keyring only).
func NewKeyStore(masterKey []byte) (*KeyStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".opendan")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &KeyStore{
		encryptionKey: masterKey,
		vaultPath:     filepath.Join(dir, vaultFile),
	}, nil
}

// Set stores a secret (tries keyring first, falls back to encrypted file).
func (ks *KeyStore) Set(name, value string) error {
	// Try OS keyring first
	if err := keyring.Set(keyringService, name, value); err == nil {
		return nil
	}

	// Fallback: encrypted vault file
	return ks.setInVault(name, value)
}

// Get retrieves a secret.
func (ks *KeyStore) Get(name string) (string, error) {
	// Try OS keyring first
	if val, err := keyring.Get(keyringService, name); err == nil {
		return val, nil
	}

	// Fallback: encrypted vault file
	return ks.getFromVault(name)
}

// Delete removes a secret.
func (ks *KeyStore) Delete(name string) error {
	_ = keyring.Delete(keyringService, name)
	return ks.deleteFromVault(name)
}

// MaskKey returns a masked version of an API key for display.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

// Vault operations (encrypted JSON file)
func (ks *KeyStore) loadVault() (map[string]string, error) {
	data, err := os.ReadFile(ks.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	if ks.encryptionKey == nil {
		return nil, fmt.Errorf("no encryption key set")
	}

	plaintext, err := Decrypt(string(data), ks.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault: %w", err)
	}

	var vault map[string]string
	if err := json.Unmarshal(plaintext, &vault); err != nil {
		return nil, fmt.Errorf("parse vault: %w", err)
	}
	return vault, nil
}

func (ks *KeyStore) saveVault(vault map[string]string) error {
	if ks.encryptionKey == nil {
		return fmt.Errorf("no encryption key set")
	}

	data, err := json.Marshal(vault)
	if err != nil {
		return err
	}

	encrypted, err := Encrypt(data, ks.encryptionKey)
	if err != nil {
		return err
	}

	return os.WriteFile(ks.vaultPath, []byte(encrypted), 0600)
}

func (ks *KeyStore) setInVault(name, value string) error {
	vault, err := ks.loadVault()
	if err != nil {
		vault = make(map[string]string)
	}
	vault[name] = value
	return ks.saveVault(vault)
}

func (ks *KeyStore) getFromVault(name string) (string, error) {
	vault, err := ks.loadVault()
	if err != nil {
		return "", err
	}
	val, ok := vault[name]
	if !ok {
		return "", fmt.Errorf("key not found: %s", name)
	}
	return val, nil
}

func (ks *KeyStore) deleteFromVault(name string) error {
	vault, err := ks.loadVault()
	if err != nil {
		return nil // nothing to delete
	}
	delete(vault, name)
	return ks.saveVault(vault)
}
