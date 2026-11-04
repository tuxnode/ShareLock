package encryption

import "github.com/google/uuid"

/* Mailbox Node structure */
type MailboxNode struct {
	FileKey   []byte
	InodeUUID uuid.UUID
}

type Access struct {
	MymailboxUUID uuid.UUID
	MymailboxKey  []byte
	Children      map[string]ChildrenInfo // Sharing Tree
}

type ChildrenInfo struct {
	MailboxUUID uuid.UUID
	MailboxKey  []byte
}

type Invitation struct {
	MailboxUUID uuid.UUID
	MailboxKey  []byte
}
