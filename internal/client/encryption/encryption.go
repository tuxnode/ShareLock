package encryption

import (
	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"
)

type User struct {
	Username      string
	PKEPrivateKey userlib.PrivateKeyType
	DSSignKey     userlib.DSSignKey
	MasterKey     []byte
	Files         map[string]userlib.UUID
}

type Inode struct {
	Size       int
	BlockUUIDs []userlib.UUID
}
