package main
import (
	"bufio"
	"os"
	"log"
	"fmt"
	"golang.org/x/term"
	"syscall"
	"strings"
)

var Build string
var Version string
var Hash string
var Dirty string

func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	return strings.TrimSpace(password), nil
}

func readUserid() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(username), nil
}

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

	Dacl = cmdline.Dacl
	if cmdline.Dacl != "All" {
		fmt.Printf("Debug ACL Activated! Waching -> %s\n", cmdline.Dacl)
	}

	config, err :=  fetch_config(cmdline.ConfigFN, cmdline.Key)
	if err!= nil {
		log.Fatal(err)
	}


	value, ok := config.Monitor["pdu_type"]
	if ok && (value=="beaker") {
		//check beaker username
		_, ok = config.Monitor["beaker_username"]
		if !ok {
			fmt.Println("Beaker userid is not in config, please enter it manually")
			s, err := readUserid()
			if err != nil {
				fmt.Printf("Error reading beaker Userid: %s\n", err.Error())
				return
			}
			config.Monitor["beaker_username"] = s
		}

		//check beaker password
		_, ok = config.Monitor["beaker_password"]
		if !ok {
			fmt.Println("Beaker password is not in config, please enter it manually")
			s, err := readPassword()
			if err != nil {
				fmt.Printf("Error reading beaker password: %s\n", err.Error())
				return
			}
			config.Monitor["beaker_password"] = s
		}
	}

	debugPrint(log.Printf, levelWarning, "Provisioner Ver. %s.%s (%s) %s\n", Version, Build, Hash, Dirty)
	go syslog_service(config.LogFile, config.SyslogPort)
	go TFTPHandler(config.TFTPDirectory)
	go HTTPHandler(config.TFTPDirectory, config.HTTPPort)

	ssh_init(10, 10)
	serialRouter := NewRouter(10)
	serialRouter.Router()
	serialRouter.AttachAt(0, SrcMachine)

	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialRouter.In[0], serialRouter.Out[0])
	go SSHHandler(config.SSHSerTun, "tunnel", serialRouter, false)


	monitorRouter := NewRouter(10)
	monitorRouter.Router()
	monitorRouter.AttachAt(0, SrcMachine)

	m := MonitorInit(monitorRouter.In[0], monitorRouter.Out[0], config.Monitor, serialRouter,  "> ", 10, 20)
	go m.doMonitor()
	go SSHHandler(config.SSHMon, "monitor", monitorRouter, true)
	if config.Calendar.Enable {
		debugPrint(log.Printf, levelInfo, "Calendar activated\n")
		go calendarPoller(config.Calendar.Credfn)
	}
	select {}
}
