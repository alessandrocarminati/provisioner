package main
import (
	"github.com/pin/tftp"
	"fmt"
	"os"
	"io"
	"log"
)
func TFTPHandler(rootDir string) {
	server := tftp.NewServer(
		func(filename string, rf io.ReaderFrom) error {
			fmt.Printf("TFTP Request: %s\n", filename)

			filePath := rootDir + filename

			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = rf.ReadFrom(file)
			return err
		},
		func(filename string, wt io.WriterTo) error {
			return fmt.Errorf("Write operation not supported")
		},
	)
	bind:="0.0.0.0:69"
	err := server.ListenAndServe(bind)
	if err != nil {
		fmt.Printf("Error starting TFTP server: %s\n", err)
	}
	log.Printf("TFTP server is active on %s", bind)
}

