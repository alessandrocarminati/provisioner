package main

import (
	"log"
	"strconv"
	"fmt"
	"sync"
        "strings"
	"sort"
)

type CommandFunction func(input string) string


type Command struct {
        Name        string
        HelpText    string
        Handler     CommandFunction
}
var monitorConfig map[string] string

var commands  map[string] Command

func command_init(){
	var m Command

	log.Println("initialyze commands struct")
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
	if c, ok := sshChannels["monitor"]; ok {
		(*c).Close()
	}
	return "can't exit"
}
func tterm(input string) string {
	out:= "can't exit"
	if c, ok := sshChannels["tunnel"]; ok {
		(*c).Close()
		out="done!"
	}
	return out
}
func listUser(input string) string {
	var out string

	for _, item := range GenAuth {
		if item.service == "tunnel" {
			out = out + fmt.Sprintf("\t%s -> %t\n\r", item.name, item.state)
		}
	}
	return out
}

func enuser(input string) string {
	sarg := strings.Index(input, " ")
	out:="user not found!"
	if sarg == -1 {
		return "Error: enuser <user>\n\rHint: user corresponds to the ssh pubkey comment."
	}
	for i, item := range GenAuth {
		if item.service == "tunnel" {
                        if item.name == input[sarg+1:] {
				GenAuth[i].state = true
				out="state updated"
			}
                }
        }
        return out
}

func help(input string) string{
	out:=""
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
	return "Not Implemented Yet :("
}

func ton(input string) string{
	res:="Error config problem"
	configOK := true
	oid, ok := monitorConfig["snmp_pdu_ctrl_oid"]
	configOK = configOK && ok
	host, ok := monitorConfig["snmp_pdu_ctrl_host"]
	configOK = configOK && ok
	user, ok := monitorConfig["snmp_pdu_ctrl_user"]
	configOK = configOK && ok
	onValue, ok := monitorConfig["snmp_pdu_ctrl_on_val"]
	configOK = configOK && ok
	val, err := strconv.Atoi(onValue)
	if err != nil {
		configOK=false
	}
	if configOK {
		res="on command sent"
		err := snmpSetv3unsec(oid, val, host, user)
		if err != nil {
			res=fmt.Sprintf("Error setting SNMP: %s", err.Error())
		}
	}
	return res
}
func toff(input string) string{
	res:="Error config problem"
	configOK := true
	oid, ok := monitorConfig["snmp_pdu_ctrl_oid"]
	configOK = configOK && ok
	host, ok := monitorConfig["snmp_pdu_ctrl_host"]
	configOK = configOK && ok
	user, ok := monitorConfig["snmp_pdu_ctrl_user"]
	configOK = configOK && ok
	offValue, ok := monitorConfig["snmp_pdu_ctrl_off_val"]
	configOK = configOK && ok
	val, err := strconv.Atoi(offValue)
	if err != nil {
		configOK=false
	}
	if configOK {
		res="off command sent"
		err := snmpSetv3unsec(oid, val, host, user)
		if err != nil {
			res=fmt.Sprintf("Error setting SNMP: %s", err.Error())
		}
	}
	return res
}

func echoCmd(input string) string{
	i := strings.Index(input, " ")

	if i == -1 {
		return "error"
	}
	return input[i+1:]
}


func Monitor(monitorIn <-chan []byte, monitorOut chan<- []byte, monConfig map[string] string) {
	var wg sync.WaitGroup
	var outputFlag bool = false
	var output []byte
	var input []byte

	monitorConfig=monConfig
	output = []byte("\n\r> ")
	outputFlag = true
	command_init()

//		log.Println(commands)

	wg.Add(1)
	go func() {
		for {
			if outputFlag {
				monitorOut <- output
				outputFlag = false
			}
		}
	}()

	wg.Add(1)
	go func() {
		for {
			buff := <- monitorIn
			output = replaceByte(buff, 127,'\b')
			outputFlag = true
			input = sanitize(append(input, replaceByte(buff, 127,'\b')...))
			if cmdEnter(input) {
				input = sanitize(left(input, 13))
				key := string(left(input, 32))
				if key!="" {
					if _, ok := commands[key]; ok {
						input = sanitize(input)
						output = []byte("\n\r" + commands[key].Handler(string(input)) + "\n\r> ")
					} else {
						output = []byte("\n\rError!\n\r> ")
					}
				} else {
					output = []byte("\n\r> ")
				}
				input = input[:0]
				outputFlag = true
			}
		}
	}()
	wg.Wait()
}

func findRunePosition(str []byte, target byte) int {
	for i, r := range str {
		if r == target {
			return i
		}
	}
	return -1
}
func left(input []byte, c byte) []byte {
    newlineIndex := findRunePosition(input, c)
    if newlineIndex != -1 {
        return input[:newlineIndex]
    }
    return input
}


func cmdEnter(input []byte) bool{
	for _, c  := range input {
		if c==13{
			return true
		}
	}
	return false
}
func sanitize(input []byte) []byte {
	var output []byte

	for _, char := range input {
		if char == '\b' {
			log.Println("\b")
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		} else {
			output=append(output,char)
		}
	}

	return output
}

func replaceByte(input []byte, aa, bb byte) []byte {
	output := make([]byte, len(input))

	for i, b := range input {
		if b == aa {
			output[i] = bb
		} else {
			output[i] = b
		}
	}

	return output
}
