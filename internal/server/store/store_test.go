package store_test

import (
	"os"
	"testing"

	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func TestSetGet(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(dir)
	s, err := store.Open(store.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Set([]byte("key1"), []byte("value1")); err != nil {
		t.Fatal(err)
	}
	val, err := s.Get([]byte("key1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "value1" {
		t.Fatalf("got %s, want value1", val)
	}
}

func TestGetNotFound(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	_, err := s.Get([]byte("nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestExists(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	s.Set([]byte("k"), []byte("v"))
	ok, err := s.Exists([]byte("k"))
	if err != nil || !ok {
		t.Fatal("expected key to exist")
	}
	ok, err = s.Exists([]byte("missing"))
	if err != nil || ok {
		t.Fatal("expected key to not exist")
	}
}

func TestDelete(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	s.Set([]byte("k"), []byte("v"))
	s.Delete([]byte("k"))
	_, err := s.Get([]byte("k"))
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestOverwrite(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	s.Set([]byte("k"), []byte("v1"))
	s.Set([]byte("k"), []byte("v2"))
	val, err := s.Get([]byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "v2" {
		t.Fatalf("got %s, want v2", val)
	}
}
