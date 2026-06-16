package handler_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/cs161-staff/project2-starter-code/internal/server/handler"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func benchmarkHandlerOp(b *testing.B, op byte, key, val []byte) {
	dir := b.TempDir()
	s, _ := store.Open(store.Options{Dir: dir})
	defer s.Close()
	h := handler.New(s)

	var buf bytes.Buffer
	writeBenchOp(&buf, op, key, val)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		writeBenchOp(&buf, op, key, val)
		h.Handle(&buf)
		buf.ReadByte()
		if op == handler.OpGet {
			vl := make([]byte, 4)
			buf.Read(vl)
			l := binary.BigEndian.Uint32(vl)
			val := make([]byte, l)
			buf.Read(val)
		}
	}
}

func writeBenchOp(buf *bytes.Buffer, op byte, key, val []byte) {
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	buf.WriteByte(op)
	buf.Write(kl)
	buf.Write(key)
	if op == handler.OpSet {
		vl := make([]byte, 4)
		binary.BigEndian.PutUint32(vl, uint32(len(val)))
		buf.Write(vl)
		buf.Write(val)
	}
}

func BenchmarkHandlerSet(b *testing.B) {
	benchmarkHandlerOp(b, handler.OpSet, []byte("key"), make([]byte, 256))
}

func BenchmarkHandlerGet(b *testing.B) {
	benchmarkHandlerOp(b, handler.OpGet, []byte("key"), nil)
}

func BenchmarkHandlerDelete(b *testing.B) {
	benchmarkHandlerOp(b, handler.OpDelete, []byte("key"), nil)
}

func BenchmarkHandlerSetKeySize(b *testing.B) {
	for _, size := range []int{8, 64, 256, 1024} {
		b.Run(fmt.Sprintf("key-%d", size), func(b *testing.B) {
			key := make([]byte, size)
			benchmarkHandlerOp(b, handler.OpSet, key, make([]byte, 256))
		})
	}
}

func BenchmarkHandlerSetValueSize(b *testing.B) {
	for _, size := range []int{64, 1024, 65536} {
		b.Run(fmt.Sprintf("val-%d", size), func(b *testing.B) {
			benchmarkHandlerOp(b, handler.OpSet, []byte("key"), make([]byte, size))
		})
	}
}
