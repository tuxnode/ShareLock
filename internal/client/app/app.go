package app

import (
	"crypto/tls"
	"net"

	client "github.com/cs161-staff/project2-starter-code/internal/client/encryption"
	"github.com/cs161-staff/project2-starter-code/internal/client/netstream"
	"github.com/cs161-staff/project2-starter-code/internal/userlib"
	"github.com/google/uuid"
)

type Client struct {
	user *client.User
}

func Connect(address string, tlsEnabled bool) {
	userlib.Connect(address, tlsEnabled)
}

func Disconnect() {
	userlib.Disconnect()
}

func (c *Client) InitUser(username string, password string) error {
	user, err := client.InitUser(username, password)
	if err != nil {
		return err
	}
	c.user = user
	return nil
}

func (c *Client) GetUser(username string, password string) error {
	user, err := client.GetUser(username, password)
	if err != nil {
		return err
	}
	c.user = user
	return nil
}

func (c *Client) StoreFile(filename string, content []byte) error {
	return c.user.StoreFile(filename, content)
}

func (c *Client) LoadFile(filename string) ([]byte, error) {
	return c.user.LoadFile(filename)
}

func (c *Client) AppendToFile(filename string, content []byte) error {
	return c.user.AppendToFile(filename, content)
}

func (c *Client) CreateInvitation(filename string, recipientUsername string) (uuid.UUID, error) {
	return c.user.CreateInvitation(filename, recipientUsername)
}

func (c *Client) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	return c.user.AcceptInvitation(senderUsername, invitationPtr, filename)
}

func (c *Client) RevokeAccess(filename string, recipientUsername string) error {
	return c.user.RevokeAccess(filename, recipientUsername)
}

func (c *Client) ReadFile(filename string, address string) error {
	return c.ReadFileTLS(filename, address, true)
}

func (c *Client) ReadFileTLS(filename string, address string, tlsEnabled bool) error {
	var conn net.Conn
	var err error

	if tlsEnabled {
		conn, err = tls.Dial("tcp", address, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	return netstream.FileReceiver(filename, &conn)
}
