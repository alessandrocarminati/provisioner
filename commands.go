package main

import (
        "log"
        "fmt"
        "sort"
	"strconv"
	"strings"
)

type CommandFunction func(string) string
type FenceFuncs func(string) error


type Command struct {
        Name        string
        HelpText    string
        Handler     CommandFunction
}

type CmdCtx struct {
	monitor        *MonCtx
	commands       map[string] Command
	fences         map[string] FenceFuncs
}

func command_init(monitor *MonCtx, fences map[string] FenceFuncs) (*CmdCtx) {

	debugPrint(log.Printf, levelInfo, "Initialyzing monitor commands struct")
	commands := make(map[string]Command, 20)
	c := &CmdCtx{
                monitor: monitor,
                commands: commands,
                fences: fences,
        }

	c.commands["echo"]=Command{
		Name: "echo",
		HelpText: "echoes back the argument",
		Handler: c.echoCmd,
	}
	c.commands["help"]=Command{
		Name: "help",
		HelpText: "this text",
		Handler: c.help,
	}
	c.commands["?"]=Command{
		Name: "?",
		HelpText: "this text",
		Handler: c.help,
	}
	c.commands["ton"]=Command{
		Name: "ton",
		HelpText: "command PDU using snmp to turn on the board",
		Handler: c.ton,
	}
	c.commands["toff"]=Command{
		Name: "toff",
		HelpText: "command PDU using snmp to turn off the board",
		Handler: c.toff,
	}
	c.commands["ulist"]=Command{
		Name: "ulist",
		HelpText: "list user state for tunnel",
		Handler: c.listUser,
	}
	c.commands["enuser"]=Command{
		Name: "enuser",
		HelpText: "enable user for tunnel",
		Handler: c.enuser,
	}
	c.commands["exit"]=Command{
		Name: "exit",
		HelpText: "exit this shell",
		Handler: c.exit,
	}
	c.commands["tterm"]=Command{
		Name: "tterm",
		HelpText: "terminate serial tunnel connection",
		Handler: c.tterm,
	}
	return c
}

func (c *CmdCtx) exit(input string) string {

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
	chn := sshChannelsMonitor[n]
	if chn != nil {
		(*chn).Close()
		sshChannelsMonitor[n] = nil
		return "\r\n"
	}
	return ret
}
func (c *CmdCtx) tterm(input string) string {
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
	chn := sshChannelsSerial[n]
	if chn != nil {
		(*chn).Close()
		sshChannelsSerial[n]=nil
		return "\r\n"
	}
	return ret
}
func (c *CmdCtx) listUser(input string) string {
	var out string

	debugPrint(log.Printf, levelInfo, "listUser command requested")
	for _, item := range GenAuth {
		if item.service == "tunnel" {
			out = out + fmt.Sprintf("  %-40s %t\n\r", item.name + " ->", item.state)
		}
	}
	return out
}

func (c *CmdCtx) enuser(input string) string {
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

func (c *CmdCtx) help(input string) string{
	out:=""
	debugPrint(log.Printf, levelInfo, "help command requested")
	list := make([]string, 0, len(c.commands))

	for k := range c.commands {
		list = append(list, k)
	}
	sort.Strings(list)

	for _, item := range list {
		out = out + fmt.Sprintf("  %-20s %s\n\r", c.commands[item].Name+" :", c.commands[item].HelpText)
	}
	return out
}

func (c *CmdCtx) dummyCmd(input string) string{
	debugPrint(log.Printf, levelInfo, "dummy command requested")
	return "Not Implemented Yet :("+ "\r\n"
}


func (c *CmdCtx) FenceSwitch(state string) string{
	var res string

	pdu_type, ok := (*(*c).monitor).monitorConfig["pdu_type"]
	if ok {
		err := c.fences[pdu_type](state)
		if err != nil {
			res=err.Error()
			return res
		}
		return "Command sent! It may take up to 10 seconds.\r\n"
	}
	return "unknown PDU type\r\n"
}


func (c *CmdCtx) ton(input string) string{
	debugPrint(log.Printf, levelInfo, "ton command requested")
	return c.FenceSwitch("ON")
}

func (c *CmdCtx) toff(input string) string{
	debugPrint(log.Printf, levelInfo, "toff command requested")
	return c.FenceSwitch("OFF")
}

func (c *CmdCtx) echoCmd(input string) string{

	debugPrint(log.Printf, levelInfo, "echo command requested")
	log.Printf("echoCmd arg'%s'\n", input)
	if len(input) == 0 {
		return "error"
	}
	return input + "\r\n"
}
