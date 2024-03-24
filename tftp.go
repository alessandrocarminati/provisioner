package main
import (
	"github.com/pin/tftp"
	"fmt"
	"os"
	"io"
	"log"
)
func TFTPHandler(rootDir string) {
	debugPrint(log.Printf, levelWarning, "Starting TFTP service with rootdir: %s", rootDir )
	server := tftp.NewServer(
		func(filename string, rf io.ReaderFrom) error {
			debugPrint(log.Printf, levelNotice, "TFTP Request: %s\n", filename)

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
		debugPrint(log.Printf, levelError, "Error starting TFTP server: %s", err.Error())
	}
	debugPrint(log.Printf, levelWarning,"TFTP server is active on %s", bind)
}
