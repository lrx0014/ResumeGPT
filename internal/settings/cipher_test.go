package settings

import "testing"

func TestAESGCMTokenCipherRoundTrip(t *testing.T) {
	cipher, err := NewAESGCMTokenCipher("test-settings-encryption-key-at-least-32-characters")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if string(encrypted) == "secret-token" {
		t.Fatal("API token was stored as plaintext")
	}
	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil || decrypted != "secret-token" {
		t.Fatalf("decrypted token = %q, error = %v", decrypted, err)
	}
}

func TestAESGCMTokenCipherRejectsShortKey(t *testing.T) {
	if _, err := NewAESGCMTokenCipher("short"); err == nil {
		t.Fatal("expected a short encryption key to be rejected")
	}
}
