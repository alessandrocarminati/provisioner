package main

import (
	"log"
	"sync"
	"os"
	"strconv"
	"time"
	"fmt"

	"github.com/tarm/serial"
)

func checker(serialPort string){
	for {
		time.Sleep(30 * time.Second)
		busy, err := IsBusy(serialPort)
		if err==nil {
			if busy {
				debugPrint(log.Printf, levelError, "%s is busy!", serialPort)
			}
		} else {
			debugPrint(log.Printf, levelWarning, "Can't check if %s is busy", serialPort)
		}
	}
}


func SerialHandler(serialPort string, BaudRate int, serialIn <-chan byte, serialOut chan<- byte) {
	var wg sync.WaitGroup
	busy, err := IsBusy(serialPort)
	if err==nil {
		if busy {
			debugPrint(log.Printf, levelPanic, "serial port %s is busy", serialPort)
		}
	} else {
		debugPrint(log.Printf, levelWarning, "Can't check if %s is busy", serialPort)
	}
	go checker(serialPort)
	cfg := &serial.Config{Name: serialPort, Baud: BaudRate}
	serialPortInstance, err := serial.OpenPort(cfg)
	if err != nil {
		debugPrint(log.Printf, levelPanic, "Error opening serial port: %s", err.Error())
	}
	defer serialPortInstance.Close()

	wg.Add(1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := serialPortInstance.Read(buf)
			debugPrint(log.Printf, levelCrazy, "Read %d bytes '%s'", n, string(buf))
			if err != nil {
				debugPrint(log.Printf, levelError, "Error reading from serial port: %s", err.Error())
				return
			}
			for i:=0;i<n;i++ {
				serialOut <- buf[i]
			}
		}
	}()

	wg.Add(1)
	go func() {
		for {
			data := <-serialIn
			_, err := serialPortInstance.Write([]byte{data})
			if err != nil {
				debugPrint(log.Printf, levelError, "Error writing to serial port: %s", err.Error())
				return
			}
		}
	}()
	wg.Wait()
}

func IsBusy(filePath string) (bool, error) {
	currentPID := os.Getpid()
	procDir, err := os.Open("/proc")
	if err != nil {
		return false, err
	}
	defer procDir.Close()
	entries, err := procDir.Readdirnames(0)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if pid, err := strconv.Atoi(entry); err == nil && pid != currentPID {
			fdDirPath := fmt.Sprintf("/proc/%d/fd", pid)
			fdDir, err := os.Open(fdDirPath)
			if err != nil {
				continue
			}
			defer fdDir.Close()
			fdEntries, err := fdDir.Readdirnames(0)
			if err != nil {
				continue
			}
			for _, fdEntry := range fdEntries {
				fdPath := fmt.Sprintf("/proc/%d/fd/%s", pid, fdEntry)
				target, err := os.Readlink(fdPath)
				if err != nil {
					continue
				}
				if target == filePath {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

