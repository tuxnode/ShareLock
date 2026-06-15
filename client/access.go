package client

import (
	userlib "github.com/cs161-staff/project2-userlib"
	"github.com/google/uuid"
)

type Access struct {
	FileKey   []byte
	InodeUUID userlib.UUID
}

func (userdata *User) CreateInvitation(filename string, recipientUsername string) (
	invitationPtr uuid.UUID, err error) {
	return
}

func (userdata *User) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	return nil
}

func (userdata *User) RevokeAccess(filename string, recipientUsername string) error {
	return nil
}
