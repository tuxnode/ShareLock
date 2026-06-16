package store

import (
	"time"

	"github.com/dgraph-io/badger/v3"
)

type Store struct {
	db *badger.DB
}

type Options struct {
	Dir string
}

func Open(opts Options) (*Store, error) {
	db, err := badger.Open(badger.DefaultOptions(opts.Dir).
		WithLogger(nil))
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Get(key []byte) (value []byte, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

func (s *Store) Set(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *Store) SetWithTTL(key, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry(key, value).WithTTL(ttl))
	})
}

func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *Store) Exists(key []byte) (bool, error) {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
