package encryption

import (
	"sync"

	"github.com/cs161-staff/project2-starter-code/internal/client/remote"
	"github.com/google/uuid"
)

type StorageService interface {
	Get(key uuid.UUID) ([]byte, bool, error)
	Set(key uuid.UUID, value []byte) error
	Delete(key uuid.UUID) error
}

type KeyStoreService interface {
	Get(key string) ([]byte, bool, error)
	Set(key string, value []byte) error
}

type MemoryStorage struct {
	mu    sync.RWMutex
	store map[uuid.UUID][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{store: make(map[uuid.UUID][]byte)}
}

func (m *MemoryStorage) Get(key uuid.UUID) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.store[key]
	return append([]byte(nil), value...), ok, nil
}

func (m *MemoryStorage) Set(key uuid.UUID, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = append([]byte(nil), value...)
	return nil
}

func (m *MemoryStorage) Delete(key uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

type MemoryKeyStore struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{store: make(map[string][]byte)}
}

func (m *MemoryKeyStore) Get(key string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.store[key]
	return append([]byte(nil), value...), ok, nil
}

func (m *MemoryKeyStore) Set(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = append([]byte(nil), value...)
	return nil
}

type RemoteStorage struct{}

func NewRemoteStorage() *RemoteStorage {
	return &RemoteStorage{}
}

func (r *RemoteStorage) Get(key uuid.UUID) ([]byte, bool, error) {
	return remote.Get(append([]byte("data:"), key[:]...))
}

func (r *RemoteStorage) Set(key uuid.UUID, value []byte) error {
	return remote.Set(append([]byte("data:"), key[:]...), value)
}

func (r *RemoteStorage) Delete(key uuid.UUID) error {
	return remote.Delete(append([]byte("data:"), key[:]...))
}

type RemoteKeyStore struct{}

func NewRemoteKeyStore() *RemoteKeyStore {
	return &RemoteKeyStore{}
}

func (r *RemoteKeyStore) Get(key string) ([]byte, bool, error) {
	return remote.Get([]byte("key:" + key))
}

func (r *RemoteKeyStore) Set(key string, value []byte) error {
	return remote.Set([]byte("key:"+key), value)
}
