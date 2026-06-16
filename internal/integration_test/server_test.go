package integration_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	crand "math/rand"

	"github.com/cs161-staff/project2-starter-code/internal/server/handler"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func generateTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	dir, _ := os.MkdirTemp("", "integ-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	certFile = dir + "/cert.pem"
	keyFile = dir + "/key.pem"

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()

	keyOut, _ := os.Create(keyFile)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyOut.Close()
	return certFile, keyFile
}

type client struct {
	conn *tls.Conn
	t    *testing.T
}

func newClient(t *testing.T, addr string, certFile string) *client {
	t.Helper()
	cert, _ := os.ReadFile(certFile)
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(cert)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		RootCAs:    rootCAs,
		ServerName: "localhost",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &client{conn: conn, t: t}
}

func (c *client) close() {
	c.conn.Close()
}

func (c *client) set(key, val []byte) {
	c.t.Helper()
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	vl := make([]byte, 4)
	binary.BigEndian.PutUint32(vl, uint32(len(val)))

	buf := []byte{handler.OpSet}
	buf = append(buf, kl...)
	buf = append(buf, key...)
	buf = append(buf, vl...)
	buf = append(buf, val...)
	if _, err := c.conn.Write(buf); err != nil {
		c.t.Fatal(err)
	}
	status := c.readStatus()
	if status != handler.StatusOK {
		c.t.Fatalf("SET failed: status %d", status)
	}
}

func (c *client) get(key []byte) []byte {
	c.t.Helper()
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))

	buf := []byte{handler.OpGet}
	buf = append(buf, kl...)
	buf = append(buf, key...)
	if _, err := c.conn.Write(buf); err != nil {
		c.t.Fatal(err)
	}
	status := c.readStatus()
	if status == handler.StatusNotFound {
		return nil
	}
	if status != handler.StatusOK {
		c.t.Fatalf("GET failed: status %d", status)
	}
	vl := make([]byte, 4)
	if _, err := c.conn.Read(vl); err != nil {
		c.t.Fatal(err)
	}
	l := binary.BigEndian.Uint32(vl)
	val := make([]byte, l)
	if _, err := c.conn.Read(val); err != nil {
		c.t.Fatal(err)
	}
	return val
}

func (c *client) delete(key []byte) {
	c.t.Helper()
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))

	buf := []byte{handler.OpDelete}
	buf = append(buf, kl...)
	buf = append(buf, key...)
	if _, err := c.conn.Write(buf); err != nil {
		c.t.Fatal(err)
	}
	status := c.readStatus()
	if status != handler.StatusOK {
		c.t.Fatalf("DELETE failed: status %d", status)
	}
}

func (c *client) readStatus() byte {
	c.t.Helper()
	buf := make([]byte, 1)
	if _, err := c.conn.Read(buf); err != nil {
		c.t.Fatal(err)
	}
	return buf[0]
}

func startTestServer(t *testing.T, certFile, keyFile string) string {
	t.Helper()
	dir, _ := os.MkdirTemp("", "srv-data-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{loadCert(t, certFile, keyFile)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	s, err := store.Open(store.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	h := handler.New(s)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				for {
					if err := h.Handle(c); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	return listener.Addr().String()
}

func loadCert(t *testing.T, certFile, keyFile string) tls.Certificate {
	t.Helper()
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestBasicKV(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	c.set([]byte("hello"), []byte("world"))
	val := c.get([]byte("hello"))
	if string(val) != "world" {
		t.Fatalf("got %s, want world", val)
	}
}

func TestGetNonExistent(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	val := c.get([]byte("nope"))
	if val != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestDeleteKey(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	c.set([]byte("k"), []byte("v"))
	c.delete([]byte("k"))
	val := c.get([]byte("k"))
	if val != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestOverwriteKey(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	c.set([]byte("k"), []byte("v1"))
	c.set([]byte("k"), []byte("v2"))
	val := c.get([]byte("k"))
	if string(val) != "v2" {
		t.Fatalf("got %s, want v2", val)
	}
}

func TestLargeValue(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	large := make([]byte, 512<<10) // 512KB
	crand.Read(large)
	c.set([]byte("large"), large)
	val := c.get([]byte("large"))
	if len(val) != len(large) {
		t.Fatalf("length mismatch: %d vs %d", len(val), len(large))
	}
}

func TestConcurrentClients(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c := newClient(t, addr, certFile)
			defer c.close()
			key := []byte{byte('a' + id)}
			val := []byte{byte(id)}
			c.set(key, val)
			got := c.get(key)
			if len(got) != 1 || got[0] != val[0] {
				t.Errorf("concurrent client %d failed", id)
			}
		}(i)
	}
	wg.Wait()
}

func TestManyOps(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	n := 100
	for i := 0; i < n; i++ {
		key := []byte{byte(i % 256)}
		c.set(key, key)
	}
	for i := 0; i < n; i++ {
		key := []byte{byte(i % 256)}
		val := c.get(key)
		if len(val) != 1 || val[0] != key[0] {
			t.Fatalf("many ops failed at %d", i)
		}
	}
}

func TestMultipleOpsSingleConn(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	addr := startTestServer(t, certFile, keyFile)

	c := newClient(t, addr, certFile)
	defer c.close()

	pairs := []struct{ k, v string }{
		{"a", "1"}, {"b", "2"}, {"c", "3"},
		{"x", "10"}, {"y", "20"}, {"z", "30"},
	}
	for _, p := range pairs {
		c.set([]byte(p.k), []byte(p.v))
	}
	for _, p := range pairs {
		val := c.get([]byte(p.k))
		if string(val) != p.v {
			t.Fatalf("key %s: got %s, want %s", p.k, val, p.v)
		}
	}
}
