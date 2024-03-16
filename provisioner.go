package main
import (
	"log"
	"fmt"
)

func main() {

	cmdline := parseCMDline()

	if cmdline.Help {
		fmt.Println(helpText())
		return
	}

	if cmdline.CalFetch {
		_, err := NextReservation("cred.json", "primary", nil)
		if err!=nil {
			fmt.Println("Error accessing calendar: ", err)
			return
		}
		fmt.Println("Calendar access ok")
		return
		}
	if cmdline.GenKeys {
		err := GenerateKeyPair("private", "public")
		if err != nil {
			fmt.Println("Error generating keys: ", err)
			return
		}
		fmt.Println("keys generated")
		return
	}

	if cmdline.Enc {
		if cmdline.Key!="" {
			data, err := EncryptConfig(cmdline.ConfigFN, cmdline.Key)
			if err != nil {
				fmt.Println("Error in crypting config: ", err)
				return
			}
			err = WriteFile("config.rsa", data)
			if err != nil {
				fmt.Println("Error in crypting config: ", err)
				return
			}
			fmt.Println("config.rsa written")
			return
		}
	}

	config, err :=  fetch_config(cmdline.ConfigFN, cmdline.Key)
	if err!= nil {
		log.Fatal(err)
	}

	go syslog_service(config.LogFile, config.SyslogPort)
	go TFTPHandler(config.TFTPDirectory)
	go HTTPHandler(config.TFTPDirectory, config.HTTPPort)

	serialToSSH := make(chan byte)
	sshToSerial := make(chan byte)
	go SSHHandler(config.SSHSerTun, "tunnel", serialToSSH, sshToSerial, false)
	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialToSSH, sshToSerial)


	monitorToSSH := make(chan byte)
	sshToMonitor := make(chan byte)
	go SSHHandler(config.SSHMon, "monitor", monitorToSSH, sshToMonitor, true)
	go Monitor(monitorToSSH, sshToMonitor, config.Monitor)
	go calendarPoller()
	select {}
}
