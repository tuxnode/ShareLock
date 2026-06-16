package client

// CS 161 Project 2

// Only the following imports are allowed! ANY additional imports
// may break the autograder!
// - bytes
// - encoding/hex
// - encoding/json
// - errors
// - fmt
// - github.com/cs161-staff/project2-userlib
// - github.com/google/uuid
// - strconv
// - strings

import (
	"encoding/json"

	userlib "github.com/cs161-staff/project2-userlib"
	"github.com/google/uuid"

	// hex.EncodeToString(...) is useful for converting []byte to string

	// Useful for formatting strings (e.g. `fmt.Sprintf`).
	"fmt"

	// Useful for creating new error messages to return using errors.New("...")
	"errors"

	// Optional.
	_ "strconv"
)

// This serves two purposes: it shows you a few useful primitives,
// and suppresses warnings for imports not being used. It can be
// safely deleted!
func someUsefulThings() {

	// Creates a random UUID.
	randomUUID := uuid.New()

	// Prints the UUID as a string. %v prints the value in a default format.
	// See https://pkg.go.dev/fmt#hdr-Printing for all Golang format string flags.
	userlib.DebugMsg("Random UUID: %v", randomUUID.String())

	// Creates a UUID deterministically, from a sequence of bytes.
	hash := userlib.Hash([]byte("user-structs/alice"))
	deterministicUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		// Normally, we would `return err` here. But, since this function doesn't return anything,
		// we can just panic to terminate execution. ALWAYS, ALWAYS, ALWAYS check for errors! Your
		// code should have hundreds of "if err != nil { return err }" statements by the end of this
		// project. You probably want to avoid using panic statements in your own code.
		panic(errors.New("An error occurred while generating a UUID: " + err.Error()))
	}
	userlib.DebugMsg("Deterministic UUID: %v", deterministicUUID.String())

	// Declares a Course struct type, creates an instance of it, and marshals it into JSON.
	type Course struct {
		name      string
		professor []byte
	}

	course := Course{"CS 161", []byte("Nicholas Weaver")}
	courseBytes, err := json.Marshal(course)
	if err != nil {
		panic(err)
	}

	userlib.DebugMsg("Struct: %v", course)
	userlib.DebugMsg("JSON Data: %v", courseBytes)

	// Generate a random private/public keypair.
	// The "_" indicates that we don't check for the error case here.
	var pk userlib.PKEEncKey
	var sk userlib.PKEDecKey
	pk, sk, _ = userlib.PKEKeyGen()
	userlib.DebugMsg("PKE Key Pair: (%v, %v)", pk, sk)

	// Here's an example of how to use HBKDF to generate a new key from an input key.
	// Tip: generate a new key everywhere you possibly can! It's easier to generate new keys on the fly
	// instead of trying to think about all of the ways a key reuse attack could be performed. It's also easier to
	// store one key and derive multiple keys from that one key, rather than
	originalKey := userlib.RandomBytes(16)
	derivedKey, err := userlib.HashKDF(originalKey, []byte("mac-key"))
	if err != nil {
		panic(err)
	}
	userlib.DebugMsg("Original Key: %v", originalKey)
	userlib.DebugMsg("Derived Key: %v", derivedKey)

	// A couple of tips on converting between string and []byte:
	// To convert from string to []byte, use []byte("some-string-here")
	// To convert from []byte to string for debugging, use fmt.Sprintf("hello world: %s", some_byte_arr).
	// To convert from []byte to string for use in a hashmap, use hex.EncodeToString(some_byte_arr).
	// When frequently converting between []byte and string, just marshal and unmarshal the data.
	//
	// Read more: https://go.dev/blog/strings

	// Here's an example of string interpolation!
	_ = fmt.Sprintf("%s_%d", "file", 1)
}

// This is the type definition for the User struct.
// A Go struct is like a Python or Java class - it can have attributes
// (e.g. like the Username attribute) and methods (e.g. like the StoreFile method below).

/* MasterKey should not be used to encrypt and extend File Cryptuion */
type User struct {
	Username      string
	PKEPrivateKey userlib.PrivateKeyType
	DSSignKey     userlib.DSSignKey
	MasterKey     []byte
	Files         map[string]userlib.UUID

	// You can add other attributes here if you want! But note that in order for attributes to
	// be included when this struct is serialized to/from JSON, they must be capitalized.
	// On the flipside, if you have an attribute that you want to be able to access from
	// this struct's methods, but you DON'T want that value to be included in the serialized value
	// of this struct that's stored in datastore, then you can use a "private" variable (e.g. one that
	// begins with a lowercase letter).
}

