package handler_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/cs161-staff/project2-starter-code/internal/server/handler"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func TestSetAndGet(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer

	// SET
	writeOp(&buf, handler.OpSet, []byte("mykey"), []byte("myvalue"))
	if err := h.Handle(&buf); err != nil {
		t.Fatal(err)
	}
	status, _ := buf.ReadByte()
	if status != handler.StatusOK {
		t.Fatalf("SET failed: status %d", status)
	}

	// GET
	writeOp(&buf, handler.OpGet, []byte("mykey"), nil)
	if err := h.Handle(&buf); err != nil {
		t.Fatal(err)
	}
	val := readValue(t, &buf)
	if string(val) != "myvalue" {
		t.Fatalf("got %s, want myvalue", val)
	}
}

func TestGetNotFound(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer
	writeOp(&buf, handler.OpGet, []byte("nokey"), nil)
	if err := h.Handle(&buf); err != nil {
		t.Fatal(err)
	}
	status, _ := buf.ReadByte()
	if status != handler.StatusNotFound {
		t.Fatalf("expected NOT_FOUND, got %d", status)
	}
}

func TestDelete(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer

	// SET
	writeOp(&buf, handler.OpSet, []byte("k"), []byte("v"))
	h.Handle(&buf)
	buf.ReadByte()

	// DELETE
	writeOp(&buf, handler.OpDelete, []byte("k"), nil)
	h.Handle(&buf)
	status, _ := buf.ReadByte()
	if status != handler.StatusOK {
		t.Fatalf("DELETE failed: status %d", status)
	}

	// GET after delete → NOT_FOUND
	writeOp(&buf, handler.OpGet, []byte("k"), nil)
	h.Handle(&buf)
	status, _ = buf.ReadByte()
	if status != handler.StatusNotFound {
		t.Fatalf("expected NOT_FOUND after delete, got %d", status)
	}
}

func TestEmptyValue(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer

	writeOp(&buf, handler.OpSet, []byte("empty"), []byte{})
	h.Handle(&buf)
	buf.ReadByte()

	writeOp(&buf, handler.OpGet, []byte("empty"), nil)
	h.Handle(&buf)
	val := readValue(t, &buf)
	if len(val) != 0 {
		t.Fatalf("expected empty value, got %d bytes", len(val))
	}
}

func TestLargeValue(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer

	large := make([]byte, 64<<10)
	for i := range large {
		large[i] = byte(i % 256)
	}
	writeOp(&buf, handler.OpSet, []byte("large"), large)
	h.Handle(&buf)
	buf.ReadByte()

	writeOp(&buf, handler.OpGet, []byte("large"), nil)
	h.Handle(&buf)
	val := readValue(t, &buf)
	if len(val) != len(large) {
		t.Fatalf("length mismatch: %d vs %d", len(val), len(large))
	}
}

func TestUnknownOp(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0, 0, 0, 1, 'x'})
	if err := h.Handle(&buf); err != nil {
		t.Fatal(err)
	}
	status, _ := buf.ReadByte()
	if status != handler.StatusError {
		t.Fatalf("expected ERROR for unknown op, got %d", status)
	}
}

func TestOrderedOps(t *testing.T) {
	s, _ := store.Open(store.Options{Dir: t.TempDir()})
	t.Cleanup(func() { s.Close() })
	h := handler.New(s)

	var buf bytes.Buffer
	for i := 0; i < 50; i++ {
		buf.Reset()
		writeOp(&buf, handler.OpSet, []byte{byte(i)}, []byte{byte(i + 1)})
		if err := h.Handle(&buf); err != nil {
			t.Fatal(err)
		}
		status, _ := buf.ReadByte()
		if status != handler.StatusOK {
			t.Fatalf("SET %d failed: status %d", i, status)
		}
	}
	for i := 0; i < 50; i++ {
		buf.Reset()
		writeOp(&buf, handler.OpGet, []byte{byte(i)}, nil)
		if err := h.Handle(&buf); err != nil {
			t.Fatal(err)
		}
		val := readValue(t, &buf)
		if len(val) != 1 || val[0] != byte(i+1) {
			t.Fatalf("GET %d failed", i)
		}
	}
}

func writeOp(buf *bytes.Buffer, op byte, key, val []byte) {
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

func readValue(t *testing.T, buf *bytes.Buffer) []byte {
	t.Helper()
	status, _ := buf.ReadByte()
	if status != handler.StatusOK {
		t.Fatalf("expected OK status, got %d", status)
	}
	vl := make([]byte, 4)
	buf.Read(vl)
	l := binary.BigEndian.Uint32(vl)
	val := make([]byte, l)
	buf.Read(val)
	return val
}
