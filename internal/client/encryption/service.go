package encryption

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// UserService handles user lifecycle operations
type UserService struct {
	storage  StorageService
	keyStore KeyStoreService
}

func NewUserService(storage StorageService, keyStore KeyStoreService) *UserService {
	return &UserService{
		storage:  storage,
		keyStore: keyStore,
	}
}

// InitUser creates a new user with cryptographic keys
func (s *UserService) InitUser(username string, password string) (*User, error) {
	userUUID, err := userID(username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user UUID: %w", err)
	}

	_, exists, err := s.storage.Get(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user '%s' already exists", username)
	}

	salt, err := randomBytes(passwordSaltSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password salt: %w", err)
	}
	masterKey := deriveMasterKey([]byte(password), salt)
	encryptionPublicKey, encryptionPrivateKey, err := generateEncryptionKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption keys: %w", err)
	}
	signingPublicKey, signingPrivateKey, err := generateSigningKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing keys: %w", err)
	}

	if err := s.keyStore.Set(username+"_enc_pub", encryptionPublicKey); err != nil {
		return nil, fmt.Errorf("failed to store encryption public key: %w", err)
	}
	if err := s.keyStore.Set(username+"_sig_pub", signingPublicKey); err != nil {
		return nil, fmt.Errorf("failed to store signing public key: %w", err)
	}

	userdata := &User{
		Username:             username,
		EncryptionPrivateKey: encryptionPrivateKey,
		SigningPrivateKey:    signingPrivateKey,
		MasterKey:            masterKey,
		PasswordSalt:         salt,
		Files:                make(map[string]uuid.UUID),
	}

	if err := saveUser(s.storage, userdata); err != nil {
		return nil, fmt.Errorf("failed to save user data: %w", err)
	}
	return userdata, nil
}