// NOTE: The following methods have toy (insecure!) implementations.

func InitUser(username string, password string) (userdataptr *User, err error) {
	hash := userlib.Hash([]byte(username + "userStruct"))
	userUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		return nil, err
	}

	if _, ok := userlib.DatastoreGet(userUUID); ok {
		return nil, errors.New("user already exists")
	}
	salt := userlib.Hash([]byte(username)) // 根据username计算唯一的salt
	masterKey := userlib.Argon2Key([]byte(password), salt, userlib.AESKeySizeBytes)

	// 密钥派生
	encKey, _ := userlib.HashKDF(masterKey, []byte("enc"))
	macKey, _ := userlib.HashKDF(masterKey, []byte("mac"))

	pkeEncKey, pkeDecKey, err := userlib.PKEKeyGen()
	if err != nil {
		return nil, err
	}

	dsSignKey, dsVerifyKey, err := userlib.DSKeyGen()
	if err != nil {
		return nil, err
	}

	userlib.KeystoreSet(username+"_enc_pub", pkeEncKey)
	userlib.KeystoreSet(username+"_sig_pub", dsVerifyKey)

	userdata := &User{
		Username:      username,
		PKEPrivateKey: pkeDecKey,
		DSSignKey:     dsSignKey,
		MasterKey:     masterKey,
		Files:         map[string]userlib.UUID{},
	}

	userBytes, _ := json.Marshal(userdata)
	payload, err := encryptAndMAC(userBytes, encKey, macKey)
	if err != nil {
		return nil, err
	}

	userlib.DatastoreSet(userUUID, payload)

	return userdata, nil
}

func GetUser(username string, password string) (userdataptr *User, err error) {
	salt := userlib.Hash([]byte(username))
	masterKey := userlib.Argon2Key([]byte(password), salt, userlib.AESKeySizeBytes)

	encKey, _ := userlib.HashKDF(masterKey, []byte("enc"))
	macKey, _ := userlib.HashKDF(masterKey, []byte("mac"))

	hash := userlib.Hash([]byte(username + "userStruct"))
	userUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		return nil, err
	}

	payload, ok := userlib.DatastoreGet(userUUID)
	if !ok {
		return nil, errors.New("Can't get User info")
	}

	plaintext, err := decryptAndVerify(payload, encKey, macKey)
	if err != nil {
		return nil, err
	}

	var userdata User
	if err := json.Unmarshal(plaintext, &userdata); err != nil {
		return nil, err
	}

	return &userdata, nil
}

type EncryptedData struct {
	Ciphertext []byte
	Hmac       []byte
}

/* inode contain an array of blockUUIDs */
type Inode struct {
	Size       int
	BlockUUIDs []userlib.UUID
}

func (userdata *User) StoreFile(filename string, content []byte) (err error) {
	if userdata.Files == nil {
		userdata.Files = make(map[string]userlib.UUID)
	}

	fileKey := userlib.RandomBytes(16)
	inodeUUID := uuid.New()

	// Init Mailbox and push to server
	mailboxnode := MailboxNode{
		FileKey:   fileKey,
		InodeUUID: inodeUUID,
	}
	mailboxBytes, err := json.Marshal(mailboxnode)
	if err != nil {
		return err
	}
	mailboxUUID := uuid.New()
	mailboxKey := userlib.RandomBytes(16)
	mEncKey, mMacKey := getMailKeys(mailboxKey)
	mailboxPayload, _ := encryptAndMAC(mailboxBytes, mEncKey, mMacKey)
	userlib.DatastoreSet(mailboxUUID, mailboxPayload)

	access := Access{
		MymailboxUUID: mailboxUUID,
		MymailboxKey:  mailboxKey,
		Chidren:       make(map[string]ChildrenInfo),
	}

	accessBytes, _ := json.Marshal(access)
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessPayload, _ := encryptAndMAC(accessBytes, pEncKey, pMacKey)

	accessUUID, _ := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	userlib.DatastoreSet(accessUUID, accessPayload)

	fEncKey, fMacKey := getFileKeys(fileKey)
	blocks := ByteToBlock(content)

	inode := Inode{
		Size:       len(content),
		BlockUUIDs: make([]userlib.UUID, 0, blocks.Len()),
	}

	// enc and store each block
	for e := blocks.Front(); e != nil; e = e.Next() {
		fb := e.Value.(*FileBlock)
		blockUUID := uuid.New() // 数据块 UUID 也是完全随机的

		blockPayload, _ := encryptAndMAC(fb.block, fEncKey, fMacKey)
		userlib.DatastoreSet(blockUUID, blockPayload)
		inode.BlockUUIDs = append(inode.BlockUUIDs, blockUUID)
	}

	// enc inode
	inodeBytes, _ := json.Marshal(inode)
	inodePayload, _ := encryptAndMAC(inodeBytes, fEncKey, fMacKey)
	userlib.DatastoreSet(inodeUUID, inodePayload)

	// update AccessUUID
	userdata.Files[filename] = accessUUID
	userdata.saveUser()

	return nil
}

