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
	user              *client.User
	userService       *client.UserService
	fileService       *client.FileService
	invitationService *client.InvitationService
}

func (c *Client) ensureServices() {
	if c.userService == nil {
		storage := client.NewUserlibStorage()
		keyStore := client.NewUserlibKeyStore()
		c.userService = client.NewUserService(storage, keyStore)
		c.fileService = client.NewFileService(storage, keyStore)
		c.invitationService = client.NewInvitationService(storage, keyStore)
	}
}

func Connect(address string, tlsEnabled bool) {
	userlib.Connect(address, tlsEnabled)
}

func Disconnect() {
	userlib.Disconnect()
}

func (c *Client) InitUser(username string, password string) error {
	c.ensureServices()
	user, err := c.userService.InitUser(username, password)
	if err != nil {
		return err
	}
	c.user = user
	return nil
}

func (c *Client) GetUser(username string, password string) error {
	c.ensureServices()
	user, err := c.userService.GetUser(username, password)
	if err != nil {
		return err
	}
	c.user = user
	return nil
}

func (c *Client) StoreFile(filename string, content []byte) error {
	return c.fileService.StoreFile(c.user, filename, content)
}

func (c *Client) LoadFile(filename string) ([]byte, error) {
	return c.fileService.LoadFile(c.user, filename)
}

func (c *Client) AppendToFile(filename string, content []byte) error {
	return c.fileService.AppendToFile(c.user, filename, content)
}

func (c *Client) CreateInvitation(filename string, recipientUsername string) (uuid.UUID, error) {
	return c.invitationService.CreateInvitation(c.user, filename, recipientUsername)
}

func (c *Client) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	return c.invitationService.AcceptInvitation(c.user, senderUsername, invitationPtr, filename)
}

func (c *Client) RevokeAccess(filename string, recipientUsername string) error {
	return c.invitationService.RevokeAccess(c.user, filename, recipientUsername)
}

func (c *Client) ListFiles() []string {
	return c.fileService.ListFiles(c.user)
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