// GetUser retrieves and decrypts a user
func (s *UserService) GetUser(username string, password string) (*User, error) {
	userUUID, err := userID(username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user UUID: %w", err)
	}

	payload, ok, err := s.storage.Get(userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user '%s': %w", username, err)
	}
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	if len(payload) < passwordSaltSize {
		return nil, errors.New("malformed user record: password salt is missing")
	}
	salt := payload[:passwordSaltSize]
	masterKey := deriveMasterKey([]byte(password), salt)
	encKey := deriveKey(masterKey, nil, []byte("sharelock/user/encryption"))
	macKey := deriveKey(masterKey, nil, []byte("sharelock/user/authentication"))
	plaintext, err := decryptAndVerify(payload[passwordSaltSize:], encKey, macKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt user data: %w", err)
	}

	var userdata User
	if err := json.Unmarshal(plaintext, &userdata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	if userdata.Username != username {
		return nil, errors.New("decrypted user data does not match requested username")
	}
	userdata.PasswordSalt = append([]byte(nil), salt...)

	return &userdata, nil
}

// FileService handles file operations
type FileService struct {
	storage  StorageService
	keyStore KeyStoreService
}

func NewFileService(storage StorageService, keyStore KeyStoreService) *FileService {
	return &FileService{
		storage:  storage,
		keyStore: keyStore,
	}
}

// StoreFile encrypts and stores a file
func (s *FileService) StoreFile(userdata *User, filename string, content []byte) error {
	if userdata.Files == nil {
		userdata.Files = make(map[string]uuid.UUID)
	}

	fileKey, err := randomBytes(masterKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate file key: %w", err)
	}
	inodeUUID := uuid.New()

	mailboxnode := MailboxNode{
		FileKey:   fileKey,
		InodeUUID: inodeUUID,
	}
	mailboxBytes, err := json.Marshal(mailboxnode)
	if err != nil {
		return fmt.Errorf("failed to marshal mailbox: %w", err)
	}

	mailboxUUID := uuid.New()
	mailboxKey, err := randomBytes(masterKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate mailbox key: %w", err)
	}
	mEncKey, mMacKey := getMailKeys(mailboxKey)
	mailboxPayload, err := encryptAndMAC(mailboxBytes, mEncKey, mMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt mailbox: %w", err)
	}
	if err := s.storage.Set(mailboxUUID, mailboxPayload); err != nil {
		return fmt.Errorf("failed to store mailbox: %w", err)
	}

	access := Access{
		MymailboxUUID: mailboxUUID,
		MymailboxKey:  mailboxKey,
		Children:      make(map[string]ChildrenInfo),
	}

	accessBytes, err := json.Marshal(access)
	if err != nil {
		return fmt.Errorf("failed to marshal access: %w", err)
	}
	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	accessPayload, err := encryptAndMAC(accessBytes, pEncKey, pMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt access: %w", err)
	}

	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	if err := s.storage.Set(accessUUID, accessPayload); err != nil {
		return fmt.Errorf("failed to store access metadata: %w", err)
	}

	fEncKey, fMacKey := getFileKeys(fileKey)
	blocks := ByteToBlock(content)

	inode := Inode{
		Size:       len(content),
		BlockUUIDs: make([]uuid.UUID, 0, blocks.Len()),
	}

	for e := blocks.Front(); e != nil; e = e.Next() {
		fb := e.Value.(*FileBlock)
		blockUUID := uuid.New()

		blockPayload, err := encryptAndMAC(fb.block, fEncKey, fMacKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt block: %w", err)
		}
		if err := s.storage.Set(blockUUID, blockPayload); err != nil {
			return fmt.Errorf("failed to store data block: %w", err)
		}
		inode.BlockUUIDs = append(inode.BlockUUIDs, blockUUID)
	}

	inodeBytes, err := json.Marshal(inode)
	if err != nil {
		return fmt.Errorf("failed to marshal inode: %w", err)
	}
	inodePayload, err := encryptAndMAC(inodeBytes, fEncKey, fMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt inode: %w", err)
	}
	if err := s.storage.Set(inodeUUID, inodePayload); err != nil {
		return fmt.Errorf("failed to store inode: %w", err)
	}

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}

// LoadFile decrypts and loads a file
func (s *FileService) LoadFile(userdata *User, filename string) ([]byte, error) {
	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access UUID: %w", err)
	}

	accessPayload, ok, err := s.storage.Get(accessUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load access metadata: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("file '%s' not found for user '%s'", filename, userdata.Username)
	}

	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access: %w", err)
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return nil, fmt.Errorf("failed to unmarshal access: %w", err)
	}

	mailboxPayload, ok, err := s.storage.Get(access.MymailboxUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load mailbox: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("mailbox not found for file '%s' (data may be corrupted)", filename)
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt mailbox: %w", err)
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mailbox: %w", err)
	}

	inodePayload, ok, err := s.storage.Get(myMailbox.InodeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load inode: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("inode not found for file '%s' (data may be corrupted)", filename)
	}
	fEncKey, fMacKey := getFileKeys(myMailbox.FileKey)
	inodeBytes, err := decryptAndVerify(inodePayload, fEncKey, fMacKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt inode: %w", err)
	}

	var inode Inode
	if err := json.Unmarshal(inodeBytes, &inode); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inode: %w", err)
	}

	var data []byte
	for _, blockUUID := range inode.BlockUUIDs {
		blockPayload, ok, err := s.storage.Get(blockUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to load data block: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("data block not found for file '%s' (data may be corrupted)", filename)
		}

		blockPlaintext, err := decryptAndVerify(blockPayload, fEncKey, fMacKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt block: %w", err)
		}
		data = append(data, blockPlaintext...)
	}

	return data, nil
}

// AppendToFile appends content to an existing file
func (s *FileService) AppendToFile(userdata *User, filename string, content []byte) error {
	if content == nil {
		return errors.New("content cannot be nil")
	}

	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	accessPayload, ok, err := s.storage.Get(accessUUID)
	if err != nil {
		return fmt.Errorf("failed to load access metadata: %w", err)
	}
	if !ok {
		return fmt.Errorf("file '%s' not found (cannot append)", filename)
	}

	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return err
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return err
	}

	mailboxPayload, ok, err := s.storage.Get(access.MymailboxUUID)
	if err != nil {
		return fmt.Errorf("failed to load mailbox: %w", err)
	}
	if !ok {
		return fmt.Errorf("mailbox not found for file '%s' (cannot append)", filename)
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return err
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return err
	}

	inodePayload, ok, err := s.storage.Get(myMailbox.InodeUUID)
	if err != nil {
		return fmt.Errorf("failed to load inode: %w", err)
	}
	if !ok {
		return fmt.Errorf("inode not found for file '%s' (cannot append)", filename)
	}

	fEncKey, fMacKey := getFileKeys(myMailbox.FileKey)
	inodeBytes, err := decryptAndVerify(inodePayload, fEncKey, fMacKey)
	if err != nil {
		return err
	}

	var inode Inode
	if err := json.Unmarshal(inodeBytes, &inode); err != nil {
		return err
	}

	newBlock := ByteToBlock(content)
	for e := newBlock.Front(); e != nil; e = e.Next() {
		fb := e.Value.(*FileBlock)
		blockUUID := uuid.New()

		blockPayload, err := encryptAndMAC(fb.block, fEncKey, fMacKey)
		if err != nil {
			return err
		}
		if err := s.storage.Set(blockUUID, blockPayload); err != nil {
			return fmt.Errorf("failed to store appended data block: %w", err)
		}
		inode.BlockUUIDs = append(inode.BlockUUIDs, blockUUID)
	}
	inode.Size += len(content)

	newInodeBytes, err := json.Marshal(inode)
	if err != nil {
		return fmt.Errorf("failed to marshal inode: %w", err)
	}
	newInodePayload, err := encryptAndMAC(newInodeBytes, fEncKey, fMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt inode: %w", err)
	}
	if err := s.storage.Set(myMailbox.InodeUUID, newInodePayload); err != nil {
		return fmt.Errorf("failed to store updated inode: %w", err)
	}

	return nil
}

