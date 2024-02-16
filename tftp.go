package main
import (
	"github.com/pin/tftp"
	"fmt"
	"os"
	"io"

)
// TFTPHandler serves files via TFTP.
func TFTPHandler(rootDir string) {
        server := tftp.NewServer(
                func(filename string, rf io.ReaderFrom) error {
                        fmt.Printf("TFTP Request: %s\n", filename)

                        // Construct the full path to the predetermined file
                        filePath := rootDir + filename

                        // Open the predetermined file for reading
                        file, err := os.Open(filePath)
                        if err != nil {
                                return err
                        }
                        defer file.Close()

                        // Use the ReaderFrom interface to transfer the file content to the TFTP client
                        _, err = rf.ReadFrom(file)
                        return err
                },
                func(filename string, wt io.WriterTo) error {
                        // Dummy write handler, since we are not accepting file uploads
                        return fmt.Errorf("Write operation not supported")
                },
        )

        err := server.ListenAndServe("0.0.0.0:96")
        if err != nil {
                fmt.Printf("Error starting TFTP server: %s\n", err)
        }
}

