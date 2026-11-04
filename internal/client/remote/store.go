package remote

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

const (
	opGet    = byte(0x01)
	opSet    = byte(0x02)
	opDelete = byte(0x03)

	statusOK       = byte(0x00)
	statusNotFound = byte(0x01)
	statusError    = byte(0x02)
)

var (
	configMu sync.RWMutex
	address  string
	useTLS   bool
	local    sync.Map
)

func Connect(addr string, tlsEnabled bool) {
	configMu.Lock()
	address = addr
	useTLS = tlsEnabled
	configMu.Unlock()
}

func Disconnect() {
	configMu.Lock()
	address = ""
	configMu.Unlock()
}

func Reset() {
	Disconnect()
	local = sync.Map{}
}

func Get(key []byte) ([]byte, bool, error) {
	conn, configured, err := dial()
	if err != nil {
		return nil, false, err
	}
	if !configured {
		value, ok := local.Load(string(key))
		if !ok {
			return nil, false, nil
		}
		return append([]byte(nil), value.([]byte)...), true, nil
	}
	defer conn.Close()

	if err := writeOperation(conn, opGet, key, nil); err != nil {
		return nil, false, err
	}
	status, err := readStatus(conn)
	if err != nil {
		return nil, false, err
	}
	switch status {
	case statusNotFound:
		return nil, false, nil
	case statusOK:
		value, err := readValue(conn)
		return value, err == nil, err
	default:
		return nil, false, readServerError(conn, status)
	}
}

func Set(key, value []byte) error {
	conn, configured, err := dial()
	if err != nil {
		return err
	}
	if !configured {
		local.Store(string(key), append([]byte(nil), value...))
		return nil
	}
	defer conn.Close()

	if err := writeOperation(conn, opSet, key, value); err != nil {
		return err
	}
	status, err := readStatus(conn)
	if err != nil {
		return err
	}
	if status != statusOK {
		return readServerError(conn, status)
	}
	return nil
}

func Delete(key []byte) error {
	conn, configured, err := dial()
	if err != nil {
		return err
	}
	if !configured {
		local.Delete(string(key))
		return nil
	}
	defer conn.Close()

	if err := writeOperation(conn, opDelete, key, nil); err != nil {
		return err
	}
	status, err := readStatus(conn)
	if err != nil {
		return err
	}
	if status != statusOK {
		return readServerError(conn, status)
	}
	return nil
}

func dial() (net.Conn, bool, error) {
	configMu.RLock()
	addr, tlsEnabled := address, useTLS
	configMu.RUnlock()
	if addr == "" {
		return nil, false, nil
	}
	if tlsEnabled {
		conn, err := tls.Dial("tcp", addr, &tls.Config{MinVersion: tls.VersionTLS12})
		if err != nil {
			return nil, true, fmt.Errorf("connect to storage server: %w", err)
		}
		return conn, true, nil
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, true, fmt.Errorf("connect to storage server: %w", err)
	}
	return conn, true, nil
}

func writeOperation(w io.Writer, op byte, key, value []byte) error {
	if uint64(len(key)) > uint64(^uint32(0)) || uint64(len(value)) > uint64(^uint32(0)) {
		return errors.New("storage key or value exceeds protocol limit")
	}
	if err := writeFull(w, []byte{op}); err != nil {
		return err
	}
	if err := writeLength(w, uint32(len(key))); err != nil {
		return err
	}
	if err := writeFull(w, key); err != nil {
		return err
	}
	if op == opSet {
		if err := writeLength(w, uint32(len(value))); err != nil {
			return err
		}
		return writeFull(w, value)
	}
	return nil
}

func writeLength(w io.Writer, length uint32) error {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], length)
	return writeFull(w, data[:])
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func readStatus(r io.Reader) (byte, error) {
	var status [1]byte
	_, err := io.ReadFull(r, status[:])
	return status[0], err
}

func readValue(r io.Reader) ([]byte, error) {
	var size [4]byte
	if _, err := io.ReadFull(r, size[:]); err != nil {
		return nil, err
	}
	value := make([]byte, binary.BigEndian.Uint32(size[:]))
	_, err := io.ReadFull(r, value)
	return value, err
}

func readServerError(r io.Reader, status byte) error {
	if status != statusError {
		return fmt.Errorf("storage server returned status %d", status)
	}
	message, err := readValue(r)
	if err != nil {
		return fmt.Errorf("storage server error (failed to read message): %w", err)
	}
	return fmt.Errorf("storage server: %s", message)
}
