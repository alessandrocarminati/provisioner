package main

import (
	"log"
	"sync"
	"os"
	"strconv"
	"time"
	"fmt"
	"context"

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
	if err == nil {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := serialPortInstance.Read(buf)
			if err != nil {
				debugPrint(log.Printf, levelError, "Error reading from serial port: %s", err.Error())
				cancel()
				return
			}
			if n <= 0 {
				continue
			}
			debugPrint(log.Printf, levelCrazy, "Read %d bytes '%s'", n, string(buf[:n]))
			// send bytes one by one (compatible) but honour ctx
			for i := 0; i < n; i++ {
				select {
				case serialOut <- buf[i]:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case data := <-serialIn:
				// consider batching multiple bytes into a small buffer for efficiency
				_, err := serialPortInstance.Write([]byte{data})
				if err != nil {
					debugPrint(log.Printf, levelError, "Error writing to serial port: %s", err.Error())
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
}

func IsBusy(filePath string) (bool, error) {
	currentPID := os.Getpid()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		name := entry.Name()
		pid, err := strconv.Atoi(name)
		if err != nil || pid == currentPID {
			continue
		}
		fdDirPath := fmt.Sprintf("/proc/%d/fd", pid)
		fdEntries, err := os.ReadDir(fdDirPath)
		if err != nil {
			continue
		}
		for _, fdEntry := range fdEntries {
			fdPath := fmt.Sprintf("/proc/%d/fd/%s", pid, fdEntry.Name())
			target, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}
			if target == filePath {
				return true, nil
			}
		}
	}
	return false, nil
}
