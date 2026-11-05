package server

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/tuxnode/ShareLock/internal/server/handler"
	"github.com/tuxnode/ShareLock/internal/server/store"
)

type Server struct {
	store   *store.Store
	handler *handler.Handler
	config  Config
}

type Config struct {
	Addr       string
	DataDir    string
	Cert       string
	Key        string
	TLSEnabled bool
}

func New(cfg Config) (*Server, error) {
	s, err := store.Open(store.Options{Dir: cfg.DataDir})
	if err != nil {
		return nil, err
	}
	return &Server{
		store:   s,
		handler: handler.New(s),
		config:  cfg,
	}, nil
}

func (srv *Server) Run() error {
	defer srv.store.Close()

	var listener net.Listener
	var err error

	if srv.config.TLSEnabled {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(srv.config.Cert, srv.config.Key)
		if err != nil {
			return err
		}
		listener, err = tls.Listen("tcp", srv.config.Addr, &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.NoClientCert,
		})
	} else {
		listener, err = net.Listen("tcp", srv.config.Addr)
	}
	if err != nil {
		return err
	}
	defer listener.Close()

	proto := "TLS"
	if !srv.config.TLSEnabled {
		proto = "TCP"
	}
	log.Printf("server listening on %s (%s, data: %s)", srv.config.Addr, proto, srv.config.DataDir)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go srv.handleConn(conn)
	}
}

func (srv *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		if err := srv.handler.Handle(conn); err != nil {
			log.Printf("connection closed: %v", err)
			return
		}
	}
}