// ListFiles returns a list of filenames the user has access to
func (s *FileService) ListFiles(userdata *User) []string {
	if userdata.Files == nil {
		return []string{}
	}
	files := make([]string, 0, len(userdata.Files))
	for filename := range userdata.Files {
		files = append(files, filename)
	}
	return files
}

// InvitationService handles sharing operations
type InvitationService struct {
	storage  StorageService
	keyStore KeyStoreService
}

func NewInvitationService(storage StorageService, keyStore KeyStoreService) *InvitationService {
	return &InvitationService{
		storage:  storage,
		keyStore: keyStore,
	}
}

// CreateInvitation creates a sharing invitation
func (s *InvitationService) CreateInvitation(userdata *User, filename string, recipientUsername string) (uuid.UUID, error) {
	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate access UUID: %w", err)
	}

	accessPayload, ok, err := s.storage.Get(accessUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to load access metadata: %w", err)
	}
	if !ok {
		return uuid.Nil, fmt.Errorf("file '%s' not found for user '%s'", filename, userdata.Username)
	}
	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decrypt access: %w", err)
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return uuid.Nil, fmt.Errorf("failed to unmarshal access: %w", err)
	}

	mailboxPayload, ok, err := s.storage.Get(access.MymailboxUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to load mailbox: %w", err)
	}
	if !ok {
		return uuid.Nil, fmt.Errorf("mailbox not found for file '%s'", filename)
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decrypt mailbox: %w", err)
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return uuid.Nil, fmt.Errorf("failed to unmarshal mailbox: %w", err)
	}

	recipientMailboxUUID := uuid.New()
	recipientMailboxKey, err := randomBytes(masterKeySize)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate recipient mailbox key: %w", err)
	}
	recipientMailbox := MailboxNode{
		FileKey:   myMailbox.FileKey,
		InodeUUID: myMailbox.InodeUUID,
	}
	recipientMailboxBytes, err := json.Marshal(recipientMailbox)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal recipient mailbox: %w", err)
	}
	rmEncKey, rmMacKey := getMailKeys(recipientMailboxKey)
	recipientMailboxPayload, err := encryptAndMAC(recipientMailboxBytes, rmEncKey, rmMacKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt recipient mailbox: %w", err)
	}
	if err := s.storage.Set(recipientMailboxUUID, recipientMailboxPayload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to store recipient mailbox: %w", err)
	}

	recipientPubkey, ok, err := s.keyStore.Get(recipientUsername + "_enc_pub")
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to load recipient public key: %w", err)
	}
	if !ok {
		return uuid.Nil, fmt.Errorf("recipient user '%s' does not exist", recipientUsername)
	}

	invitation := Invitation{
		MailboxUUID: recipientMailboxUUID,
		MailboxKey:  recipientMailboxKey,
	}
	invBytes, err := json.Marshal(invitation)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal invitation: %w", err)
	}

	ciphertext, err := encryptFor(recipientPubkey, invBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt invitation: %w", err)
	}

	signature, err := sign(userdata.SigningPrivateKey, ciphertext)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to sign invitation: %w", err)
	}

	invPayload := append(ciphertext, signature...)
	invRandomPtr := uuid.New()
	if err := s.storage.Set(invRandomPtr, invPayload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to store invitation: %w", err)
	}

	access.Children[recipientUsername] = ChildrenInfo{
		MailboxUUID: recipientMailboxUUID,
		MailboxKey:  recipientMailboxKey,
	}
	newAccessBytes, err := json.Marshal(access)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal access: %w", err)
	}
	newAccessPayload, err := encryptAndMAC(newAccessBytes, pEncKey, pMacKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt access: %w", err)
	}
	if err := s.storage.Set(accessUUID, newAccessPayload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to update access metadata: %w", err)
	}

	return invRandomPtr, nil
}

