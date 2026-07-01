package encryption

import (
	"sync"

	userlib "github.com/cs161-staff/project2-starter-code/internal/userlib"
)

// MemoryStorage is an in-memory implementation of StorageService for testing
type MemoryStorage struct {
	mu    sync.RWMutex
	store map[userlib.UUID][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		store: make(map[userlib.UUID][]byte),
	}
}

func (m *MemoryStorage) Get(key userlib.UUID) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.store[key]
	return val, ok
}

func (m *MemoryStorage) Set(key userlib.UUID, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
}

func (m *MemoryStorage) Delete(key userlib.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
}

// MemoryKeyStore is an in-memory implementation of KeyStoreService for testing
type MemoryKeyStore struct {
	mu    sync.RWMutex
	store map[string]interface{}
}

func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{
		store: make(map[string]interface{}),
	}
}

func (m *MemoryKeyStore) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.store[key]
	return val, ok
}

func (m *MemoryKeyStore) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
}

// UserlibStorage adapts userlib.DatastoreGet/Set to StorageService interface
type UserlibStorage struct{}

func NewUserlibStorage() *UserlibStorage {
	return &UserlibStorage{}
}

func (u *UserlibStorage) Get(key userlib.UUID) ([]byte, bool) {
	return userlib.DatastoreGet(key)
}

func (u *UserlibStorage) Set(key userlib.UUID, value []byte) {
	userlib.DatastoreSet(key, value)
}

func (u *UserlibStorage) Delete(key userlib.UUID) {
	userlib.DatastoreDelete(key)
}

// UserlibKeyStore adapts userlib.KeystoreGet/Set to KeyStoreService interface.
// Each instance maintains a local cache and falls through to the global
// userlib keystore (which shards by Ginkgo spec line). This prevents
// cross-test key contamination when multiple tests call InitUser for
// the same username without clearing the global keystore.
type UserlibKeyStore struct {
	mu    sync.RWMutex
	local map[string]userlib.PublicKeyType
}

func NewUserlibKeyStore() *UserlibKeyStore {
	return &UserlibKeyStore{
		local: make(map[string]userlib.PublicKeyType),
	}
}

func (u *UserlibKeyStore) Get(key string) (interface{}, bool) {
	u.mu.RLock()
	if v, ok := u.local[key]; ok {
		u.mu.RUnlock()
		return v, true
	}
	u.mu.RUnlock()
	return userlib.KeystoreGet(key)
}

func (u *UserlibKeyStore) Set(key string, value interface{}) {
	if pk, ok := value.(userlib.PublicKeyType); ok {
		u.mu.Lock()
		u.local[key] = pk
		u.mu.Unlock()
		userlib.KeystoreSet(key, pk)
	}
}
