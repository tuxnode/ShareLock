package userlib

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type UUID = uuid.UUID

const AESBlockSizeBytes = aes.BlockSize
const AESKeySizeBytes = 16
const HashSizeBytes = sha512.Size

const rsaKeySizeBits = 2048
const UUIDSizeBytes = 16

// Network protocol ops
const (
	opGet    = byte(0x01)
	opSet    = byte(0x02)
	opDelete = byte(0x03)
)

const (
	statusOK       = byte(0x00)
	statusNotFound = byte(0x01)
)

type PublicKeyType struct {
	KeyType string
	PubKey  rsa.PublicKey
}

type PrivateKeyType struct {
	KeyType string
	PrivKey rsa.PrivateKey
}

type DatastoreEntry struct {
	UUID  string
	Value string
}

// Network connection state
var (
	remoteAddr string
	useTLS     bool
)

func Connect(address string, tlsEnabled bool) {
	remoteAddr = address
	useTLS = tlsEnabled
}

func Disconnect() {
	remoteAddr = ""
}

func isConnected() bool {
	return remoteAddr != ""
}

func dial() (net.Conn, error) {
	if useTLS {
		return tls.Dial("tcp", remoteAddr, &tls.Config{InsecureSkipVerify: true})
	}
	return net.Dial("tcp", remoteAddr)
}

// In-memory fallback (used when not connected, e.g., in tests)
var datastoreBandwidth sync.Map
var datastore sync.Map
var keystore sync.Map

type keystoreType map[string]PublicKeyType
type datastoreType map[UUID][]byte

func getKeystoreShard() keystoreType {
	pid := CurrentSpecReport().LineNumber()
	shard, _ := keystore.LoadOrStore(pid, make(keystoreType))
	return shard.(keystoreType)
}

func getDatastoreShard() datastoreType {
	pid := CurrentSpecReport().LineNumber()
	shard, _ := datastore.LoadOrStore(pid, make(datastoreType))
	return shard.(datastoreType)
}

func getDatastoreBandwidthShard() *int {
	pid := CurrentSpecReport().LineNumber()
	newBandwidth := 0
	bandwidth, _ := datastoreBandwidth.LoadOrStore(pid, &newBandwidth)
	return bandwidth.(*int)
}

// Sets the value in the datastore
func datastoreSet(key UUID, value []byte) {
	if isConnected() {
		conn, err := dial()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		writeOp(conn, opSet, key[:], value)
		readStatus(conn)
		return
	}
	bandwidth := getDatastoreBandwidthShard()
	*bandwidth += len(value)
	foo := make([]byte, len(value))
	copy(foo, value)
	datastoreShard := getDatastoreShard()
	datastoreShard[key] = foo
}

var DatastoreSet = datastoreSet

func datastoreGet(key UUID) (value []byte, ok bool) {
	if isConnected() {
		conn, err := dial()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		writeOp(conn, opGet, key[:], nil)
		return readValue(conn)
	}
	datastoreShard := getDatastoreShard()
	value, ok = datastoreShard[key]
	if ok && value != nil {
		bandwidth := getDatastoreBandwidthShard()
		*bandwidth += len(value)
		foo := make([]byte, len(value))
		copy(foo, value)
		return foo, ok
	}
	return
}

var DatastoreGet = datastoreGet

func datastoreDelete(key UUID) {
	if isConnected() {
		conn, err := dial()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		writeOp(conn, opDelete, key[:], nil)
		readStatus(conn)
		return
	}
	datastoreShard := getDatastoreShard()
	delete(datastoreShard, key)
}

var DatastoreDelete = datastoreDelete

func datastoreClear() {
	if isConnected() {
		return
	}
	datastoreShard := getDatastoreShard()
	for k := range datastoreShard {
		delete(datastoreShard, k)
	}
}

var DatastoreClear = datastoreClear

