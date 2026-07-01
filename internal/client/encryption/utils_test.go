package encryption

import (
	"bytes"
	"testing"

	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"
)

func TestEncryptAndMACRoundtrip(t *testing.T) {
	encKey := userlib.RandomBytes(16)
	macKey := userlib.RandomBytes(16)
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
	encKey := userlib.RandomBytes(16)
	macKey := userlib.RandomBytes(16)

	_, err := decryptAndVerify([]byte("short"), encKey, macKey)
	if err == nil {
		t.Error("expected error for short payload")
	}
}

func TestDecryptAndVerifyTamperedData(t *testing.T) {
	encKey := userlib.RandomBytes(16)
	macKey := userlib.RandomBytes(16)
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
	encKey := userlib.RandomBytes(16)
	macKey := userlib.RandomBytes(16)
	wrongMACKey := userlib.RandomBytes(16)
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

func TestDecryptAndVerifyWrongEncKeyProducesGarbage(t *testing.T) {
	encKey := userlib.RandomBytes(16)
	macKey := userlib.RandomBytes(16)
	wrongEncKey := userlib.RandomBytes(16)
	data := []byte("secret")

	payload, err := encryptAndMAC(data, encKey, macKey)
	if err != nil {
		t.Fatalf("encryptAndMAC: %v", err)
	}

	plaintext, err := decryptAndVerify(payload, wrongEncKey, macKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytes.Equal(plaintext, data) {
		t.Error("wrong encryption key should produce different plaintext")
	}
}

func TestGetPersonalKeyDeterministic(t *testing.T) {
	masterKey := userlib.RandomBytes(16)
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
	fileKey := userlib.RandomBytes(16)

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
	mailboxKey := userlib.RandomBytes(16)

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
	key1 := userlib.RandomBytes(16)
	key2 := userlib.RandomBytes(16)
	data := []byte("same data")

	payload1, _ := encryptAndMAC(data, key1, key1)
	payload2, _ := encryptAndMAC(data, key2, key2)

	if bytes.Equal(payload1, payload2) {
		t.Error("different keys should produce different ciphertexts")
	}
}
