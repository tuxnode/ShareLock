package store_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func BenchmarkStoreSet(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	data := make([]byte, 256)
	rand.Read(data)
	key := []byte("benchkey")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(key, data)
	}
}

func BenchmarkStoreGet(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	data := make([]byte, 256)
	rand.Read(data)
	key := []byte("benchkey")
	s.Set(key, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(key)
	}
}

func BenchmarkStoreSetGet(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	key := []byte("benchkey")
	data := make([]byte, 256)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(key, data)
		s.Get(key)
	}
}

func BenchmarkStoreSetParallel(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 64)
		rand.Read(buf)
		key := []byte("pk")
		for pb.Next() {
			s.Set(key, buf)
		}
	})
}

func BenchmarkStoreGetParallel(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	data := make([]byte, 64)
	rand.Read(data)
	key := []byte("pk")
	s.Set(key, data)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(key)
		}
	})
}

func BenchmarkStoreValueSize(b *testing.B) {
	for _, size := range []int{64, 1024, 65536} {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			dir, _ := os.MkdirTemp("", "bench-store-*")
			defer os.RemoveAll(dir)
			s, _ := store.Open(store.Options{Dir: dir})
			defer s.Close()

			data := make([]byte, size)
			rand.Read(data)
			key := []byte("vsize")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Set(key, data)
				s.Get(key)
			}
		})
	}
}

func BenchmarkStoreDelete(b *testing.B) {
	dir, _ := os.MkdirTemp("", "bench-store-*")
	defer os.RemoveAll(dir)
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()

	keys := make([][]byte, b.N)
	for i := range keys {
		keys[i] = []byte{byte(i)}
		s.Set(keys[i], []byte("v"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Delete(keys[i])
	}
}
