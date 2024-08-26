package main

import (
        "log"
	"os"
	"time"
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
	gwScr          []*ScriptGwData
}

var log_serial_in_progress bool


func command_init(monitor *MonCtx, maxFences, maxScrSess int) (*CmdCtx) {

	debugPrint(log.Printf, levelInfo, "Initialyzing monitor commands struct")
	fences := make(map[string]FenceFuncs, maxFences)
	gws := make([]*ScriptGwData, maxScrSess)
	commands := make(map[string]Command, 20)
	c := &CmdCtx{
                monitor:   monitor,
                commands:  commands,
                fences:    fences,
		gwScr:     gws,
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
	c.commands["exec_assm"]=Command{
		Name: "exec_assm",
		HelpText: "Load and executes the specified assm script",
		Handler: c.exec_assm,
	}
	c.commands["exec_scr"]=Command{
		Name: "exec_scr",
		HelpText: "Load and executes the specified script",
		Handler: c.exec_scr,
	}

	c.commands["exec_state"]=Command{
		Name: "exec_state",
		HelpText: "returns the state of the specified script",
		Handler: c.exec_state,
	}

	c.commands["log_serial"]=Command{
		Name: "log_serial",
		HelpText: "copies in a file ser.log all sent and received from the serial. Note: overwrites previous.",
		Handler: c.log_serial,
	}

	c.commands["log_serial_stop"]=Command{
		Name: "log_serial_stop",
		HelpText: "Requires serila log subsystem to stop.",
		Handler: c.log_serial_stop,
	}

	return c
}

func (c *CmdCtx) exec_state(input string) string {
	debugPrint(log.Printf, levelInfo, "script command state")
	pos, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Sprintf("Argument error: %s\r\n", err.Error())
	}
	if c.gwScr[pos] == nil {
		return fmt.Sprintf("The position %d is not available:\r\n", pos)
	}
	return fmt.Sprintf("Script %d is in %s state\r\n", pos, c.gwScr[pos].GetState() )
}

func (c *CmdCtx) exec_scr(input string) string {
	debugPrint(log.Printf, levelInfo, "script command requested")

	args := strings.Split(input, " ")

	if len(args)!=3 {
		return fmt.Sprintf("exec_src <script_path> <term_type> <slot>\r\n")
	}
	ttype:=UndefinedTerm
	switch args[1] {
	case "line":
		ttype=LineOriented
	case "char":
		ttype=CharOriented
	default:
		return fmt.Sprintf("Unknown terminal type: %s\r\n")
	}

	pos, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Sprintf("Argument error: %s\r\n", err.Error())
	}
	if c.gwScr[pos] != nil {
		return fmt.Sprintf("The position %d is not available\r\n", pos)
	}

	n, err := (*(*c).monitor).router.GetFreePos()
	if err != nil {
		return fmt.Sprintf("no available channels: %s\r\n", err.Error())
	}
	(*(*c).monitor).router.AttachAt(n, SrcHuman)

	c.gwScr[pos] = ScriptGwInit(args[0], ttype, (*(*c).monitor).router.In[n], (*(*c).monitor).router.Out[n])

	go func(c *CmdCtx, pos int){
		defer (*(*c).monitor).router.DetachAt(n)
		c.gwScr[pos].ScriptGwExec()
		debugPrint(log.Printf, levelWarning, "execution terminated: %d", c.gwScr[pos].state)
	}(c, pos)
	return "script is processing text from serial\r\n"
}

func (c *CmdCtx) exec_assm(input string) string {
	debugPrint(log.Printf, levelInfo, "script command requested")
	n, err := (*(*c).monitor).router.GetFreePos()
	if err != nil {
		return fmt.Sprintf("no available channels: %s\r\n", err.Error())
	}
	(*(*c).monitor).router.AttachAt(n, SrcHuman)
	if !strings.HasSuffix(input, ".assm") {
		return "unknown script type\r\n"
	}
	ex, err := einit(input,  (*(*c).monitor).router.In[n], (*(*c).monitor).router.Out[n])
	if err != nil {
		(*(*c).monitor).router.DetachAt(n)
		return fmt.Sprintf("Syntax error: %s\r\n", err.Error())
	}
	go func(c *CmdCtx){
		defer (*(*c).monitor).router.DetachAt(n)
		err = ex.Execute(500)
		if err != nil {
			debugPrint(log.Printf, levelError, err.Error())
			return
		}
		debugPrint(log.Printf, levelWarning, "execution terminated")
	}(c)
	return "script is processing text from serial\r\n"
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
func (c *CmdCtx) log_serial_stop(input string) string{
	log_serial_in_progress=false
	return fmt.Sprintf("Sent request to stop logging.\r\n")
}

func (c *CmdCtx) log_serial(input string) string{

	if log_serial_in_progress {
		return fmt.Sprintf("Already in progress\r\n")
	}
	if (input == "") {
		return fmt.Sprintf("no input file given\r\n")
	}
	items := strings.Split(input, " ")
	if (len(items)!=1) {
		return fmt.Sprintf("Syntax error. Command has only an argument. it is the log file name.\r\n")
	}
	debugPrint(log.Printf, levelInfo, "log_serial command requested")
	n, err := (*(*c).monitor).router.GetFreePos()
	if err != nil {
		 return fmt.Sprintf("no available channels: %s\r\n", err.Error())
	}
	(*(*c).monitor).router.AttachAt(n, SrcHuman)
	log_serial_in_progress=true
	go func(c *CmdCtx){
		var buffer []byte
		defer (*(*c).monitor).router.DetachAt(n)

		f, err := os.Create(input)
		if err!=nil {
			debugPrint(log.Printf, levelError, "Can't create file %s: %s", input, err.Error())
		}
		defer f.Close()

		debugPrint(log.Printf, levelInfo, "Goroutine started")
		inStrChan := (*(*c).monitor).router.In[n]

		go func(){
			for log_serial_in_progress {
				if len(buffer) > 0 {
					debugPrint(log.Printf, levelDebug, "Writing buffer in the file '%s'", buffer)
					n2, err := f.Write(buffer)
					if err != nil {
						debugPrint(log.Printf, levelError, "Cant write log file: %s", err.Error())
					}
					debugPrint(log.Printf, levelDebug, "Wrote %d bytes", n2)
					f.Sync()
					buffer = nil
				}
				time.Sleep(5 * time.Second)
			}
		}()
		for log_serial_in_progress {
			select {
			case b, ok := <-inStrChan:
				if !ok {
					debugPrint(log.Printf, levelError, "can't read from channel, write buffer and end the goroutine")
					if len(buffer) > 0 {
						f.Write(buffer)
					}
					return
				}
				buffer = append(buffer, b)
			}
		}
	}(c)
        return fmt.Sprintf("Logging on '%s'\r\n", input)
}
