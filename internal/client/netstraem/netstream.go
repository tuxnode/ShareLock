package netstraem

import (
	"io"
	"log"
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

	written, err := io.Copy(*conn, file)
	if err != nil {
		return nil
	}

	log.Printf("Successfully sent %d bytes for file: %s\n", written, filename)
	return nil
}
