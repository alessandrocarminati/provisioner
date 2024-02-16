package main
import (
	"github.com/pin/tftp"
	"fmt"
	"os"
	"io"

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

	err := server.ListenAndServe("0.0.0.0:96")
	if err != nil {
		fmt.Printf("Error starting TFTP server: %s\n", err)
	}
}

