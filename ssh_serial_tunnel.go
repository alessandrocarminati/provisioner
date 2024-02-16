package main

import (
	"log"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/tarm/serial"
)

func SSHHandler(sshPort string, sshIn chan<- []byte, sshOut <-chan []byte) {
	ssh.Handle(func(s ssh.Session) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer s.Close()
			for {
				data := <-sshOut
				_, err := s.Write(data)
				if err != nil {
					log.Println("Error writing to SSH session:", err)
					return
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer s.Close()
			buf := make([]byte, 4096)
			for {
				n, err := s.Read(buf)
				if err != nil {
					log.Println("Error reading from SSH session:", err)
					return
				}
				sshIn <- buf[:n]
			}
		}()
		wg.Wait()
	})

	log.Println("Starting SSH server on port", sshPort)
	err := ssh.ListenAndServe(":"+sshPort, nil)
	if err != nil {
		log.Fatal("Error starting SSH server:", err)
	}
}

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