func (userdata *User) AppendToFile(filename string, content []byte) error {
	if content == nil {
		return errors.New("Invalid argument")
	}

	accessUUID, _ := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	accessPayload, ok := userlib.DatastoreGet(accessUUID)
	if !ok {
		return errors.New("file not found: cannot append")
	}

	// get decrpt key
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return err
	}

	// Unmarshal accessBytes
	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return err
	}

	// Decrypt own MailboxNode to get file key and inode UUID
	mailboxPayload, ok := userlib.DatastoreGet(access.MymailboxUUID)
	if !ok {
		return errors.New("file not found: cannot append")
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

	// Get File Inode
	inodePayload, ok := userlib.DatastoreGet(myMailbox.InodeUUID)
	if !ok {
		return errors.New("file not found: cannot append")
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
		userlib.DatastoreSet(blockUUID, blockPayload)
		inode.BlockUUIDs = append(inode.BlockUUIDs, blockUUID)
	}
	inode.Size += len(content)

	newInodeBytes, _ := json.Marshal(inode)
	newInodePayload, _ := encryptAndMAC(newInodeBytes, fEncKey, fMacKey)
	userlib.DatastoreSet(myMailbox.InodeUUID, newInodePayload)

	return nil
}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	accessUUID, _ := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	accessPayload, ok := userlib.DatastoreGet(accessUUID)
	if !ok {
		return nil, errors.New("file not found: access struct missing")
	}

	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return nil, err
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return nil, err
	}

	// Decrypt own MailboxNode to get file key and inode UUID
	mailboxPayload, ok := userlib.DatastoreGet(access.MymailboxUUID)
	if !ok {
		return nil, errors.New("file structure corrupted: mailbox missing")
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return nil, err
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return nil, err
	}

	// decrypt inode
	inodePayload, ok := userlib.DatastoreGet(myMailbox.InodeUUID)
	if !ok {
		return nil, errors.New("file structure corrupted: inode missing")
	}
	fEncKey, fMacKey := getFileKeys(myMailbox.FileKey)
	inodeBytes, err := decryptAndVerify(inodePayload, fEncKey, fMacKey)
	if err != nil {
		return nil, err
	}

	var inode Inode
	if err := json.Unmarshal(inodeBytes, &inode); err != nil {
		return nil, err
	}

	// decrypt Data
	var data []byte
	for _, blockUUID := range inode.BlockUUIDs {
		blockPayload, ok := userlib.DatastoreGet(blockUUID)
		if !ok {
			return nil, errors.New("Can't get data by this UUID")
		}

		blockPlaintext, err := decryptAndVerify(blockPayload, fEncKey, fMacKey)
		if err != nil {
			return nil, err
		}
		data = append(data, blockPlaintext...)
	}

	return data, nil
}

