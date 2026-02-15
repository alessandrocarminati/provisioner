package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CommandFunction func(string) string
type FenceFuncs func(string) error

type Command struct {
	Name     string
	HelpText string
	Handler  CommandFunction
}

type CmdCtx struct {
	monitor  *MonCtx
	commands map[string]Command
	fences   map[string]FenceFuncs
	gwScr    []*ScriptGwData
}

var log_serial_in_progress bool

func command_init(monitor *MonCtx, maxFences, maxScrSess int) *CmdCtx {

	debugPrint(log.Printf, levelInfo, "Initialyzing monitor commands struct")
	fences := make(map[string]FenceFuncs, maxFences)
	gws := make([]*ScriptGwData, maxScrSess)
	commands := make(map[string]Command, 20)
	c := &CmdCtx{
		monitor:  monitor,
		commands: commands,
		fences:   fences,
		gwScr:    gws,
	}

	c.commands["echo"] = Command{
		Name:     "echo",
		HelpText: "echoes back the argument",
		Handler:  c.echoCmd,
	}
	c.commands["help"] = Command{
		Name:     "help",
		HelpText: "this text",
		Handler:  c.help,
	}
	c.commands["?"] = Command{
		Name:     "?",
		HelpText: "this text",
		Handler:  c.help,
	}
	c.commands["ton"] = Command{
		Name:     "ton",
		HelpText: "command PDU using snmp to turn on the board",
		Handler:  c.ton,
	}
	c.commands["toff"] = Command{
		Name:     "toff",
		HelpText: "command PDU using snmp to turn off the board",
		Handler:  c.toff,
	}
	c.commands["ulist"] = Command{
		Name:     "ulist",
		HelpText: "list user state for tunnel",
		Handler:  c.listUser,
	}
	c.commands["enuser"] = Command{
		Name:     "enuser",
		HelpText: "enable user for tunnel",
		Handler:  c.enuser,
	}
	c.commands["exit"] = Command{
		Name:     "exit",
		HelpText: "exit this shell",
		Handler:  c.exit,
	}
	c.commands["tterm"] = Command{
		Name:     "tterm",
		HelpText: "terminate serial tunnel connection",
		Handler:  c.tterm,
	}
	c.commands["exec_assm"] = Command{
		Name:     "exec_assm",
		HelpText: "Load and executes the specified assm script",
		Handler:  c.exec_assm,
	}
	c.commands["exec_scr"] = Command{
		Name:     "exec_scr",
		HelpText: "Load and executes the specified script",
		Handler:  c.exec_scr,
	}

	c.commands["exec_state"] = Command{
		Name:     "exec_state",
		HelpText: "returns the state of the specified script",
		Handler:  c.exec_state,
	}

	c.commands["log_serial"] = Command{
		Name:     "log_serial",
		HelpText: "copies in a file ser.log all sent and received from the serial. Note: overwrites previous.",
		Handler:  c.log_serial,
	}

	c.commands["log_serial_stop"] = Command{
		Name:     "log_serial_stop",
		HelpText: "Requires serila log subsystem to stop.",
		Handler:  c.log_serial_stop,
	}
	c.commands["send_serial"] = Command{
		Name:     "send_serial",
		HelpText: "send file over serial: send_serial <file> <plain|gzip|xmodem_unix|xmodem_uboot> [dest_path]",
		Handler:  c.send_serial,
	}
	c.commands["board_stat"] = Command{
		Name:     "board_stat",
		HelpText: "report last goinit board stat; board must have printed PROVISIONER_MGMT_*",
		Handler:  c.board_stat,
	}
	c.commands["filter"] = Command{
		Name:     "filter",
		HelpText: "Filter commands: type 'filter help' for more info",
		Handler:  c.Filter,
	}
	c.commands["send_serial_deps"] = Command{
		Name:     "send_serial_deps",
		HelpText: "check remote deps for send_serial plain/gzip: stty, dd, base64, gzip, rm (or busybox).",
		Handler:  c.send_serial_deps,
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
	return fmt.Sprintf("Script %d is in %s state\r\n", pos, c.gwScr[pos].GetState())
}

func (c *CmdCtx) exec_scr(input string) string {
	debugPrint(log.Printf, levelInfo, "script command requested")

	args := strings.Split(input, " ")

	if len(args) != 3 {
		return fmt.Sprintf("exec_src <script_path> <term_type> <slot>\r\n")
	}
	ttype := UndefinedTerm
	switch args[1] {
	case "line":
		ttype = LineOriented
	case "char":
		ttype = CharOriented
	default:
		return fmt.Sprintf("Unknown terminal type: %s\r\n", args[1])
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

	go func(c *CmdCtx, pos int) {
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
	ex, err := einit(input, (*(*c).monitor).router.In[n], (*(*c).monitor).router.Out[n])
	if err != nil {
		(*(*c).monitor).router.DetachAt(n)
		return fmt.Sprintf("Syntax error: %s\r\n", err.Error())
	}
	go func(c *CmdCtx) {
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

	ret := ""
	if len(input) == 0 {
		ret = "Available sessions:\r\n"
		for i, item := range sshChannelsMonitor {
			if item != nil {
				ret = ret + fmt.Sprintf(" %d", i)
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

	ret := ""
	if len(input) == 0 {
		ret = "Available sessions:\r\n"
		for i, item := range sshChannelsSerial {
			if item != nil {
				ret = ret + fmt.Sprintf(" %d", i)
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
		sshChannelsSerial[n] = nil
		return "\r\n"
	}
	return ret
}
func (c *CmdCtx) listUser(input string) string {
	var out string

	debugPrint(log.Printf, levelInfo, "listUser command requested")
	for _, item := range GenAuth {
		if item.service == "tunnel" {
			out = out + fmt.Sprintf("  %-40s %t\n\r", item.name+" ->", item.state)
		}
	}
	return out
}

func (c *CmdCtx) enuser(input string) string {
	out := "user not found!"
	debugPrint(log.Printf, levelInfo, "enuser command requested")
	if len(input) == 0 {
		out = "Error: enuser <user>\n\rHint: user corresponds to the ssh pubkey comment."
	} else {
		for i, item := range GenAuth {
			if item.service == "tunnel" {
				if item.name == input {
					GenAuth[i].state = true
					out = "state updated"
				}
			}
		}
	}
	return out + "\r\n"
}

func (c *CmdCtx) help(input string) string {
	out := ""
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

func (c *CmdCtx) dummyCmd(input string) string {
	debugPrint(log.Printf, levelInfo, "dummy command requested")
	return "Not Implemented Yet :(" + "\r\n"
}

func (c *CmdCtx) FenceSwitch(state string) string {
	var res string

	pdu_type, ok := (*(*c).monitor).monitorConfig["pdu_type"]
	if ok {
		err := c.fences[pdu_type](state)
		if err != nil {
			res = err.Error()
			return res
		}
		return "Command sent! It may take up to 10 seconds.\r\n"
	}
	return "unknown PDU type\r\n"
}

func (c *CmdCtx) ton(input string) string {
	debugPrint(log.Printf, levelInfo, "ton command requested")
	return c.FenceSwitch("ON")
}

func (c *CmdCtx) toff(input string) string {
	debugPrint(log.Printf, levelInfo, "toff command requested")
	return c.FenceSwitch("OFF")
}

func (c *CmdCtx) echoCmd(input string) string {

	debugPrint(log.Printf, levelInfo, "echo command requested")
	log.Printf("echoCmd arg'%s'\n", input)
	if len(input) == 0 {
		return "error"
	}
	return input + "\r\n"
}
func (c *CmdCtx) log_serial_stop(input string) string {
	log_serial_in_progress = false
	return fmt.Sprintf("Sent request to stop logging.\r\n")
}

func (c *CmdCtx) send_serial(input string) string {
	parts := strings.SplitN(strings.TrimSpace(input), " ", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "send_serial <file> <plain|gzip|xmodem_unix|xmodem_uboot> [dest_path]\r\n"
	}
	localPath := parts[0]
	mode := strings.ToLower(parts[1])
	destPath := ""
	if len(parts) >= 3 {
		destPath = strings.TrimSpace(parts[2])
	}
	validModes := map[string]bool{"plain": true, "gzip": true, "xmodem_unix": true, "xmodem_uboot": true}
	if !validModes[mode] {
		return fmt.Sprintf("send_serial: invalid mode %q (use: plain, gzip, xmodem_unix, xmodem_uboot)\r\n", mode)
	}
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Sprintf("send_serial: %s\r\n", err.Error())
	}

	router := (*(*c).monitor).router
	n, err := router.GetFreePos()
	if err != nil {
		return fmt.Sprintf("send_serial: no free channel: %s\r\n", err.Error())
	}
	if err := router.AttachAt(n, SrcHuman); err != nil {
		return fmt.Sprintf("send_serial: attach: %s\r\n", err.Error())
	}
	defer router.DetachAt(n)

	serialIO := &SerialIO{In: router.In[n], Out: router.Out[n]}
	if err := SendFile(serialIO, localPath, mode, destPath); err != nil {
		return fmt.Sprintf("send_serial: %s\r\n", err.Error())
	}
	return "send_serial: done.\r\n"
}

func (c *CmdCtx) board_stat(input string) string {
	boardIP, boardIf, statJSON, lastTime, ok := GetBoardStat()
	if !ok {
		return "board_stat: no board IP or stat yet (watch serial for PROVISIONER_MGMT_IP= from goinit).\r\n"
	}
	var out strings.Builder
	out.WriteString(fmt.Sprintf("board %s (if %s) last=%s\r\n", boardIP, boardIf, lastTime.Format(time.RFC3339)))
	if len(statJSON) == 0 {
		out.WriteString("(no stat polled yet)\r\n")
		return out.String()
	}
	var pretty json.RawMessage
	if err := json.Unmarshal(statJSON, &pretty); err != nil {
		out.WriteString(string(statJSON))
		out.WriteString("\r\n")
		return out.String()
	}
	indented, _ := json.MarshalIndent(pretty, "", "  ")
	out.Write(indented)
	out.WriteString("\r\n")
	return out.String()
}

func (c *CmdCtx) send_serial_deps(input string) string {
	router := (*(*c).monitor).router
	n, err := router.GetFreePos()
	if err != nil {
		return fmt.Sprintf("send_serial_deps: no free channel: %s\r\n", err.Error())
	}
	if err := router.AttachAt(n, SrcHuman); err != nil {
		return fmt.Sprintf("send_serial_deps: attach: %s\r\n", err.Error())
	}
	defer router.DetachAt(n)
	serialIO := &SerialIO{In: router.In[n], Out: router.Out[n]}
	result, _ := CheckSerialDeps(serialIO)
	switch result {
	case "ok":
		return "send_serial_deps: ok (stty, dd, base64, gzip, rm available on board).\r\n"
	case "missing":
		return "send_serial_deps: missing (board needs stty, dd, base64, gzip, rm or busybox).\r\n"
	default:
		return "send_serial_deps: timeout (no reply; run on board: command -v stty dd base64 gzip rm).\r\n"
	}
}

func (c *CmdCtx) Filter(input string) string {
	router := (*(*c).monitor).router
	parts := strings.Fields(strings.TrimSpace(input))

	if len(parts) == 0 {
		return c.getFilterStatus(router)
	}

	cmd := parts[0]

	debugPrint(log.Printf, /**/levelDebug, "requested command %s\n", cmd)
	switch cmd {
	case "enable":
		return c.enableFilter(router)

	case "disable":
		return c.disableFilter(router)

	case "default":
		return c.resetToDefault(router)

	case "show":
		return c.showFilterRules(router)

	case "add":
		if len(parts) < 2 {
			return "error: add requires format and sequences\r\n" +
				   "usage: filter add ascii <received> [forwarded] [answered]\r\n" +
				   "	   filter add hex <received_hex> [forwarded_hex] [answered_hex]\r\n"
		}
		return c.addFilterRule(router, parts[1:])

	case "remove":
		if len(parts) < 2 {
			return "error: remove requires rule index (format: remove <index>)\r\n"
		}
		return c.removeFilterRule(router, parts[1])
	case "help":
		return "available: enable, disable, default, show, add, remove, help\r\n"+
			fmt.Sprintf("  %-20s %s\n\r", "enable :",  "Enables filters - Example: filter enable")+
			fmt.Sprintf("  %-20s %s\n\r", "disable :", "Disables filters - Example: filter disable")+
			fmt.Sprintf("  %-20s %s\n\r", "default :", "Restores filters default rules - Example: filter default")+
			fmt.Sprintf("  %-20s %s\n\r", "show :",    "Shows current filter state and rules - Example: filter show")+
			fmt.Sprintf("  %-20s %s\n\r", "add :",     "adds rules to filters")+
			fmt.Sprintf("  %-20s %s\n\r", " Example >"," filter add hex 48656c6c6f 48656c6c6f 576f726c64")+
			fmt.Sprintf("  %-20s %s\n\r", " Example >"," filter add ascii Hello Hello World")+
			fmt.Sprintf("  %-20s %s\n\r", "remove :",  "removes rules to filters - Example: filter remove 5")+
			fmt.Sprintf("  %-20s %s\n\r", "help :",    "this message")
	default:
		debugPrint(log.Printf, /**/levelWarning, "No such command %s\n", cmd)
		return c.getFilterStatus(router)
	}
}

func (c *CmdCtx) getFilterStatus(router *Router) string {
	router.FilterMu.RLock()
	enabled := router.Filter
	router.FilterMu.RUnlock()

	debugPrint(log.Printf, /**/levelDebug, "current state %t change to %t\n", enabled, !enabled)
	if enabled {
		return "filter on\r\n"
	}
	return "filter off\r\n"
}

func (c *CmdCtx) enableFilter(router *Router) string {
	router.FilterMu.Lock()
	defer router.FilterMu.Unlock()

	debugPrint(log.Printf, /**/levelDebug, "filter is going live\n")

	if router.outgoingFilter == nil {
		debugPrint(log.Printf, /**/levelDebug, "filter outgoing rules needs to be created\n")
		router.outgoingFilter = &StreamFilter{
			rules: copyFilterRule(defaultFilterRule),
		}
	}
	if router.incomingFilter == nil {
		debugPrint(log.Printf, /**/levelDebug, "filter incoming rules needs to be created\n")
		router.incomingFilter = &StreamFilter{
			rules: copyFilterRule(defaultFilterRule),
		}
	}

	router.Filter = true
	return "filter enabled\r\n"
}

func (c *CmdCtx) disableFilter(router *Router) string {
	router.FilterMu.Lock()
	router.Filter = false
	router.FilterMu.Unlock()

	debugPrint(log.Printf, /**/levelDebug, "filter is going down\n")
	return "filter disabled\r\n"
}

func (c *CmdCtx) resetToDefault(router *Router) string {
	router.FilterMu.Lock()
	defer router.FilterMu.Unlock()

	debugPrint(log.Printf, /**/levelDebug, "set filter rules to default\n")

	if router.outgoingFilter != nil {
		router.outgoingFilter.mu.Lock()
		router.outgoingFilter.rules = copyFilterRule(defaultFilterRule)
		router.outgoingFilter.mu.Unlock()
	}
	if router.incomingFilter != nil {
		router.incomingFilter.mu.Lock()
		router.incomingFilter.rules = copyFilterRule(defaultFilterRule)
		router.incomingFilter.mu.Unlock()
	}

	return "filter reset to default rules\r\n"
}

func (c *CmdCtx) showFilterRules(router *Router) string {
	router.FilterMu.RLock()
	enabled := router.Filter
	outFilter := router.outgoingFilter
	inFilter := router.incomingFilter
	router.FilterMu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Filter status: %s\r\n", map[bool]string{true: "enabled", false: "disabled"}[enabled]))

	if outFilter != nil {
		outFilter.mu.Lock()
		sb.WriteString("\nOutgoing filter rules:\r\n")
		sb.WriteString(c.formatFilterRules(&outFilter.rules))
		outFilter.mu.Unlock()
	}

	if inFilter != nil {
		inFilter.mu.Lock()
		sb.WriteString("\nIncoming filter rules:\r\n")
		sb.WriteString(c.formatFilterRules(&inFilter.rules))
		inFilter.mu.Unlock()
	}

	return sb.String()
}

func (c *CmdCtx) formatFilterRules(rules *FilterRule) string {
	var sb strings.Builder

	defaultCount := len(defaultFilterRule.Received)

	for i, seq := range rules.Received {
		var fwd, ans string
		if i < len(rules.Forwarded) {
			fwd = formatSequence(rules.Forwarded[i])
		} else {
			fwd = "-"
		}
		if i < len(rules.Answered) {
			ans = formatSequence(rules.Answered[i])
		} else {
			ans = "-"
		}

		prefix := "  "
		suffix := ""
		if i < defaultCount {
			prefix = "* "
			suffix = " (default)"
		}

		sb.WriteString(fmt.Sprintf("%s[%d] R: %s | F: %s | A: %s%s\r\n",
			prefix, i, formatSequence(seq), fwd, ans, suffix))
	}

	return sb.String()
}

func (c *CmdCtx) addFilterRule(router *Router, args []string) string {
	if len(args) < 2 {
		return "error: insufficient arguments\r\n" +
			   "usage: filter add ascii <received> [forwarded] [answered]\r\n" +
			   "	   filter add hex <received_hex> [forwarded_hex] [answered_hex]\r\n"
	}

	format := args[0]
	sequences := args[1:]

	var received, forwarded, answered Sequence
	var err error

	switch format {
	case "ascii":
		received = Sequence(sequences[0])
	case "hex":
		received, err = hexToBytes(sequences[0])
		if err != nil {
			return fmt.Sprintf("error: invalid received hex sequence: %v\r\n", err)
		}
	default:
		return fmt.Sprintf("error: unknown format '%s', use 'ascii' or 'hex'\r\n", format)
	}

	if len(sequences) > 1 && sequences[1] != "-" && sequences[1] != "" {
		switch format {
		case "ascii":
			forwarded = Sequence(sequences[1])
		case "hex":
			forwarded, err = hexToBytes(sequences[1])
			if err != nil {
				return fmt.Sprintf("error: invalid forwarded hex sequence: %v\r\n", err)
			}
		}
	}

	if len(sequences) > 2 && sequences[2] != "-" && sequences[2] != "" {
		switch format {
		case "ascii":
			answered = Sequence(sequences[2])
		case "hex":
			answered, err = hexToBytes(sequences[2])
			if err != nil {
				return fmt.Sprintf("error: invalid answered hex sequence: %v\r\n", err)
			}
		}
	}

	router.FilterMu.Lock()
	defer router.FilterMu.Unlock()

	if router.outgoingFilter != nil {
		router.outgoingFilter.mu.Lock()
		router.outgoingFilter.rules.Received = append(router.outgoingFilter.rules.Received, received)
		router.outgoingFilter.rules.Forwarded = append(router.outgoingFilter.rules.Forwarded, forwarded)
		router.outgoingFilter.rules.Answered = append(router.outgoingFilter.rules.Answered, answered)
		router.outgoingFilter.mu.Unlock()
	}
	if router.incomingFilter != nil {
		router.incomingFilter.mu.Lock()
		router.incomingFilter.rules.Received = append(router.incomingFilter.rules.Received, received)
		router.incomingFilter.rules.Forwarded = append(router.incomingFilter.rules.Forwarded, forwarded)
		router.incomingFilter.rules.Answered = append(router.incomingFilter.rules.Answered, answered)
		router.incomingFilter.mu.Unlock()
	}

	return "filter rule added\r\n"
}

func (c *CmdCtx) removeFilterRule(router *Router, indexStr string) string {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Sprintf("error: invalid index: %v\r\n", err)
	}

	router.FilterMu.Lock()
	defer router.FilterMu.Unlock()

	defaultCount := len(defaultFilterRule.Received)

	if index < defaultCount {
		return fmt.Sprintf("error: cannot remove default rule at index %d (read-only)\r\n", index)
	}

	if router.outgoingFilter != nil {
		router.outgoingFilter.mu.Lock()
		if index >= len(router.outgoingFilter.rules.Received) {
			router.outgoingFilter.mu.Unlock()
			return fmt.Sprintf("error: index %d out of range\r\n", index)
		}
		router.outgoingFilter.rules.Received = removeAt(router.outgoingFilter.rules.Received, index)
		if index < len(router.outgoingFilter.rules.Forwarded) {
			router.outgoingFilter.rules.Forwarded = removeAt(router.outgoingFilter.rules.Forwarded, index)
		}
		if index < len(router.outgoingFilter.rules.Answered) {
			router.outgoingFilter.rules.Answered = removeAt(router.outgoingFilter.rules.Answered, index)
		}
		router.outgoingFilter.mu.Unlock()
	}

	if router.incomingFilter != nil {
		router.incomingFilter.mu.Lock()
		if index < len(router.incomingFilter.rules.Received) {
			router.incomingFilter.rules.Received = removeAt(router.incomingFilter.rules.Received, index)
			if index < len(router.incomingFilter.rules.Forwarded) {
				router.incomingFilter.rules.Forwarded = removeAt(router.incomingFilter.rules.Forwarded, index)
			}
			if index < len(router.incomingFilter.rules.Answered) {
				router.incomingFilter.rules.Answered = removeAt(router.incomingFilter.rules.Answered, index)
			}
		}
		router.incomingFilter.mu.Unlock()
	}

	return fmt.Sprintf("filter rule %d removed\r\n", index)
}

func copyFilterRule(src FilterRule) FilterRule {
	dst := FilterRule{
		Received:  make([]Sequence, len(src.Received)),
		Forwarded: make([]Sequence, len(src.Forwarded)),
		Answered:  make([]Sequence, len(src.Answered)),
	}

	for i, seq := range src.Received {
		dst.Received[i] = make(Sequence, len(seq))
		copy(dst.Received[i], seq)
	}
	for i, seq := range src.Forwarded {
		dst.Forwarded[i] = make(Sequence, len(seq))
		copy(dst.Forwarded[i], seq)
	}
	for i, seq := range src.Answered {
		dst.Answered[i] = make(Sequence, len(seq))
		copy(dst.Answered[i], seq)
	}

	return dst
}

func removeAt(slice []Sequence, index int) []Sequence {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

func hexToBytes(s string) (Sequence, error) {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "0x", "")
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return Sequence(bytes), nil
}

func formatSequence(seq Sequence) string {
	if len(seq) == 0 {
		return "-"
	}

	if isPrintable(seq) {
		return fmt.Sprintf("\"%s\"", string(seq))
	}

	return "0x" + hex.EncodeToString(seq)
}

func isPrintable(seq Sequence) bool {
	if len(seq) == 0 {
		return false
	}
	for _, b := range seq {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

func (c *CmdCtx) log_serial(input string) string {

	if log_serial_in_progress {
		return fmt.Sprintf("Already in progress\r\n")
	}
	if input == "" {
		return fmt.Sprintf("no input file given\r\n")
	}
	items := strings.Split(input, " ")
	if len(items) != 1 {
		return fmt.Sprintf("Syntax error. Command has only an argument. it is the log file name.\r\n")
	}
	debugPrint(log.Printf, levelInfo, "log_serial command requested")
	n, err := (*(*c).monitor).router.GetFreePos()
	if err != nil {
		return fmt.Sprintf("no available channels: %s\r\n", err.Error())
	}
	(*(*c).monitor).router.AttachAt(n, SrcHuman)
	log_serial_in_progress = true
	go func(c *CmdCtx) {
		var buffer []byte
		(*(*c).monitor).router.DetachAt(n) // n is consistent, golang lifetime maintains it after parent terminates.

		f, err := os.Create(input)
		if err != nil {
			debugPrint(log.Printf, levelError, "Can't create file %s: %s", input, err.Error())
		}
		defer f.Close()

		debugPrint(log.Printf, levelInfo, "Goroutine started")
		inStrChan := (*(*c).monitor).router.In[n]

		go func() {
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
