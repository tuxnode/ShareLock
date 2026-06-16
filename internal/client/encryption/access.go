package encryption

import (
	userlib "github.com/cs161-staff/project2-userlib"
)

/* Mailbox Node structure */
type MailboxNode struct {
	FileKey   []byte
	InodeUUID userlib.UUID
}

type Access struct {
	MymailboxUUID userlib.UUID
	MymailboxKey  []byte
	Chidren       map[string]ChildrenInfo // Sharing Tree
}

type ChildrenInfo struct {
	MailboxUUID userlib.UUID
	MailboxKey  []byte
}

type Invitation struct {
	MailboxUUID userlib.UUID
	MailboxKey  []byte
}
