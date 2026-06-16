package integration_test

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	crand "crypto/rand"

	"github.com/cs161-staff/project2-starter-code/internal/server/handler"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

func benchCert(b *testing.B) (certFile, keyFile string) {
	b.Helper()
	dir, _ := os.MkdirTemp("", "bench-cert-*")
	b.Cleanup(func() { os.RemoveAll(dir) })
	certFile = dir + "/cert.pem"
	keyFile = dir + "/key.pem"

	key, _ := rsa.GenerateKey(crand.Reader, 2048)
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
	der, _ := x509.CreateCertificate(crand.Reader, &template, &template, &key.PublicKey, key)
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()
	keyOut, _ := os.Create(keyFile)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyOut.Close()
	return
}

func benchServer(b *testing.B) (addr string, rootCAs *x509.CertPool) {
	b.Helper()
	dir, _ := os.MkdirTemp("", "bench-srv-*")
	b.Cleanup(func() { os.RemoveAll(dir) })

	certFile, keyFile := benchCert(b)
	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { listener.Close() })

	s, err := store.Open(store.Options{Dir: dir})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.Close() })

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

	certData, _ := os.ReadFile(certFile)
	rootCAs = x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(certData)
	return listener.Addr().String(), rootCAs
}

func benchDial(b *testing.B, addr string, rootCAs *x509.CertPool) *tls.Conn {
	b.Helper()
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		RootCAs:    rootCAs,
		ServerName: "localhost",
	})
	if err != nil {
		b.Fatal(err)
	}
	return conn
}

func benchSet(b *testing.B, conn *tls.Conn, key, val []byte) {
	b.Helper()
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	vl := make([]byte, 4)
	binary.BigEndian.PutUint32(vl, uint32(len(val)))

	buf := []byte{handler.OpSet}
	buf = append(buf, kl...)
	buf = append(buf, key...)
	buf = append(buf, vl...)
	buf = append(buf, val...)
	if _, err := conn.Write(buf); err != nil {
		b.Fatal(err)
	}
	status := make([]byte, 1)
	if _, err := io.ReadFull(conn, status); err != nil {
		b.Fatal(err)
	}
	if status[0] != handler.StatusOK {
		b.Fatalf("SET failed: status %d", status[0])
	}
}

func benchGet(b *testing.B, conn *tls.Conn, key []byte) []byte {
	b.Helper()
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	buf := []byte{handler.OpGet}
	buf = append(buf, kl...)
	buf = append(buf, key...)
	if _, err := conn.Write(buf); err != nil {
		b.Fatal(err)
	}
	status := make([]byte, 1)
	if _, err := io.ReadFull(conn, status); err != nil {
		b.Fatal(err)
	}
	if status[0] == handler.StatusNotFound {
		return nil
	}
	if status[0] != handler.StatusOK {
		b.Fatalf("GET failed: status %d", status[0])
	}
	vl := make([]byte, 4)
	if _, err := io.ReadFull(conn, vl); err != nil {
		b.Fatal(err)
	}
	l := binary.BigEndian.Uint32(vl)
	val := make([]byte, l)
	if _, err := io.ReadFull(conn, val); err != nil {
		b.Fatal(err)
	}
	return val
}

func BenchmarkTLS_Set(b *testing.B) {
	addr, ca := benchServer(b)
	conn := benchDial(b, addr, ca)
	defer conn.Close()

	data := make([]byte, 256)
	rand.Read(data)
	key := []byte("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSet(b, conn, key, data)
	}
}

func BenchmarkTLS_Get(b *testing.B) {
	addr, ca := benchServer(b)
	conn := benchDial(b, addr, ca)
	defer conn.Close()

	key := []byte("bench")
	benchSet(b, conn, key, []byte("value"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGet(b, conn, key)
	}
}

func BenchmarkTLS_SetGet(b *testing.B) {
	addr, ca := benchServer(b)
	conn := benchDial(b, addr, ca)
	defer conn.Close()

	key := []byte("bench")
	val := make([]byte, 256)
	rand.Read(val)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSet(b, conn, key, val)
		benchGet(b, conn, key)
	}
}

func BenchmarkTLS_SetParallel(b *testing.B) {
	addr, ca := benchServer(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		conn := benchDial(b, addr, ca)
		defer conn.Close()
		buf := make([]byte, 64)
		rand.Read(buf)
		key := []byte("pk")
		for pb.Next() {
			benchSet(b, conn, key, buf)
		}
	})
}

func BenchmarkTLS_GetParallel(b *testing.B) {
	addr, ca := benchServer(b)

	c0 := benchDial(b, addr, ca)
	benchSet(b, c0, []byte("pk"), []byte("v"))
	c0.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		conn := benchDial(b, addr, ca)
		defer conn.Close()
		for pb.Next() {
			benchGet(b, conn, []byte("pk"))
		}
	})
}

func BenchmarkTLS_ValueSize(b *testing.B) {
	addr, ca := benchServer(b)
	conn := benchDial(b, addr, ca)
	defer conn.Close()

	for _, size := range []int{64, 1024, 65536} {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			key := []byte("vsize")
			val := make([]byte, size)
			rand.Read(val)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchSet(b, conn, key, val)
				benchGet(b, conn, key)
			}
		})
	}
}

func BenchmarkTLS_Pipeline(b *testing.B) {
	addr, ca := benchServer(b)
	conn := benchDial(b, addr, ca)
	defer conn.Close()

	const batchSize = 100
	buf := make([]byte, 0, 32*batchSize)
	status := make([]byte, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = buf[:0]
		for j := 0; j < batchSize; j++ {
			kl := make([]byte, 4)
			binary.BigEndian.PutUint32(kl, 1)
			vl := make([]byte, 4)
			binary.BigEndian.PutUint32(vl, 8)
			buf = append(buf, handler.OpSet)
			buf = append(buf, kl...)
			buf = append(buf, byte(j))
			buf = append(buf, vl...)
			buf = append(buf, []byte("pipelin8")...)
		}
		if _, err := conn.Write(buf); err != nil {
			b.Fatal(err)
		}
		for j := 0; j < batchSize; j++ {
			if _, err := io.ReadFull(conn, status); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkTLS_ConcurrencyScale(b *testing.B) {
	addr, ca := benchServer(b)

	for _, conns := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("conns-%d", conns), func(b *testing.B) {
			var wg sync.WaitGroup
			b.SetParallelism(conns)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				wg.Add(1)
				defer wg.Done()
				conn := benchDial(b, addr, ca)
				defer conn.Close()
				key := []byte("cs")
				val := make([]byte, 128)
				for pb.Next() {
					benchSet(b, conn, key, val)
				}
			})
			wg.Wait()
		})
	}
}
