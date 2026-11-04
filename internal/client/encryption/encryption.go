package encryption

import "github.com/google/uuid"

type User struct {
	Username             string
	EncryptionPrivateKey []byte
	SigningPrivateKey    []byte
	MasterKey            []byte
	PasswordSalt         []byte `json:"-"`
	Files                map[string]uuid.UUID
}

type Inode struct {
	Size       int
	BlockUUIDs []uuid.UUID
}