func (userdata *User) CreateInvitation(filename string, recipientUsername string) (
	invitationPtr uuid.UUID, err error) {

	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return uuid.Nil, err
	}

	accessPayload, ok := userlib.DatastoreGet(accessUUID)
	if !ok {
		return uuid.Nil, errors.New("file not found: cannot create invitation")
	}
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return uuid.Nil, err
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return uuid.Nil, err
	}

	// Decrypt own MailboxNode to get file key and inode UUID
	mailboxPayload, ok := userlib.DatastoreGet(access.MymailboxUUID)
	if !ok {
		return uuid.Nil, errors.New("file not found: mailbox missing")
	}
	mEncKey, mMacKey := getMailKeys(access.MymailboxKey)
	mailboxBytes, err := decryptAndVerify(mailboxPayload, mEncKey, mMacKey)
	if err != nil {
		return uuid.Nil, err
	}

	var myMailbox MailboxNode
	if err := json.Unmarshal(mailboxBytes, &myMailbox); err != nil {
		return uuid.Nil, err
	}

	// Create recipient's MailboxNode
	recipientMailboxUUID := uuid.New()
	recipientMailboxKey := userlib.RandomBytes(16)
	recipientMailbox := MailboxNode{
		FileKey:   myMailbox.FileKey,
		InodeUUID: myMailbox.InodeUUID,
	}
	recipientMailboxBytes, err := json.Marshal(recipientMailbox)
	if err != nil {
		return uuid.Nil, err
	}
	rmEncKey, rmMacKey := getMailKeys(recipientMailboxKey)
	recipientMailboxPayload, err := encryptAndMAC(recipientMailboxBytes, rmEncKey, rmMacKey)
	if err != nil {
		return uuid.Nil, err
	}
	userlib.DatastoreSet(recipientMailboxUUID, recipientMailboxPayload)

	// Get recipient's Publioc Key From KeyStore
	recipientPubkey, ok := userlib.KeystoreGet(recipientUsername + "_enc_pub")
	if !ok {
		return uuid.Nil, errors.New("recipient user does not exist")
	}

	invitation := Invitation{
		MailboxUUID: recipientMailboxUUID,
		MailboxKey:  recipientMailboxKey,
	}
	invBytes, err := json.Marshal(invitation)
	if err != nil {
		return uuid.Nil, err
	}

	// encrypt invitation by recipientPubKey
	ciphertext, err := userlib.PKEEnc(recipientPubkey, invBytes)
	if err != nil {
		return uuid.Nil, err
	}

	// Sign the invitation by DssignKey
	signature, err := userlib.DSSign(userdata.DSSignKey, ciphertext)
	if err != nil {
		return uuid.Nil, err
	}

	invPayload := append(ciphertext, signature...)

	// Push the Invitation to remote server
	invRandomPtr := uuid.New()
	userlib.DatastoreSet(invRandomPtr, invPayload)

	// Update sender's Access Chidren
	access.Chidren[recipientUsername] = ChildrenInfo{
		MailboxUUID: recipientMailboxUUID,
		MailboxKey:  recipientMailboxKey,
	}
	newAccessBytes, _ := json.Marshal(access)
	newAccessPayload, _ := encryptAndMAC(newAccessBytes, pEncKey, pMacKey)
	userlib.DatastoreSet(accessUUID, newAccessPayload)

	return invRandomPtr, nil
}

func (userdata *User) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	// Get invitation from remote server
	invitationPayload, ok := userlib.DatastoreGet(invitationPtr)
	if !ok {
		return errors.New("Can't get invitation by this ptr")
	}

	if len(invitationPayload) < 256 {
		return errors.New("invitation payload too short to contain a signature")
	}
	sigOffset := len(invitationPayload) - 256
	ciphertext := invitationPayload[:sigOffset]
	signature := invitationPayload[sigOffset:]

	// get sender public key from KeyStore server
	senderVerifyKey, ok := userlib.KeystoreGet(senderUsername + "_sig_pub")
	if !ok {
		return errors.New("sender public key not found in Keystore")
	}
	// DS check
	if err := userlib.DSVerify(senderVerifyKey, ciphertext, signature); err != nil {
		return errors.New("cryptographic doom: signature verification failed")
	}

	plaintext, err := userlib.PKEDec(userdata.PKEPrivateKey, ciphertext)
	if err != nil {
		return err
	}

	var getInvitation Invitation
	if err := json.Unmarshal(plaintext, &getInvitation); err != nil {
		return err
	}

	// Store this Transfrom Access data and push to remote server
	receiveAccess := Access{
		MymailboxUUID: getInvitation.MailboxUUID,
		MymailboxKey:  getInvitation.MailboxKey,
		Chidren:       make(map[string]ChildrenInfo),
	}
	currAccessPayload, err := json.Marshal(receiveAccess)
	if err != nil {
		return err
	}
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	currAccessBytes, err := encryptAndMAC(currAccessPayload, pEncKey, pMacKey)
	if err != nil {
		return err
	}
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return err
	}
	userlib.DatastoreSet(accessUUID, currAccessBytes)

	// update local User status
	userdata.Files[filename] = accessUUID
	userdata.saveUser()
	return nil
}