// AcceptInvitation accepts a sharing invitation
func (s *InvitationService) AcceptInvitation(userdata *User, senderUsername string, invitationPtr uuid.UUID, filename string) error {
	invitationPayload, ok, err := s.storage.Get(invitationPtr)
	if err != nil {
		return fmt.Errorf("failed to load invitation: %w", err)
	}
	if !ok {
		return fmt.Errorf("invitation %s not found", invitationPtr.String())
	}

	if len(invitationPayload) < ed25519.SignatureSize {
		return errors.New("invitation payload too short")
	}
	sigOffset := len(invitationPayload) - ed25519.SignatureSize
	ciphertext := invitationPayload[:sigOffset]
	signature := invitationPayload[sigOffset:]

	senderVerifyKey, ok, err := s.keyStore.Get(senderUsername + "_sig_pub")
	if err != nil {
		return fmt.Errorf("failed to load sender public key: %w", err)
	}
	if !ok {
		return fmt.Errorf("sender '%s' public key not found in keystore", senderUsername)
	}
	if err := verify(senderVerifyKey, ciphertext, signature); err != nil {
		return fmt.Errorf("invitation signature verification failed for sender '%s'", senderUsername)
	}

	plaintext, err := decryptFor(userdata.EncryptionPrivateKey, ciphertext)
	if err != nil {
		return fmt.Errorf("failed to decrypt invitation: %w", err)
	}

	var getInvitation Invitation
	if err := json.Unmarshal(plaintext, &getInvitation); err != nil {
		return fmt.Errorf("failed to unmarshal invitation: %w", err)
	}

	receiveAccess := Access{
		MymailboxUUID: getInvitation.MailboxUUID,
		MymailboxKey:  getInvitation.MailboxKey,
		Children:      make(map[string]ChildrenInfo),
	}
	currAccessPayload, err := json.Marshal(receiveAccess)
	if err != nil {
		return fmt.Errorf("failed to marshal access: %w", err)
	}
	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	currAccessBytes, err := encryptAndMAC(currAccessPayload, pEncKey, pMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt access: %w", err)
	}
	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	if err := s.storage.Set(accessUUID, currAccessBytes); err != nil {
		return fmt.Errorf("failed to store access metadata: %w", err)
	}

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}

