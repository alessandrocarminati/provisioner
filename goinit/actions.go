package main

import (
	"errors"
	"syscall"
	"net/url"
	"path/filepath"
	"time"
	"strconv"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	logbuf "pippo.com/goinit/logbuf"
)
type actionFuncs func(msgs chan string, config map[string]string)error

//var actions map[string] actionFuncs

func flashRootfs(msgs chan string, config map[string]string) error {
	var arg1, arg2 string
	var ok	bool

	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "check arguments")
	arg1, ok = config["pr.actionArg1"]
	if !ok {
		return errors.New("flashRootfs no Arg1: Arg1=<Url rootfs>, Arg2=dest device")
	}
	arg2, ok = config["pr.actionArg2"]
	if !ok {
		return errors.New("flashRootfs no Arg2: Arg1=<Url rootfs>, Arg2=dest device")
	}
	if !checkurl(msgs, arg1) {
		return errors.New("Arg1 must be a valid http url")
	}
	if !blockDeviceExists(msgs, arg2) {
		return errors.New("Arg2 must be a valid device in this board")
	}

	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Action successfully completed")
	return nil
}

func checkurl(msgs chan string, input string) bool {
	_, err := url.ParseRequestURI(input)

	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "checking %s", input)
	if err != nil {
		return false
	}

	u, err := url.Parse(input)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}

	return true
}

func listBlockDevices(msgs chan string) ([]string, error) {
	var blockDevices []string

	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "Start")
	blockDir := "/sys/block"
	files, err := ioutil.ReadDir(blockDir)
	if err != nil {
		return nil, err
	}
	fmt.Println(blockDir)
	for _, file := range files {
		sizePath := filepath.Join(blockDir, file.Name(), "size")
		fmt.Println(sizePath)
		_, err := os.Stat(sizePath)
		if err == nil {
			blockDevices = append(blockDevices, file.Name())
		}
	}
	msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "end, found %d device(s)", len(blockDevices))

	return blockDevices, nil
}

func blockDeviceExists(msgs chan string, dev string) bool {
	blockDir := "/sys/block"

	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "verify %s", dev)
	deviceName := strings.TrimPrefix(dev, "/dev/")
	var matchingSubstrings []string
	files, err := filepath.Glob(filepath.Join(blockDir, "*"))
	if err != nil {
		return false
	}

	for _, file := range files {
		basename := filepath.Base(file)
		if strings.Contains(deviceName, basename) {
			matchingSubstrings = append(matchingSubstrings, basename)
			msgs <- logbuf.LogSprintf(logbuf.LevelDebug, "root device %s found", basename)

		}
	}

	for _, substring := range matchingSubstrings {
		devicePath := filepath.Join(blockDir, substring, deviceName)
		_, err := os.Stat(devicePath)
		if err == nil {
			msgs <- logbuf.LogSprintf(logbuf.LevelDebug, "device %s found", devicePath)
			return true
		}
	}
	msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "device %s not found :(", dev)
	return false
}

func resetSystem(msgs chan string, config map[string]string) error {
//	LINUX_REBOOT_MAGIC1 := 0xfee1dead
//	LINUX_REBOOT_MAGIC2 := 672274793
//	LINUX_REBOOT_CMD_RESTART := 0x1234567

//	if err := syscall.Reboot(LINUX_REBOOT_MAGIC1, LINUX_REBOOT_MAGIC2, LINUX_REBOOT_CMD_RESTART, nil); err != nil {
//		return err
//	}
	var arg1 string
	var ok bool

        msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "check arguments")
        arg1, ok = config["pr.actionArg1"]
        if !ok {
		arg1, ok = config["pr.reboot"]
		if !ok {
	                return errors.New("reboot no time specified")
		}
        }

	n , err := strconv.Atoi(arg1)
	if err != nil{
		return errors.New("Arg1 must be a valid number")
		}
	msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "Restart device")
	time.Sleep(time.Duration(n) * time.Second)

	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART); err != nil {
		return err
	}

	return nil
}

func initActions()map[string]actionFuncs{
	actions:=make(map[string] actionFuncs,30)
	actions["flashRootfs"]=flashRootfs
	actions["reboot"]=resetSystem
	return actions
}


