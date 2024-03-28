package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os/signal"
	"bufio"
	"syscall"
	"os"
	"strings"
	"strconv"
	"golang.org/x/exp/slices"
	logbuf "pippo.com/goinit/logbuf"
)
var startmsg chan bool
var logLevel int

func isSymbolicLink(path string, msgs chan string) bool {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink
}

func mount(device, target string, msgs chan string){
	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "Mount %s", device)

	if err := os.Mkdir(target, os.ModePerm); err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelError, "Error creating procfs: %s", err.Error())
		os.Exit(0xfff2) 
		}
	if err := unix.Mount(device, target, device, 0, ""); err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelError, "Error mounting %s: %s", device, err.Error())
		os.Exit(0xfff3) 
	}
	msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "%s mounted successfully at %s", device, target)
}


func fetchConfig(s string) map[string] string{

	res := make(map[string] string, 50)
	tmp := strings.Split(s, " ")
	for _, item := range tmp {
		if strings.HasPrefix(item, "pr.") {
			tmp2 := strings.Split(item, "=")
			res[tmp2[0]]=tmp2[1]
		}
	}
	return res
}

func main() {
	var config map[string] string

	msgs := make(chan string, 300)
	startmsg = make(chan bool ,1)

	actions:=initActions()

	msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "Starting Init")
	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "Checking pid")
	if os.Getpid() != 1 {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "This is not pid 1")
	}
	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "Mounting file systems")
	mount("proc", "/proc", msgs)
	mount("sysfs", "/sys", msgs)

	file, err := os.Open("/proc/cmdline")
	if err != nil {
		os.Exit(0xfff2)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		s:=scanner.Text()
		msgs <- logbuf.LogSprintf(logbuf.LevelDebug, s)
		config = fetchConfig(s)
	}

	msgs <- logbuf.LogSprintf(logbuf.LevelDebug, "Variables from cmdline")
	for key, value := range config {
		msgs <- logbuf.LogSprintf(logbuf.LevelDebug, "%s=%s", key, value)
	}
	if err := scanner.Err(); err != nil {
		os.Exit(0xfff1)
	}

	sysIfs := listdev(msgs)

	c:= make(chan  int)

	sllevel, ok := config["pr.debuglevel"]
	if ok {
		lev , err := strconv.Atoi(sllevel)
		if err!=nil {
			msgs <- logbuf.LogSprintf(logbuf.LevelError, "loglevel %s is not supported", sllevel)
		} else {
			msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "loglevel is set to %s", sllevel)
			logLevel = lev
		}
	}

	ifName, ok := config["pr.ifname"]
	if ok {
		if slices.Contains(sysIfs, ifName) {
			go dhcpFetch(ifName, c, msgs)
			config["hasif"]="ok"
		}
	}
	go syslogSender(msgs, config, logLevel)


	action, ok := config["pr.action"]
	if ok {
		acfun, ok := actions[action]
		if ok {
			err:= acfun(msgs, config)
			if err!= nil {
				msgs <- logbuf.LogSprintf(logbuf.LevelError, "%s action error: %s", action, err.Error())
			}else{
				msgs <- logbuf.LogSprintf(logbuf.LevelDebug, "%s action success", action)
			}
		} else {
			msgs <- logbuf.LogSprintf(logbuf.LevelError, "requested action %s is unknown", action)
		}
	} else {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "No action!!!")
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		sig := <-sigs
		fmt.Println()
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Received %s, exiting...", sig)
		done <- true
	}()


/*	devices, err := listBlockDevices()
	if err != nil {
//		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Error:", err)
		panic(err)
	}

	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Block Devices:")
	for _, device := range devices {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, device)
	}*/



	fmt.Println("Press Ctrl-C to exit...")
	<-done
	fmt.Println("Done")
	os.Exit(0xfff0)
}