// RevokeAccess revokes a user's access
func (s *InvitationService) RevokeAccess(userdata *User, filename string, recipientUsername string) error {
	accessUUID, err := accessID(userdata.Username, filename)
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	accessPayload, ok, err := s.storage.Get(accessUUID)
	if err != nil {
		return fmt.Errorf("failed to load access metadata: %w", err)
	}
	if !ok {
		return fmt.Errorf("file '%s' not found for user '%s'", filename, userdata.Username)
	}
	pEncKey, pMacKey := getPersonalKey(userdata.MasterKey, filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt access: %w", err)
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return fmt.Errorf("failed to unmarshal access: %w", err)
	}

	mailboxPayload, ok, err := s.storage.Get(access.MymailboxUUID)
	if err != nil {
		return fmt.Errorf("failed to load mailbox: %w", err)
	}
	if !ok {
		return fmt.Errorf("mailbox not found for file '%s'", filename)
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt mailbox: %w", err)
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return fmt.Errorf("failed to unmarshal mailbox: %w", err)
	}

	newFileKey, err := randomBytes(masterKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate replacement file key: %w", err)
	}
	newFEncKey, newFMacKey := getFileKeys(newFileKey)

	inodePayload, ok, err := s.storage.Get(myMailbox.InodeUUID)
	if err != nil {
		return fmt.Errorf("failed to load inode: %w", err)
	}
	if !ok {
		return fmt.Errorf("inode not found for file '%s'", filename)
	}
	oldFEncKey, oldFMacKey := getFileKeys(myMailbox.FileKey)
	inodeBytes, err := decryptAndVerify(inodePayload, oldFEncKey, oldFMacKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt inode: %w", err)
	}

	var inode Inode
	if err := json.Unmarshal(inodeBytes, &inode); err != nil {
		return fmt.Errorf("failed to unmarshal inode: %w", err)
	}

	for _, blockUUID := range inode.BlockUUIDs {
		blockPayload, ok, err := s.storage.Get(blockUUID)
		if err != nil {
			return fmt.Errorf("failed to load data block: %w", err)
		}
		if !ok {
			return fmt.Errorf("data block not found for file '%s'", filename)
		}
		blockPlaintext, err := decryptAndVerify(blockPayload, oldFEncKey, oldFMacKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt block: %w", err)
		}
		newBlockPayload, err := encryptAndMAC(blockPlaintext, newFEncKey, newFMacKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt block: %w", err)
		}
		if err := s.storage.Set(blockUUID, newBlockPayload); err != nil {
			return fmt.Errorf("failed to store re-encrypted data block: %w", err)
		}
	}

	newInodeBytes, err := json.Marshal(inode)
	if err != nil {
		return fmt.Errorf("failed to marshal inode: %w", err)
	}
	newInodePayload, err := encryptAndMAC(newInodeBytes, newFEncKey, newFMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt inode: %w", err)
	}
	if err := s.storage.Set(myMailbox.InodeUUID, newInodePayload); err != nil {
		return fmt.Errorf("failed to store re-encrypted inode: %w", err)
	}

	newMailboxKey, err := randomBytes(masterKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate replacement mailbox key: %w", err)
	}
	newMailboxUUID := uuid.New()
	newMailbox := MailboxNode{
		FileKey:   newFileKey,
		InodeUUID: myMailbox.InodeUUID,
	}
	newMailboxBytes, err := json.Marshal(newMailbox)
	if err != nil {
		return fmt.Errorf("failed to marshal mailbox: %w", err)
	}
	nmEncKey, nmMacKey := getMailKeys(newMailboxKey)
	newMailboxPayload, err := encryptAndMAC(newMailboxBytes, nmEncKey, nmMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt mailbox: %w", err)
	}
	if err := s.storage.Set(newMailboxUUID, newMailboxPayload); err != nil {
		return fmt.Errorf("failed to store replacement mailbox: %w", err)
	}

	for childUsername, childInfo := range access.Children {
		if childUsername == recipientUsername {
			continue
		}
		childMailboxPayload, ok, err := s.storage.Get(childInfo.MailboxUUID)
		if err != nil {
			return fmt.Errorf("failed to load shared mailbox for '%s': %w", childUsername, err)
		}
		if !ok {
			return fmt.Errorf("shared mailbox for '%s' not found", childUsername)
		}
		cmEncKey, cmMacKey := getMailKeys(childInfo.MailboxKey)
		childMailboxBytes, err := decryptAndVerify(childMailboxPayload, cmEncKey, cmMacKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt shared mailbox for '%s': %w", childUsername, err)
		}

		var childMailbox MailboxNode
		if err := json.Unmarshal(childMailboxBytes, &childMailbox); err != nil {
			return fmt.Errorf("failed to unmarshal shared mailbox for '%s': %w", childUsername, err)
		}
		childMailbox.FileKey = newFileKey
		updatedChildMailboxBytes, err := json.Marshal(childMailbox)
		if err != nil {
			return fmt.Errorf("failed to marshal child mailbox: %w", err)
		}
		updatedChildMailboxPayload, err := encryptAndMAC(updatedChildMailboxBytes, cmEncKey, cmMacKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt child mailbox: %w", err)
		}
		if err := s.storage.Set(childInfo.MailboxUUID, updatedChildMailboxPayload); err != nil {
			return fmt.Errorf("failed to update shared mailbox for '%s': %w", childUsername, err)
		}
	}

	delete(access.Children, recipientUsername)
	access.MymailboxUUID = newMailboxUUID
	access.MymailboxKey = newMailboxKey

	newAccessBytes, err := json.Marshal(access)
	if err != nil {
		return fmt.Errorf("failed to marshal access: %w", err)
	}
	newAccessPayload, err := encryptAndMAC(newAccessBytes, pEncKey, pMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt access: %w", err)
	}
	if err := s.storage.Set(accessUUID, newAccessPayload); err != nil {
		return fmt.Errorf("failed to update access metadata: %w", err)
	}

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}
