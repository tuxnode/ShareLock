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

	// Useful for string manipulation
	"strings"

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
type User struct {
	Username      string
	PKEPrivateKey userlib.PrivateKeyType
	DSSignKey     userlib.DSSignKey
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

/* 加密数据并打包MAC封条的过程 */
func encryptAndMAC(data []byte, encKey []byte, macKey []byte) (payload []byte, err error) {
	if len(encKey) != 16 {
		return nil, errors.New("encryption key must be exactly 16 bytes")
	}

	// 生成随机向量
	iv := userlib.RandomBytes(16)
	// 对称加密
	ciphertext := userlib.SymEnc(encKey, iv, data)
	mac, err := userlib.HMACEval(macKey, ciphertext)
	if err != nil {
		return nil, err
	}

	// 构造密文+封条: iv ciphertest mac
	payload = append(ciphertext, mac...)
	return payload, nil
}

/* 对应验证MAC和解密 */
func decaryptAndVerify(payload []byte, encKey []byte, macKey []byte) (plaintext []byte, err error) {
	const ivLen = 16
	const macLen = 64 // SHA512

	if len(payload) < ivLen+macLen {
		return nil, errors.New("malformed payload: data stream too short")
	}

	// splite package
	macOffset := len(payload) - macLen
	receiveMac := payload[macLen:]
	ciphertext := payload[ivLen:macOffset]

	macInput := payload[macOffset:]
	expectMac, err := userlib.HMACEval(macKey, macInput)
	if err != nil {
		return nil, err
	}

	if !userlib.HMACEqual(receiveMac, expectMac) {
		return nil, errors.New("cryptographic doom: MAC verification failed, data tampered")
	}

	plaintext = userlib.SymDec(encKey, ciphertext)

	return plaintext, nil
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

func (userdata *User) StoreFile(filename string, content []byte) (err error) {
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + userdata.Username))[:16])
	if err != nil {
		return err
	}
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return err
	}
	userlib.DatastoreSet(storageKey, contentBytes)
	return
}

func (userdata *User) AppendToFile(filename string, content []byte) error {
	return nil
}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + userdata.Username))[:16])
	if err != nil {
		return nil, err
	}
	dataJSON, ok := userlib.DatastoreGet(storageKey)
	if !ok {
		return nil, errors.New(strings.ToTitle("file not found"))
	}
	err = json.Unmarshal(dataJSON, &content)
	return content, err
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
