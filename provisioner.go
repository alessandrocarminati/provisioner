package main
import (
	"log"
	"fmt"
)

var Build string
var Version string
var Hash string
var Dirty string

func main() {

	cmdline := parseCMDline()
	DebugLevel = cmdline.DebLev

	if cmdline.VerJ {
		fmt.Printf("{\n\t\"Major\": \"%s\",\n\t\"Minor\": \"%s\",\n\t\"Hash\": \"%s\",\n\t\"Dirty\": \"%s\"\n}\n", Version, Build, Hash, Dirty)
		return
	}

	if cmdline.VerRq {
		fmt.Printf("Provisioner Ver. %s.%s (%s) %s\n", Version, Build, Hash, Dirty)
		return
	}
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

	debugPrint(log.Printf, levelWarning, "Provisioner Ver. %s.%s (%s) %s\n", Version, Build, Hash, Dirty)
	go syslog_service(config.LogFile, config.SyslogPort)
	go TFTPHandler(config.TFTPDirectory)
	go HTTPHandler(config.TFTPDirectory, config.HTTPPort)

	ssh_init(10, 10)
	serialRouter := NewRouter(10)
	serialRouter.Router()
	serialRouter.AttachAt(0, SrcMachine)

	go SSHHandler(config.SSHSerTun, "tunnel", serialRouter, false)
	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialRouter.In[0], serialRouter.Out[0])


	monitorRouter := NewRouter(10)
	monitorRouter.Router()
	monitorRouter.AttachAt(0, SrcMachine)

	go SSHHandler(config.SSHMon, "monitor", monitorRouter, true)
	go Monitor(monitorRouter.In[0], monitorRouter.Out[0], config.Monitor)
	go calendarPoller()
	select {}
}
