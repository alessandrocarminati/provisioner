package main
import (
	"log"
)

func main() {

	config, err :=  fetch_config("config.json")
	if err!= nil {
		log.Fatal(err)
	}
	go TFTPHandler(config.TFTPDirectory)
	go HTTPHandler(config.TFTPDirectory, config.HTTPPort)

	serialToSSH := make(chan []byte)
	sshToSerial := make(chan []byte)
	go SSHHandler(config.SSHSerTun, "tunnel", serialToSSH, sshToSerial, false)
	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialToSSH, sshToSerial)


	monitorToSSH := make(chan []byte)
	sshToMonitor := make(chan []byte)
	go SSHHandler(config.SSHMon, "monitor", monitorToSSH, sshToMonitor, true)
	go Monitor(monitorToSSH, sshToMonitor, config.Monitor)
	select {}
}
