package main

import (
	"log"
	"sync"

	"github.com/tarm/serial"
)

func SerialHandler(serialPort string, BaudRate int, serialIn <-chan byte, serialOut chan<- byte) {
	var wg sync.WaitGroup
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
			if err != nil {
				debugPrint(log.Printf, levelDebug, "Error reading from serial port: %s", err.Error())
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
				debugPrint(log.Printf, levelDebug, "Error writing to serial port: %s", err.Error())
				return
			}
		}
	}()
	wg.Wait()
}
