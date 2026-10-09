package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func newKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, KeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand key: %v", err)
	}
	return k
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c, err := New(newKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := []byte("sensitive-field-value-42")
	token, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := c.Decrypt(token)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: got %q, want %q", got, plain)
	}
}

func TestEncrypt_FreshNoncePerCall(t *testing.T) {
	c, _ := New(newKey(t))
	a, _ := c.Encrypt([]byte("same"))
	b, _ := c.Encrypt([]byte("same"))
	if a == b {
		t.Fatal("identical ciphertext for same plaintext — nonce not random")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	enc, _ := New(newKey(t))
	token, _ := enc.Encrypt([]byte("secret"))

	dec, _ := New(newKey(t)) // different key
	if _, err := dec.Decrypt(token); err == nil {
		t.Fatal("decrypt with wrong key must fail")
	}
}

func TestDecrypt_TamperedFails(t *testing.T) {
	c, _ := New(newKey(t))
	token, _ := c.Encrypt([]byte("secret"))
	raw, _ := base64.StdEncoding.DecodeString(token)
	raw[len(raw)-1] ^= 0xFF // flip a tag bit
	if _, err := c.Decrypt(base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("decrypt of tampered ciphertext must fail")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	c, _ := New(newKey(t))
	if _, err := c.Decrypt(base64.StdEncoding.EncodeToString([]byte("x"))); err == nil {
		t.Fatal("decrypt of too-short input must fail")
	}
}

func TestNew_RejectsBadKeySize(t *testing.T) {
	if _, err := New(make([]byte, 16)); err == nil {
		t.Fatal("New must reject a 16-byte key")
	}
}

func TestNewFromBase64(t *testing.T) {
	key := newKey(t)
	c, err := NewFromBase64(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("NewFromBase64: %v", err)
	}
	token, _ := c.Encrypt([]byte("hi"))
	if _, err := c.Decrypt(token); err != nil {
		t.Fatalf("round-trip after base64 key: %v", err)
	}
	if _, err := NewFromBase64("not!base64"); err == nil {
		t.Fatal("NewFromBase64 must reject invalid base64")
	}
}
