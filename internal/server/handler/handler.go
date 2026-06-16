package handler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/dgraph-io/badger/v3"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

const (
	OpGet    = byte(0x01)
	OpSet    = byte(0x02)
	OpDelete = byte(0x03)
)

const (
	StatusOK        = byte(0x00)
	StatusNotFound  = byte(0x01)
	StatusError     = byte(0x02)
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Handle(rw io.ReadWriter) error {
	var op [1]byte
	if _, err := io.ReadFull(rw, op[:]); err != nil {
		return err
	}

	var keyLen [4]byte
	if _, err := io.ReadFull(rw, keyLen[:]); err != nil {
		return err
	}
	kl := binary.BigEndian.Uint32(keyLen[:])
	key := make([]byte, kl)
	if _, err := io.ReadFull(rw, key); err != nil {
		return err
	}

	switch op[0] {
	case OpGet:
		return h.handleGet(rw, key)
	case OpSet:
		return h.handleSet(rw, key)
	case OpDelete:
		return h.handleDelete(rw, key)
	default:
		return writeError(rw, fmt.Sprintf("unknown op: %d", op[0]))
	}
}

func (h *Handler) handleGet(rw io.ReadWriter, key []byte) error {
	val, err := h.store.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return writeStatus(rw, StatusNotFound)
	}
	if err != nil {
		return writeError(rw, err.Error())
	}
	return writeValue(rw, val)
}

func (h *Handler) handleSet(rw io.ReadWriter, key []byte) error {
	var valLen [4]byte
	if _, err := io.ReadFull(rw, valLen[:]); err != nil {
		return err
	}
	vl := binary.BigEndian.Uint32(valLen[:])
	val := make([]byte, vl)
	if _, err := io.ReadFull(rw, val); err != nil {
		return err
	}
	if err := h.store.Set(key, val); err != nil {
		return writeError(rw, err.Error())
	}
	return writeStatus(rw, StatusOK)
}

func (h *Handler) handleDelete(rw io.ReadWriter, key []byte) error {
	if err := h.store.Delete(key); err != nil {
		return writeError(rw, err.Error())
	}
	return writeStatus(rw, StatusOK)
}

func writeStatus(rw io.Writer, status byte) error {
	_, err := rw.Write([]byte{status})
	return err
}

func writeValue(rw io.Writer, val []byte) error {
	buf := make([]byte, 1+4+len(val))
	buf[0] = StatusOK
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(val)))
	copy(buf[5:], val)
	_, err := rw.Write(buf)
	return err
}

func writeError(rw io.Writer, msg string) error {
	buf := make([]byte, 1+4+len(msg))
	buf[0] = StatusError
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(msg)))
	copy(buf[5:], msg)
	_, err := rw.Write(buf)
	return err
}
