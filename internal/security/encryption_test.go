package security

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}

	key := DeriveKey("test-password", salt)
	plaintext := []byte("super secret API key sk-abc123")

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if encrypted == string(plaintext) {
		t.Fatal("encrypted should differ from plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("password1", salt)
	key2 := DeriveKey("password2", salt)

	encrypted, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("fixed-salt-value")
	key1 := DeriveKey("password", salt)
	key2 := DeriveKey("password", salt)

	if !bytes.Equal(key1, key2) {
		t.Fatal("same password and salt should produce same key")
	}
}
