package encryption

import (
	"encoding/json"
	"errors"
	"fmt"

	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"
	"github.com/google/uuid"
)

// StorageService abstracts storage operations for testability and flexibility
type StorageService interface {
	Get(key userlib.UUID) ([]byte, bool)
	Set(key userlib.UUID, value []byte)
	Delete(key userlib.UUID)
}

// KeyStoreService abstracts public key storage
type KeyStoreService interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

// UserService handles user lifecycle operations
type UserService struct {
	storage   StorageService
	keyStore  KeyStoreService
}

func NewUserService(storage StorageService, keyStore KeyStoreService) *UserService {
	return &UserService{
		storage:  storage,
		keyStore: keyStore,
	}
}

// InitUser creates a new user with cryptographic keys
func (s *UserService) InitUser(username string, password string) (*User, error) {
	hash := userlib.Hash([]byte(username + "userStruct"))
	userUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		return nil, fmt.Errorf("failed to generate user UUID: %w", err)
	}

	if _, ok := s.storage.Get(userUUID); ok {
		return nil, fmt.Errorf("user '%s' already exists", username)
	}

	salt := userlib.Hash([]byte(username))
	masterKey := userlib.Argon2Key([]byte(password), salt, userlib.AESKeySizeBytes)

	encKey, err := userlib.HashKDF(masterKey, []byte("enc"))
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	macKey, err := userlib.HashKDF(masterKey, []byte("mac"))
	if err != nil {
		return nil, fmt.Errorf("failed to derive MAC key: %w", err)
	}

	pkeEncKey, pkeDecKey, err := userlib.PKEKeyGen()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKE keys: %w", err)
	}

	dsSignKey, dsVerifyKey, err := userlib.DSKeyGen()
	if err != nil {
		return nil, fmt.Errorf("failed to generate DS keys: %w", err)
	}

	s.keyStore.Set(username+"_enc_pub", pkeEncKey)
	s.keyStore.Set(username+"_sig_pub", dsVerifyKey)

	userdata := &User{
		Username:      username,
		PKEPrivateKey: pkeDecKey,
		DSSignKey:     dsSignKey,
		MasterKey:     masterKey,
		Files:         make(map[string]userlib.UUID),
	}

	userBytes, err := json.Marshal(userdata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	payload, err := encryptAndMAC(userBytes, encKey, macKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt user data: %w", err)
	}

	s.storage.Set(userUUID, payload)
	return userdata, nil
}

// GetUser retrieves and decrypts a user
func (s *UserService) GetUser(username string, password string) (*User, error) {
	salt := userlib.Hash([]byte(username))
	masterKey := userlib.Argon2Key([]byte(password), salt, userlib.AESKeySizeBytes)

	encKey, err := userlib.HashKDF(masterKey, []byte("enc"))
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	macKey, err := userlib.HashKDF(masterKey, []byte("mac"))
	if err != nil {
		return nil, fmt.Errorf("failed to derive MAC key: %w", err)
	}

	hash := userlib.Hash([]byte(username + "userStruct"))
	userUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		return nil, fmt.Errorf("failed to generate user UUID: %w", err)
	}

	payload, ok := s.storage.Get(userUUID)
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	plaintext, err := decryptAndVerify(payload, encKey, macKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt user data: %w", err)
	}

	var userdata User
	if err := json.Unmarshal(plaintext, &userdata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

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
		userdata.Files = make(map[string]userlib.UUID)
	}

	fileKey := userlib.RandomBytes(16)
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
	mailboxKey := userlib.RandomBytes(16)
	mEncKey, mMacKey := getMailKeys(mailboxKey)
	mailboxPayload, err := encryptAndMAC(mailboxBytes, mEncKey, mMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt mailbox: %w", err)
	}
	s.storage.Set(mailboxUUID, mailboxPayload)

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

	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	s.storage.Set(accessUUID, accessPayload)

	fEncKey, fMacKey := getFileKeys(fileKey)
	blocks := ByteToBlock(content)

	inode := Inode{
		Size:       len(content),
		BlockUUIDs: make([]userlib.UUID, 0, blocks.Len()),
	}

	for e := blocks.Front(); e != nil; e = e.Next() {
		fb := e.Value.(*FileBlock)
		blockUUID := uuid.New()

		blockPayload, err := encryptAndMAC(fb.block, fEncKey, fMacKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt block: %w", err)
		}
		s.storage.Set(blockUUID, blockPayload)
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
	s.storage.Set(inodeUUID, inodePayload)

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}

