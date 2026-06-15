package client

import (
	"encoding/json"

	userlib "github.com/cs161-staff/project2-userlib"
	"github.com/google/uuid"
)

func (userdata *User) saveUser() error {
	userBytes, _ := json.Marshal(userdata)

	encKey, err := userlib.HashKDF(userdata.MasterKey, []byte("enc"))
	if err != nil {
		return err
	}
	macKey, err := userlib.HashKDF(userdata.MasterKey, []byte("mac"))

	paylaod, _ := encryptAndMAC(userBytes, encKey[:16], macKey[:16])
	hash := userlib.Hash([]byte(userdata.Username + "userStruct"))

	// generate useruuid by username hash
	userUUID, _ := uuid.FromBytes(hash[:16])

	userlib.DatastoreSet(userUUID, paylaod)
	return nil
}