/*
RevokeAccess revokes a recipient's access to a file by rotating the file key.
流程: 1)生成新FileKey 2)重新加密所有block和inode

	3)创建owner的新MailboxNode 4)更新其他children的MailboxNode为新FileKey
	5)从Chidren移除被撤销用户 6)持久化Access
*/
func (userdata *User) RevokeAccess(filename string, recipientUsername string) error {
	// Get Access struct
	accessUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename))[:16])
	if err != nil {
		return err
	}
	accessPayload, ok := userlib.DatastoreGet(accessUUID)
	if !ok {
		return errors.New("file not found: cannot revoke")
	}
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessBytes, err := decryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return err
	}

	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return err
	}

	// Decrypt own MailboxNode to get old file key and inode UUID
	mailboxPayload, ok := userlib.DatastoreGet(access.MymailboxUUID)
	if !ok {
		return errors.New("file not found: mailbox missing")
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

	// Generate new file key
	newFileKey := userlib.RandomBytes(16)
	newFEncKey, newFMacKey := getFileKeys(newFileKey)

	// Re-encrypt all blocks and inode with new file key
	inodePayload, ok := userlib.DatastoreGet(myMailbox.InodeUUID)
	if !ok {
		return errors.New("file structure corrupted: inode missing")
	}
	oldFEncKey, oldFMacKey := getFileKeys(myMailbox.FileKey)
	inodeBytes, err := decryptAndVerify(inodePayload, oldFEncKey, oldFMacKey)
	if err != nil {
		return err
	}

	var inode Inode
	if err := json.Unmarshal(inodeBytes, &inode); err != nil {
		return err
	}

	for _, blockUUID := range inode.BlockUUIDs {
		blockPayload, ok := userlib.DatastoreGet(blockUUID)
		if !ok {
			return errors.New("Can't get data by this UUID")
		}
		blockPlaintext, err := decryptAndVerify(blockPayload, oldFEncKey, oldFMacKey)
		if err != nil {
			return err
		}
		newBlockPayload, err := encryptAndMAC(blockPlaintext, newFEncKey, newFMacKey)
		if err != nil {
			return err
		}
		userlib.DatastoreSet(blockUUID, newBlockPayload)
	}

	newInodeBytes, _ := json.Marshal(inode)
	newInodePayload, err := encryptAndMAC(newInodeBytes, newFEncKey, newFMacKey)
	if err != nil {
		return err
	}
	userlib.DatastoreSet(myMailbox.InodeUUID, newInodePayload)

	// Create new MailboxNode for owner with new file key
	newMailboxKey := userlib.RandomBytes(16)
	newMailboxUUID := uuid.New()
	newMailbox := MailboxNode{
		FileKey:   newFileKey,
		InodeUUID: myMailbox.InodeUUID,
	}
	newMailboxBytes, err := json.Marshal(newMailbox)
	if err != nil {
		return err
	}
	nmEncKey, nmMacKey := getMailKeys(newMailboxKey)
	newMailboxPayload, err := encryptAndMAC(newMailboxBytes, nmEncKey, nmMacKey)
	if err != nil {
		return err
	}
	userlib.DatastoreSet(newMailboxUUID, newMailboxPayload)

	// Update remaining children's MailboxNodes with new file key
	for childUsername, childInfo := range access.Chidren {
		if childUsername == recipientUsername {
			continue
		}
		childMailboxPayload, ok := userlib.DatastoreGet(childInfo.MailboxUUID)
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
		updatedChildMailboxBytes, _ := json.Marshal(childMailbox)
		updatedChildMailboxPayload, err := encryptAndMAC(updatedChildMailboxBytes, cmEncKey, cmMacKey)
		if err != nil {
			return err
		}
		userlib.DatastoreSet(childInfo.MailboxUUID, updatedChildMailboxPayload)
	}

	// Remove revoked user from Chidren
	delete(access.Chidren, recipientUsername)

	// Update owner's Access with new mailbox info
	access.MymailboxUUID = newMailboxUUID
	access.MymailboxKey = newMailboxKey

	newAccessBytes, _ := json.Marshal(access)
	newAccessPayload, _ := encryptAndMAC(newAccessBytes, pEncKey, pMacKey)
	userlib.DatastoreSet(accessUUID, newAccessPayload)

	// update local User status
	userdata.Files[filename] = accessUUID
	userdata.saveUser()

	return nil
}
