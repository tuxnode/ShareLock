package encryption

import (
	"bytes"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key, err := randomBytes(masterKeySize)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestEncryptAndMACRoundtrip(t *testing.T) {
	encKey := testKey(t)
	macKey := testKey(t)
	data := []byte("hello world")

	payload, err := encryptAndMAC(data, encKey, macKey)
	if err != nil {
		t.Fatalf("encryptAndMAC: %v", err)
	}

	plaintext, err := decryptAndVerify(payload, encKey, macKey)
	if err != nil {
		t.Fatalf("decryptAndVerify: %v", err)
	}

	if !bytes.Equal(plaintext, data) {
		t.Errorf("got %q, want %q", plaintext, data)
	}
}

func TestDecryptAndVerifyShortPayload(t *testing.T) {
	encKey := testKey(t)
	macKey := testKey(t)

	_, err := decryptAndVerify([]byte("short"), encKey, macKey)
	if err == nil {
		t.Error("expected error for short payload")
	}
}

func TestDecryptAndVerifyTamperedData(t *testing.T) {
	encKey := testKey(t)
	macKey := testKey(t)
	data := []byte("sensitive data")

	payload, err := encryptAndMAC(data, encKey, macKey)
	if err != nil {
		t.Fatalf("encryptAndMAC: %v", err)
	}

	payload[0] ^= 0xFF

	_, err = decryptAndVerify(payload, encKey, macKey)
	if err == nil {
		t.Error("expected error for tampered data")
	}
}

func TestDecryptAndVerifyWrongMACKey(t *testing.T) {
	encKey := testKey(t)
	macKey := testKey(t)
	wrongMACKey := testKey(t)
	data := []byte("secret")

	payload, err := encryptAndMAC(data, encKey, macKey)
	if err != nil {
		t.Fatalf("encryptAndMAC: %v", err)
	}

	_, err = decryptAndVerify(payload, encKey, wrongMACKey)
	if err == nil {
		t.Error("expected error for wrong MAC key")
	}
}

func TestDecryptAndVerifyWrongEncryptionKey(t *testing.T) {
	encKey := testKey(t)
	macKey := testKey(t)
	wrongEncKey := testKey(t)
	data := []byte("secret")

	payload, err := encryptAndMAC(data, encKey, macKey)
	if err != nil {
		t.Fatalf("encryptAndMAC: %v", err)
	}

	if _, err := decryptAndVerify(payload, wrongEncKey, macKey); err == nil {
		t.Fatal("expected authentication failure with wrong encryption key")
	}
}

func TestGetPersonalKeyDeterministic(t *testing.T) {
	masterKey := testKey(t)
	filename := "test.txt"

	enc1, mac1 := getPersonalKey(masterKey, filename)
	enc2, mac2 := getPersonalKey(masterKey, filename)

	if !bytes.Equal(enc1, enc2) {
		t.Error("getPersonalKey not deterministic for encKey")
	}
	if !bytes.Equal(mac1, mac2) {
		t.Error("getPersonalKey not deterministic for macKey")
	}
}

func TestGetFileKeysDeterministic(t *testing.T) {
	fileKey := testKey(t)

	enc1, mac1 := getFileKeys(fileKey)
	enc2, mac2 := getFileKeys(fileKey)

	if !bytes.Equal(enc1, enc2) {
		t.Error("getFileKeys not deterministic for encKey")
	}
	if !bytes.Equal(mac1, mac2) {
		t.Error("getFileKeys not deterministic for macKey")
	}
}

func TestGetMailKeysDeterministic(t *testing.T) {
	mailboxKey := testKey(t)

	enc1, mac1 := getMailKeys(mailboxKey)
	enc2, mac2 := getMailKeys(mailboxKey)

	if !bytes.Equal(enc1, enc2) {
		t.Error("getMailKeys not deterministic for encKey")
	}
	if !bytes.Equal(mac1, mac2) {
		t.Error("getMailKeys not deterministic for macKey")
	}
}

func TestDifferentKeysProduceDifferentOutput(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)
	data := []byte("same data")

	payload1, _ := encryptAndMAC(data, key1, key1)
	payload2, _ := encryptAndMAC(data, key2, key2)

	if bytes.Equal(payload1, payload2) {
		t.Error("different keys should produce different ciphertexts")
	}
}

func TestInvitationEncryptionAndSignature(t *testing.T) {
	encryptionPublic, encryptionPrivate, err := generateEncryptionKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	signingPublic, signingPrivate, err := generateSigningKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("invitation")

	ciphertext, err := encryptFor(encryptionPublic, message)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := sign(signingPrivate, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if err := verify(signingPublic, ciphertext, signature); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	plaintext, err := decryptFor(encryptionPrivate, ciphertext)
	if err != nil {
		t.Fatalf("decrypt invitation: %v", err)
	}
	if !bytes.Equal(plaintext, message) {
		t.Fatalf("got %q, want %q", plaintext, message)
	}

	ciphertext[0] ^= 1
	if err := verify(signingPublic, ciphertext, signature); err == nil {
		t.Fatal("expected tampered invitation signature to fail")
	}
}
