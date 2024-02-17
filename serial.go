package main

import (
	"log"
	"sync"

	"github.com/tarm/serial"
)

func SerialHandler(serialPort string, BaudRate int, serialIn <-chan []byte, serialOut chan<- []byte) {
	var wg sync.WaitGroup
	cfg := &serial.Config{Name: serialPort, Baud: BaudRate}
	serialPortInstance, err := serial.OpenPort(cfg)
	if err != nil {
		log.Fatal("Error opening serial port:", err)
	}
	defer serialPortInstance.Close()

	wg.Add(1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := serialPortInstance.Read(buf)
			if err != nil {
				log.Println("Error reading from serial port:", err)
				return
			}
			serialOut <- buf[:n]
		}
	}()

	wg.Add(1)
	go func() {
		for {
			data := <-serialIn
			_, err := serialPortInstance.Write(data)
			if err != nil {
				log.Println("Error writing to serial port:", err)
				return
			}
		}
	}()
	wg.Wait()
}