func DatastoreResetBandwidth() {
	bandwidth := getDatastoreBandwidthShard()
	*bandwidth = 0
}

func DatastoreGetBandwidth() int {
	bandwidth := getDatastoreBandwidthShard()
	return *bandwidth
}

func keystoreClear() {
	if isConnected() {
		return
	}
	keystoreShard := getKeystoreShard()
	for k := range keystoreShard {
		delete(keystoreShard, k)
	}
}

var KeystoreClear = keystoreClear

func keystoreSet(key string, value PublicKeyType) error {
	if isConnected() {
		conn, err := dial()
		if err != nil {
			return err
		}
		defer conn.Close()
		data, _ := json.Marshal(value)
		writeOp(conn, opSet, []byte(key), data)
		st := readStatus(conn)
		if st == statusOK {
			return nil
		}
		return errors.New("keystore set failed")
	}
	keystoreShard := getKeystoreShard()
	_, present := keystoreShard[key]
	if present {
		return errors.New("entry in keystore has been taken")
	}
	keystoreShard[key] = value
	return nil
}

var KeystoreSet = keystoreSet

func keystoreGet(key string) (value PublicKeyType, ok bool) {
	if isConnected() {
		conn, err := dial()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		writeOp(conn, opGet, []byte(key), nil)
		data, found := readValue(conn)
		if !found {
			return PublicKeyType{}, false
		}
		var v PublicKeyType
		json.Unmarshal(data, &v)
		return v, true
	}
	keystoreShard := getKeystoreShard()
	value, ok = keystoreShard[key]
	return
}

var KeystoreGet = keystoreGet

func DatastoreGetMap() map[UUID][]byte {
	datastoreShard := getDatastoreShard()
	return datastoreShard
}

func KeystoreGetMap() map[string]PublicKeyType {
	keystoreShard := getKeystoreShard()
	return keystoreShard
}

// Network protocol helpers
func writeOp(conn io.Writer, op byte, key, val []byte) {
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	conn.Write([]byte{op})
	conn.Write(kl)
	conn.Write(key)
	if op == opSet {
		vl := make([]byte, 4)
		binary.BigEndian.PutUint32(vl, uint32(len(val)))
		conn.Write(vl)
		conn.Write(val)
	}
}

func readStatus(conn io.Reader) byte {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		panic(err)
	}
	return buf[0]
}

func readValue(conn io.Reader) ([]byte, bool) {
	status := readStatus(conn)
	if status == statusNotFound {
		return nil, false
	}
	vl := make([]byte, 4)
	if _, err := io.ReadFull(conn, vl); err != nil {
		panic(err)
	}
	l := binary.BigEndian.Uint32(vl)
	val := make([]byte, l)
	if _, err := io.ReadFull(conn, val); err != nil {
		panic(err)
	}
	return val, true
}

// RandomBytes
func randomBytes(size int) (data []byte) {
	data = make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}
	return
}

var RandomBytes = randomBytes

// Argon2Key
func argon2Key(password []byte, salt []byte, keyLen uint32) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, keyLen)
}

var Argon2Key = argon2Key

// SHA512
func hash(data []byte) []byte {
	hashVal := sha512.Sum512(data)
	return hashVal[:]
}

var Hash = hash

// PKE
type PKEEncKey = PublicKeyType
type PKEDecKey = PrivateKeyType
type DSSignKey = PrivateKeyType
type DSVerifyKey = PublicKeyType

func pkeKeyGen() (PKEEncKey, PKEDecKey, error) {
	RSAPrivKey, err := rsa.GenerateKey(rand.Reader, rsaKeySizeBits)
	RSAPubKey := RSAPrivKey.PublicKey
	return PKEEncKey{"PKE", RSAPubKey}, PKEDecKey{"PKE", *RSAPrivKey}, err
}

var PKEKeyGen = pkeKeyGen

