package main

import (
        "log"
        "fmt"
        "sort"
	"strconv"
)

type CommandFunction func(input string) string
type FenceFuncs func(string) error


type Command struct {
        Name        string
        HelpText    string
        Handler     CommandFunction
}

var commands  map[string] Command

var fences map[string] FenceFuncs


func command_init(){
	var m Command

	debugPrint(log.Printf, levelInfo, "Initialyzing monitor commands struct")
	commands = make(map[string]Command, 20)

	m=Command{
		Name: "echo",
		HelpText: "echoes back the argument",
		Handler: echoCmd,
	}
	commands["echo"]=m
	m=Command{
		Name: "help",
		HelpText: "this text",
		Handler: help,
	}
	commands["help"]=m
	m=Command{
		Name: "?",
		HelpText: "this text",
		Handler: help,
	}
	commands["?"]=m
	m=Command{
		Name: "ton",
		HelpText: "command PDU using snmp to turn on the board",
		Handler: ton,
	}
	commands["ton"]=m
	m=Command{
		Name: "toff",
		HelpText: "command PDU using snmp to turn off the board",
		Handler: toff,
	}
	commands["toff"]=m
	m=Command{
		Name: "ulist",
		HelpText: "list user state for tunnel",
		Handler: listUser,
	}
	commands["ulist"]=m
	m=Command{
		Name: "enuser",
		HelpText: "enable user for tunnel",
		Handler: enuser,
	}
	commands["enuser"]=m
	m=Command{
		Name: "exit",
		HelpText: "exit this shell",
		Handler: exit,
	}
	commands["exit"]=m
	m=Command{
		Name: "tterm",
		HelpText: "terminate serial tunnel connection",
		Handler: tterm,
	}
	commands["tterm"]=m
}

func exit(input string) string {

	debugPrint(log.Printf, levelInfo, "exit command requested")

	ret :=""
	if len(input) == 0 {
		ret = "Available sessions:\r\n"
		for i, item  := range sshChannelsMonitor {
			if item != nil {
				ret = ret +fmt.Sprintf(" %d", i)
			}
		}
		return ret + "\r\n"
	}
	ret = "invalid argument\r\n"
	n, err := strconv.Atoi(input)
	if err != nil {
		return ret
	}
	c:= sshChannelsMonitor[n]
	if c != nil {
		(*c).Close()
		sshChannelsMonitor[n] = nil
		return "\r\n"
	}
	return ret
}
func tterm(input string) string {
	debugPrint(log.Printf, levelInfo, "tterm command requested")

	ret :=""
	if len(input) == 0 {
		ret = "Available sessions:\r\n"
		for i, item  := range sshChannelsSerial {
			if item != nil {
				ret = ret +fmt.Sprintf(" %d", i)
			}
		}
		return ret + "\r\n"
	}
	ret = "invalid argument\r\n"
	n, err := strconv.Atoi(input)
	if err != nil {
		return ret
	}
	c:= sshChannelsSerial[n]
	if c != nil {
		(*c).Close()
		sshChannelsSerial[n]=nil
		return "\r\n"
	}
	return ret
}
func listUser(input string) string {
	var out string

	debugPrint(log.Printf, levelInfo, "listUser command requested")
	for _, item := range GenAuth {
		if item.service == "tunnel" {
			out = out + fmt.Sprintf("\t%s -> %t\n\r", item.name, item.state)
		}
	}
	return out
}

func enuser(input string) string {
	out:="user not found!"
	debugPrint(log.Printf, levelInfo, "enuser command requested")
	if len(input) == 0 {
		out = "Error: enuser <user>\n\rHint: user corresponds to the ssh pubkey comment."
	} else {
		for i, item := range GenAuth {
			if item.service == "tunnel" {
                	        if item.name == input {
					GenAuth[i].state = true
					out="state updated"
				}
        	        }
	        }
	}
        return out + "\r\n"
}

func help(input string) string{
	out:=""
	debugPrint(log.Printf, levelInfo, "help command requested")
	list := make([]string, 0, len(commands))

	for k := range commands {
		list = append(list, k)
	}
	sort.Strings(list)

	for _, item := range list {
		out = out + fmt.Sprintf("\t%s:\t%s\n\r", commands[item].Name, commands[item].HelpText)
	}
	return out
}

func dummyCmd(input string) string{
	debugPrint(log.Printf, levelInfo, "dummy command requested")
	return "Not Implemented Yet :("+ "\r\n"
}


func FenceSwitch(state string) string{
	var res string

	pdu_type, ok := monitorConfig["pdu_type"]
	if ok {
		err := fences[pdu_type](state)
		if err != nil {
			res=err.Error()
			return res
		}
		return "Command sent! It may take up to 10 seconds.\r\n"
	}
	return "unknown PDU type\r\n"
}


func ton(input string) string{
	debugPrint(log.Printf, levelInfo, "ton command requested")
	return FenceSwitch("ON")
}

func toff(input string) string{
	debugPrint(log.Printf, levelInfo, "toff command requested")
	return FenceSwitch("OFF")
}

func echoCmd(input string) string{

	debugPrint(log.Printf, levelInfo, "echo command requested")
	log.Printf("echoCmd arg'%s'\n", input)
	if len(input) == 0 {
		return "error"
	}
	return input + "\r\n"
}
