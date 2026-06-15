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

	plaintext, err := decaryptAndVerify(payload, encKey, macKey)
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

	access := Access{
		FileKey:   fileKey,
		InodeUUID: inodeUUID,
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

		blockPayload, _ := encryptAndMAC(fb.block[:], fEncKey, fMacKey)
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

	accessUUID, _ := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + filename)))
	accessPayload, ok := userlib.DatastoreGet(accessUUID)
	if !ok {
		return errors.New("file not found: cannot append")
	}

	// get decrpt key
	pEncKey, pMacKey := userdata.getPersonalKey(filename)
	accessBytes, err := decaryptAndVerify(accessPayload, pEncKey, pMacKey)
	if err != nil {
		return err
	}

	// Unmarshal accessBytes
	var access Access
	if err := json.Unmarshal(accessBytes, &access); err != nil {
		return err
	}

	// Get File Inode
	inodePayload, ok := userlib.DatastoreGet(access.InodeUUID)
	if !ok {
		return errors.New("file not found: cannot append")
	}

	fEncKey, fMacKey := getFileKeys(access.FileKey)
	inodeBytes, err := decaryptAndVerify(inodePayload, fEncKey, fMacKey)
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

		blockPayload, err := encryptAndMAC(fb.block[:], fEncKey, fMacKey)
		if err != nil {
			return err
		}
		userlib.DatastoreSet(blockUUID, blockPayload)
		inode.BlockUUIDs = append(inode.BlockUUIDs, blockUUID)
	}
	inode.Size += len(content)

	newInodeBytes, _ := json.Marshal(inode)
	newInodePayload, _ := encryptAndMAC(newInodeBytes, fEncKey, fMacKey)
	userlib.DatastoreSet(access.InodeUUID, newInodePayload)

	return nil
}

func (userdata *User) getFileKeys(key []byte) (any, any) {
	panic("unimplemented")
}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	salt := userlib.Hash([]byte(filename))
	encKey, macKey := userdata.getUserKey(salt)

	inodeUUID := userlib.UUID(userlib.Hash([]byte(filename + userdata.Username))[:16])
	value, ok := userlib.DatastoreGet(inodeUUID)
	if !ok {
		return nil, errors.New("Can't get file by this inodeUUID")
	}

	var ed EncryptedData
	if err := json.Unmarshal(value, &ed); err != nil {
		return nil, err
	}

	// 解密验证inode
	plaintext, err := decaryptAndVerify(append(ed.Ciphertext, ed.Hmac...), encKey, macKey)
	if err != nil {
		return nil, err
	}

	// 读取Inode id list
	var inode Inode
	if err := json.Unmarshal(plaintext, &inode); err != nil {
		return nil, err
	}

	var data []byte
	for _, blockUUID := range inode.BlockUUIDs {
		value, ok := userlib.DatastoreGet(blockUUID)
		if !ok {
			return nil, errors.New("Can't get File by UUID")
		}

		var blockData EncryptedData
		if err := json.Unmarshal(value, &blockData); err != nil {
			return nil, err
		}
		blockPlaintext, err := decaryptAndVerify(append(blockData.Ciphertext, blockData.Hmac...), encKey, macKey)
		if err != nil {
			return nil, err
		}
		data = append(data, blockPlaintext...)
	}
	return data[:inode.Size], nil
}
