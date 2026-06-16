package netstraem

import (
	"io"
	"net"
	"os"
	"path"
)

func FileSeander(filename string, conn *net.Conn) error {
	safename := path.Base(filename)

	file, err := os.OpenFile(safename, os.O_RDONLY, 0444)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(*conn, file)
	if err != nil {
		return nil
	}

	return nil
}

func FileReceiver(filename string, conn *net.Conn) error {
	safename := path.Base(filename)

	file, err := os.OpenFile(safename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, *conn)
	if err != nil {
		return err
	}
	return nil
}
