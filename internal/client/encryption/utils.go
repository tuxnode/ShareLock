package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltSize = 16
	masterKeySize    = 32
)

func randomBytes(size int) ([]byte, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	return data, nil
}

func hash(data []byte) []byte {
	sum := sha512.Sum512(data)
	return sum[:]
}

func deriveKey(secret, salt, info []byte) []byte {
	extract := hmac.New(sha256.New, salt)
	_, _ = extract.Write(secret)
	prk := extract.Sum(nil)

	expand := hmac.New(sha256.New, prk)
	_, _ = expand.Write(info)
	_, _ = expand.Write([]byte{1})
	return expand.Sum(nil)
}

func encryptAndMAC(data, encKey, macKey []byte) ([]byte, error) {
	if len(encKey) == 0 || len(macKey) == 0 {
		return nil, errors.New("encryption and authentication keys must not be empty")
	}
	key := deriveKey(encKey, macKey, []byte("sharelock/data/aes-gcm"))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 1, 1+len(nonce)+len(data)+aead.Overhead())
	payload[0] = 1
	payload = append(payload, nonce...)
	payload = aead.Seal(payload, nonce, data, nil)
	return payload, nil
}

func decryptAndVerify(payload, encKey, macKey []byte) ([]byte, error) {
	if len(encKey) == 0 || len(macKey) == 0 {
		return nil, errors.New("encryption and authentication keys must not be empty")
	}
	if len(payload) < 1 || payload[0] != 1 {
		return nil, errors.New("unsupported or malformed encrypted payload")
	}
	key := deriveKey(encKey, macKey, []byte("sharelock/data/aes-gcm"))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	minimumSize := 1 + aead.NonceSize() + aead.Overhead()
	if len(payload) < minimumSize {
		return nil, errors.New("malformed encrypted payload: too short")
	}
	nonce := payload[1 : 1+aead.NonceSize()]
	plaintext, err := aead.Open(nil, nonce, payload[1+aead.NonceSize():], nil)
	if err != nil {
		return nil, fmt.Errorf("authenticated decryption failed: %w", err)
	}
	return plaintext, nil
}

func generateEncryptionKeyPair() (publicKey, privateKey []byte, err error) {
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate X25519 key pair: %w", err)
	}
	return private.PublicKey().Bytes(), private.Bytes(), nil
}

func generateSigningKeyPair() (publicKey, privateKey []byte, err error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate Ed25519 key pair: %w", err)
	}
	return public, private, nil
}

func encryptFor(publicKey, plaintext []byte) ([]byte, error) {
	recipientKey, err := ecdh.X25519().NewPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("parse recipient X25519 public key: %w", err)
	}
	ephemeralKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral X25519 key: %w", err)
	}
	sharedSecret, err := ephemeralKey.ECDH(recipientKey)
	if err != nil {
		return nil, fmt.Errorf("derive invitation key: %w", err)
	}
	key := deriveKey(sharedSecret, ephemeralKey.PublicKey().Bytes(), []byte("sharelock/invitation/aes-gcm"))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create invitation cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create invitation AES-GCM: %w", err)
	}
	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, 0, 32+len(nonce)+len(plaintext)+aead.Overhead())
	ciphertext = append(ciphertext, ephemeralKey.PublicKey().Bytes()...)
	ciphertext = append(ciphertext, nonce...)
	ciphertext = aead.Seal(ciphertext, nonce, plaintext, ephemeralKey.PublicKey().Bytes())
	return ciphertext, nil
}

func decryptFor(privateKey, ciphertext []byte) ([]byte, error) {
	const publicKeySize = 32
	if len(ciphertext) < publicKeySize {
		return nil, errors.New("invitation ciphertext is too short")
	}
	recipientKey, err := ecdh.X25519().NewPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse recipient X25519 private key: %w", err)
	}
	ephemeralPublic := ciphertext[:publicKeySize]
	ephemeralKey, err := ecdh.X25519().NewPublicKey(ephemeralPublic)
	if err != nil {
		return nil, fmt.Errorf("parse ephemeral X25519 public key: %w", err)
	}
	sharedSecret, err := recipientKey.ECDH(ephemeralKey)
	if err != nil {
		return nil, fmt.Errorf("derive invitation key: %w", err)
	}
	key := deriveKey(sharedSecret, ephemeralPublic, []byte("sharelock/invitation/aes-gcm"))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create invitation cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create invitation AES-GCM: %w", err)
	}
	if len(ciphertext) < publicKeySize+aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("invitation ciphertext is too short")
	}
	nonceStart := publicKeySize
	nonce := ciphertext[nonceStart : nonceStart+aead.NonceSize()]
	plaintext, err := aead.Open(nil, nonce, ciphertext[nonceStart+aead.NonceSize():], ephemeralPublic)
	if err != nil {
		return nil, fmt.Errorf("decrypt invitation: %w", err)
	}
	return plaintext, nil
}

func sign(privateKey, message []byte) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}
	return ed25519.Sign(ed25519.PrivateKey(privateKey), message), nil
}

func verify(publicKey, message, signature []byte) error {
	if len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(publicKey), message, signature) {
		return errors.New("invalid Ed25519 signature")
	}
	return nil
}

func saveUser(storage StorageService, userdata *User) error {
	userBytes, err := json.Marshal(userdata)
	if err != nil {
		return err
	}
	encKey := deriveKey(userdata.MasterKey, nil, []byte("sharelock/user/encryption"))
	macKey := deriveKey(userdata.MasterKey, nil, []byte("sharelock/user/authentication"))
	sealedUser, err := encryptAndMAC(userBytes, encKey, macKey)
	if err != nil {
		return err
	}
	if len(userdata.PasswordSalt) != passwordSaltSize {
		return errors.New("invalid password salt")
	}
	payload := append(append([]byte(nil), userdata.PasswordSalt...), sealedUser...)
	id, err := userID(userdata.Username)
	if err != nil {
		return err
	}
	return storage.Set(id, payload)
}

func userID(username string) (uuid.UUID, error) {
	digest := hash(append([]byte("sharelock/user-id/"), []byte(username)...))
	return uuid.FromBytes(digest[:16])
}

func accessID(username, filename string) (uuid.UUID, error) {
	input := append([]byte("sharelock/access/"), []byte(username)...)
	input = append(input, 0)
	input = append(input, []byte(filename)...)
	digest := hash(input)
	return uuid.FromBytes(digest[:16])
}

func getPersonalKey(masterKey []byte, filename string) ([]byte, []byte) {
	salt := []byte(filename)
	return deriveKey(masterKey, salt, []byte("sharelock/personal/encryption")),
		deriveKey(masterKey, salt, []byte("sharelock/personal/authentication"))
}

func getFileKeys(fileKey []byte) ([]byte, []byte) {
	return deriveKey(fileKey, nil, []byte("sharelock/file/encryption")),
		deriveKey(fileKey, nil, []byte("sharelock/file/authentication"))
}

func getMailKeys(mailboxKey []byte) ([]byte, []byte) {
	return deriveKey(mailboxKey, nil, []byte("sharelock/mailbox/encryption")),
		deriveKey(mailboxKey, nil, []byte("sharelock/mailbox/authentication"))
}

func deriveMasterKey(password, salt []byte) []byte {
	return argon2Key(password, salt, masterKeySize)
}

func argon2Key(password, salt []byte, keyLen uint32) []byte {
	return argon2.IDKey(password, salt, 3, 64*1024, 4, keyLen)
}
