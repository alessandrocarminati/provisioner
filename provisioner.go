package main
import (
	"context"
	"sync/atomic"
	"bufio"
	"time"
	"os"
	"log"
	"fmt"
	"golang.org/x/term"
	"syscall"
	"strings"
	"errors"
	"strconv"
	"runtime"
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

func getMaxs(config Config) (int, int, error) {
	maxFenceTypesS, ok := config.Monitor["max_fence_types"]
	if !ok {
		return 0, 0, errors.New("config max_fence_types is missing")

	}
	maxFenceTypes, err := strconv.Atoi(maxFenceTypesS)
	if err != nil {
		return 0, 0, errors.New("config max_fence_types is wrong format")
	}
	maxScriptSessS, ok := config.Monitor["max_script_sess"]
	if !ok {
		return 0, 0, errors.New("config max_script_sess is missing")
	}
	maxScriptSess, err := strconv.Atoi(maxScriptSessS)
	if err != nil {
		return 0, 0, errors.New("config max_script_sess is wrong format")
	}
	return maxFenceTypes, maxScriptSess, nil
}

func noiseGenerator(ctx context.Context) {
	var counter uint64
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			atomic.AddUint64(&counter, 1)
			runtime.Gosched()
		}
	}
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
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
	maxFenceTypes, maxScriptSess, err :=getMaxs(config)
	if err!= nil {
		fmt.Printf("Config error: %s\n", err.Error())
		return
	}

	debugPrint(log.Printf, levelWarning, "Provisioner Ver. %s.%s (%s) %s\n", Version, Build, Hash, Dirty)
	go syslog_service(config.NetServices.LogFile, config.NetServices.SyslogPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go noiseGenerator(ctx)

	go TFTPHandler(config.NetServices.TFTPDirectory)
	go HTTPHandler(config.NetServices.TFTPDirectory, config.NetServices.HTTPPort)

	ssh_init(config.Router.MonitorChans, config.Router.SerialChans)
	serialRouter := NewRouter(config.Router.SerialChans)
	serialRouter.Router()
	serialRouter.AttachAt(config.Router.SerialMain, SrcMachine)

	go SerialHandler(config.SerialConfig.Port, config.SerialConfig.BaudRate, serialRouter.In[0], serialRouter.Out[0])
	go SSHHandler(config.SSHSerTun, "tunnel", serialRouter, false)
	go StartBoardWatcher(serialRouter)

	monitorRouter := NewRouter(config.Router.MonitorChans)
	monitorRouter.Router()
	monitorRouter.AttachAt(config.Router.MonitorMain, SrcMachine)

	m := MonitorInit(monitorRouter.In[config.Router.MonitorMain], monitorRouter.Out[config.Router.MonitorMain], config.Monitor, serialRouter,  "> ", maxFenceTypes, maxScriptSess)
	go m.doMonitor()
	go SSHHandler(config.SSHMon, "monitor", monitorRouter, true)
	if config.Calendar.Enable {
		debugPrint(log.Printf, levelInfo, "Calendar activated\n")
		go calendarPoller(config.Calendar.Credfn)
	}
	select {}
}