// LoadFile decrypts and loads a file
func (s *FileService) LoadFile(userdata *User, filename string) ([]byte, error) {
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return nil, fmt.Errorf("failed to generate access UUID: %w", err)
	}

	accessPayload, ok := s.storage.Get(accessUUID)
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

	mailboxPayload, ok := s.storage.Get(access.MymailboxUUID)
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

	inodePayload, ok := s.storage.Get(myMailbox.InodeUUID)
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
		blockPayload, ok := s.storage.Get(blockUUID)
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

	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	accessPayload, ok := s.storage.Get(accessUUID)
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

	mailboxPayload, ok := s.storage.Get(access.MymailboxUUID)
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

	inodePayload, ok := s.storage.Get(myMailbox.InodeUUID)
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
		s.storage.Set(blockUUID, blockPayload)
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
	s.storage.Set(myMailbox.InodeUUID, newInodePayload)

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
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate access UUID: %w", err)
	}

	accessPayload, ok := s.storage.Get(accessUUID)
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

	mailboxPayload, ok := s.storage.Get(access.MymailboxUUID)
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
	recipientMailboxKey := userlib.RandomBytes(16)
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
	s.storage.Set(recipientMailboxUUID, recipientMailboxPayload)

	recipientPubkey, ok := s.keyStore.Get(recipientUsername + "_enc_pub")
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

	ciphertext, err := userlib.PKEEnc(recipientPubkey.(userlib.PKEEncKey), invBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt invitation: %w", err)
	}

	signature, err := userlib.DSSign(userdata.DSSignKey, ciphertext)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to sign invitation: %w", err)
	}

	invPayload := append(ciphertext, signature...)
	invRandomPtr := uuid.New()
	s.storage.Set(invRandomPtr, invPayload)

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
	s.storage.Set(accessUUID, newAccessPayload)

	return invRandomPtr, nil
}

// AcceptInvitation accepts a sharing invitation
func (s *InvitationService) AcceptInvitation(userdata *User, senderUsername string, invitationPtr uuid.UUID, filename string) error {
	invitationPayload, ok := s.storage.Get(invitationPtr)
	if !ok {
		return fmt.Errorf("invitation %s not found", invitationPtr.String())
	}

	if len(invitationPayload) < 256 {
		return errors.New("invitation payload too short (expected at least 256 bytes for signature)")
	}
	sigOffset := len(invitationPayload) - 256
	ciphertext := invitationPayload[:sigOffset]
	signature := invitationPayload[sigOffset:]

	senderVerifyKey, ok := s.keyStore.Get(senderUsername + "_sig_pub")
	if !ok {
		return fmt.Errorf("sender '%s' public key not found in keystore", senderUsername)
	}
	if err := userlib.DSVerify(senderVerifyKey.(userlib.DSVerifyKey), ciphertext, signature); err != nil {
		return fmt.Errorf("invitation signature verification failed for sender '%s'", senderUsername)
	}

	plaintext, err := userlib.PKEDec(userdata.PKEPrivateKey, ciphertext)
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
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	s.storage.Set(accessUUID, currAccessBytes)

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}

// RevokeAccess revokes a user's access
func (s *InvitationService) RevokeAccess(userdata *User, filename string, recipientUsername string) error {
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return fmt.Errorf("failed to generate access UUID: %w", err)
	}
	accessPayload, ok := s.storage.Get(accessUUID)
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

	mailboxPayload, ok := s.storage.Get(access.MymailboxUUID)
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

	newFileKey := userlib.RandomBytes(16)
	newFEncKey, newFMacKey := getFileKeys(newFileKey)

	inodePayload, ok := s.storage.Get(myMailbox.InodeUUID)
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
		blockPayload, ok := s.storage.Get(blockUUID)
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
		s.storage.Set(blockUUID, newBlockPayload)
	}

	newInodeBytes, err := json.Marshal(inode)
	if err != nil {
		return fmt.Errorf("failed to marshal inode: %w", err)
	}
	newInodePayload, err := encryptAndMAC(newInodeBytes, newFEncKey, newFMacKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt inode: %w", err)
	}
	s.storage.Set(myMailbox.InodeUUID, newInodePayload)

	newMailboxKey := userlib.RandomBytes(16)
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
	s.storage.Set(newMailboxUUID, newMailboxPayload)

	for childUsername, childInfo := range access.Children {
		if childUsername == recipientUsername {
			continue
		}
		childMailboxPayload, ok := s.storage.Get(childInfo.MailboxUUID)
		if !ok {
			continue
		}
		cmEncKey, cmMacKey := getMailKeys(childInfo.MailboxKey)
		childMailboxBytes, err := decryptAndVerify(childMailboxPayload, cmEncKey, cmMacKey)
		if err != nil {
			continue
		}

		var childMailbox MailboxNode
		if err := json.Unmarshal(childMailboxBytes, &childMailbox); err != nil {
			continue
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
		s.storage.Set(childInfo.MailboxUUID, updatedChildMailboxPayload)
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
	s.storage.Set(accessUUID, newAccessPayload)

	userdata.Files[filename] = accessUUID
	return saveUser(s.storage, userdata)
}
