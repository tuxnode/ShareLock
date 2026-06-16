package server

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/cs161-staff/project2-starter-code/internal/server/handler"
	"github.com/cs161-staff/project2-starter-code/internal/server/store"
)

type Server struct {
	store   *store.Store
	handler *handler.Handler
	config  Config
}

type Config struct {
	Addr    string
	DataDir string
	Cert    string
	Key     string
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

	cert, err := tls.LoadX509KeyPair(srv.config.Cert, srv.config.Key)
	if err != nil {
		return err
	}

	listener, err := tls.Listen("tcp", srv.config.Addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
	})
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("server listening on %s (data: %s)", srv.config.Addr, srv.config.DataDir)

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
