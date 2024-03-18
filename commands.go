package main

import (
        "log"
        "strconv"
        "fmt"
        "sort"
)

type CommandFunction func(input string) string


type Command struct {
        Name        string
        HelpText    string
        Handler     CommandFunction
}

var commands  map[string] Command


func command_init(){
	var m Command

	log.Println("Initialyzing monitor commands struct")
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
	return "can't exit\r\n"
}
func tterm(input string) string {
	out:= "can't exit"
	if c, ok := sshChannels["tunnel"]; ok {
		(*c).Close()
		out="done!"
	}
	return out + "\r\n"
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
	out:="user not found!"
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
	return "Not Implemented Yet :("+ "\r\n"
}

func ton(input string) string{
	res:="Error config problem"
	configOK := true
	pdu_type, ok := monitorConfig["pdu_type"]
	if ok {
		switch pdu_type {
		case "tasmota":
			tasmota_host, ok := monitorConfig["tasmota_host"]
			if ok {
				res="on command sent"
				err := TasmotaSetState(tasmota_host, "ON")
				if err != nil {
					res=fmt.Sprintf("Error setting Tasmota device: %s", err.Error())
				}
			}
		case "snmp":
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
		default:
			res="unknown PDU type"
		}
	}
	return res + "\r\n"
}

func toff(input string) string{
	res:="Error config problem"
	configOK := true
	pdu_type, ok := monitorConfig["pdu_type"]
	if ok {
		switch pdu_type {
		case "tasmota":
			tasmota_host, ok := monitorConfig["tasmota_host"]
			if ok {
				res="off command sent"
				err := TasmotaSetState(tasmota_host, "OFF")
				if err != nil {
					res=fmt.Sprintf("Error setting Tasmota device: %s", err.Error())
				}
			}
		case "snmp":
			oid, ok := monitorConfig["snmp_pdu_ctrl_oid"]
			configOK = configOK && ok
			host, ok := monitorConfig["snmp_pdu_ctrl_host"]
			configOK = configOK && ok
			user, ok := monitorConfig["snmp_pdu_ctrl_user"]
			configOK = configOK && ok
			offValue, ok := monitorConfig["snmp_pdu_ctrl_on_val"]
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
		default:
			res="unknown PDU type"
		}
	}
	return res + "\r\n"
}
/*
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
	return res + "\r\n"
}
*/
func echoCmd(input string) string{

	log.Printf("echoCmd arg'%s'\n", input)
	if len(input) == 0 {
		return "error"
	}
	return input + "\r\n"
}

