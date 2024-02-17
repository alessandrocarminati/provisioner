package main

import (
	"log"
	"strconv"
	"fmt"
	"sync"
        "strings"
)

type CommandFunction func(input string) string


type Command struct {
        Name        string
        HelpText    string
        Handler     CommandFunction
}
var monitorConfig map[string] string

var commands = map[string] Command{
	"echo": {
		Name: "echo",
		HelpText: "echoes back the argument",
		Handler: echoCmd,
	},
	"help": {
		Name: "help",
		HelpText: "this text",
		Handler: dummyCmd,
	},
	"ton": {
		Name: "ton",
		HelpText: "command PDU using snmp to turn on the board",
		Handler: ton,
	},
	"toff": {
		Name: "toff",
		HelpText: "command PDU using snmp to turn off the board",
		Handler: toff,
	},

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
//				log.Printf("input@1 = '%s'", input)
				input = sanitize(left(input, 13))
//				log.Printf("input@2 = '%s'", left(input, ' '))
				if _, ok := commands[string(left(input, 32))]; ok {
					input = sanitize(input)
					output = []byte("\n\r" + commands[string(left(input, 32))].Handler(string(input)) + "\n\r> ")
				} else {
					output = []byte("\n\rError!\n\r> ")
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
