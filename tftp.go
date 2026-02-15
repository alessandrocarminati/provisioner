package main

import (
	//	"github.com/pin/tftp"
	"fmt"
	"github.com/pin/tftp/v3"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func TFTPHandler(rootDir string) {
	debugPrint(log.Printf, levelWarning, "Starting TFTP service with rootdir: %s", rootDir)
	server := tftp.NewServer(
		func(filename string, rf io.ReaderFrom) error {
			var (
				err  error
				resp io.ReadCloser
			)
			debugPrint(log.Printf, levelNotice, "TFTP Request: %s\n", filename)
			if strings.HasPrefix(filename, "http___") {
				url := strings.Replace(filename, "http___", "http://", 1)
				debugPrint(log.Printf, levelNotice, "TFTP Remote proxy service Request: %s\n", url)
				httpResp, err := http.Get(url)
				if err != nil {
					return err
				}
				if httpResp.StatusCode != http.StatusOK {
					httpResp.Body.Close()
					return fmt.Errorf("failed to fetch file: %s", httpResp.Status)
				}
				resp = httpResp.Body
				defer resp.Close()
			} else {
				debugPrint(log.Printf, levelNotice, "TFTP Local service Request: %s\n", filename)
				filePath := rootDir + filename

				file, err := os.Open(filePath)
				if err != nil {
					return err
				}
				resp = file
				defer resp.Close()
			}
			_, err = rf.ReadFrom(resp)
			return err
		},
		func(filename string, wt io.WriterTo) error {
			debugPrint(log.Printf, levelNotice, "TFTP Write Request: %s\n", filename)

			filePath := rootDir + filename
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = wt.WriteTo(file)
			return err
		},
	)
	server.SetBlockSize(1468)
	server.SetAnticipate(64)
	bind := "0.0.0.0:69"
	err := server.ListenAndServe(bind)
	if err != nil {
		debugPrint(log.Printf, levelError, "Error starting TFTP server: %s", err.Error())
	}
	debugPrint(log.Printf, levelWarning, "TFTP server is active on %s", bind)
}
