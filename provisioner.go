package main

import (
    "log"
    "sync"

    "github.com/gliderlabs/ssh"
    "github.com/tarm/serial"
)


// SSHHandler handles SSH communication.
func SSHHandler(sshPort string, sshIn chan<- []byte, sshOut <-chan []byte) {
    // Define SSH server configuration
    ssh.Handle(func(s ssh.Session) {
        var wg sync.WaitGroup
	wg.Add(1)
        // Channel for sending data from SSH to serial
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
        // Channel for receiving data from serial to SSH
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

    // Start SSH server
    log.Println("Starting SSH server on port", sshPort)
    err := ssh.ListenAndServe(":"+sshPort, nil)
    if err != nil {
        log.Fatal("Error starting SSH server:", err)
    }
}

// SerialHandler handles serial communication.
func SerialHandler(serialPort string, BaudRate int, serialIn <-chan []byte, serialOut chan<- []byte) {
    var wg sync.WaitGroup
    // Open serial port
    cfg := &serial.Config{Name: serialPort, Baud: BaudRate}
    serialPortInstance, err := serial.OpenPort(cfg)
    if err != nil {
        log.Fatal("Error opening serial port:", err)
    }
    defer serialPortInstance.Close()

    // Channel for sending data from serial to SSH
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

    // Channel for receiving data from SSH to serial
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


func main() {

	config, err :=  fetch_config("config.json")
	if err!= nil {
		log.Fatal(err)
	}
	go TFTPHandler(config.TFTPDirectory)
	go HTTPHandler(config.TFTPDirectory, config.HTTPPort)

	serialToSSH := make(chan []byte)
	sshToSerial := make(chan []byte)
	go SSHHandler(config.SSHPort, serialToSSH, sshToSerial)
	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialToSSH, sshToSerial)

	select {}
}