func pkeEnc(ek PKEEncKey, plaintext []byte) ([]byte, error) {
	if ek.KeyType != "PKE" {
		return nil, errors.New("using a non-pke key for pke")
	}
	return rsa.EncryptOAEP(sha512.New(), rand.Reader, &ek.PubKey, plaintext, nil)
}

var PKEEnc = pkeEnc

func pkeDec(dk PKEDecKey, ciphertext []byte) ([]byte, error) {
	if dk.KeyType != "PKE" {
		return nil, errors.New("using a non-pke for pke")
	}
	return rsa.DecryptOAEP(sha512.New(), rand.Reader, &dk.PrivKey, ciphertext, nil)
}

var PKEDec = pkeDec

// DS
func dsKeyGen() (DSSignKey, DSVerifyKey, error) {
	RSAPrivKey, err := rsa.GenerateKey(rand.Reader, rsaKeySizeBits)
	RSAPubKey := RSAPrivKey.PublicKey
	return DSSignKey{"DS", *RSAPrivKey}, DSVerifyKey{"DS", RSAPubKey}, err
}

var DSKeyGen = dsKeyGen

func dsSign(sk DSSignKey, msg []byte) ([]byte, error) {
	if sk.KeyType != "DS" {
		return nil, errors.New("using a non-ds key for ds")
	}
	hashed := sha512.Sum512(msg)
	return rsa.SignPKCS1v15(rand.Reader, &sk.PrivKey, crypto.SHA512, hashed[:])
}

var DSSign = dsSign

func dsVerify(vk DSVerifyKey, msg []byte, sig []byte) error {
	if vk.KeyType != "DS" {
		return errors.New("using a non-ds key for ds")
	}
	hashed := sha512.Sum512(msg)
	return rsa.VerifyPKCS1v15(&vk.PubKey, crypto.SHA512, hashed[:], sig)
}

var DSVerify = dsVerify

// HMAC
func hmacEval(key []byte, msg []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, errors.New("input as key for hmac should be a 16-byte key")
	}
	mac := hmac.New(sha512.New, key)
	mac.Write(msg)
	return mac.Sum(nil), nil
}

var HMACEval = hmacEval

func hmacEqual(a []byte, b []byte) bool {
	return hmac.Equal(a, b)
}

var HMACEqual = hmacEqual

// HashKDF
func hashKDF(key []byte, msg []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, errors.New("input as key for HashKDF should be a 16-byte key")
	}
	mac := hmac.New(sha512.New, key)
	mac.Write(msg)
	return mac.Sum(nil), nil
}

var HashKDF = hashKDF

// SymEnc / SymDec
func symEnc(key []byte, iv []byte, plaintext []byte) []byte {
	if len(iv) != AESBlockSizeBytes {
		panic("IV length not equal to AESBlockSizeBytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext := make([]byte, AESBlockSizeBytes+len(plaintext))
	mode := cipher.NewCTR(block, iv)
	mode.XORKeyStream(ciphertext[AESBlockSizeBytes:], plaintext)
	copy(ciphertext[:AESBlockSizeBytes], iv)
	return ciphertext
}

var SymEnc = symEnc

func symDec(key []byte, ciphertext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < AESBlockSizeBytes {
		panic("ciphertext too short")
	}
	iv := ciphertext[:AESBlockSizeBytes]
	ciphertext = ciphertext[AESBlockSizeBytes:]
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCTR(block, iv)
	mode.XORKeyStream(plaintext, ciphertext)
	return plaintext
}

var SymDec = symDec

var DebugOutput = true

func DebugMsg(format string, args ...interface{}) {
	if DebugOutput {
		msg := fmt.Sprintf("%v ", time.Now().Format("15:04:05.00000"))
		log.Printf(msg+strings.Trim(format, "\r\n ")+"\n", args...)
	}
}

func MapKeyFromBytes(data []byte) (truncated string) {
	return fmt.Sprintf("%x", sha512.Sum512(data))
}
