package main

import (
	"log"
	"sort"
	"strings"
)

type CharFunc func(b byte, line *[]byte) []byte
type MonCtx struct {
	monitorConfig    map[string] string
	prompt           []byte
	HandleChar       [256] CharFunc
	commands         *CmdCtx
	monitorIn        <-chan byte
	monitorOut       chan<- byte
	router           *Router
}

func (m *MonCtx) HandleCharInit(){
	var HandleChar [256] CharFunc

	for i := 0; i <= 31; i++ {
		HandleChar[i] = m.ChDiscard
	}
	HandleChar[0x03] = m.ChCtrlC
	HandleChar[0x09] = m.ChTab
	for i := 32; i <= 127; i++ {
		HandleChar[i] = m.ChNormal
	}
	for i:=128;i<=255;i++ {
		HandleChar[i]=m.ChDiscard
	}
	HandleChar[0x0d]=m.ChEnter
	HandleChar[0x7f]=m.ChBackspace
	HandleChar[0x1b]=m.ChEscape
	m.HandleChar=HandleChar
}

func MonitorInit(monitorIn <-chan byte, monitorOut chan<- byte, monConfig map[string] string, r *Router, prompt string, maxFences, maxScrSess int) (*MonCtx) {
	debugPrint(log.Printf, levelDebug, "Monitor initialization\n")
	cmdctx := command_init(nil, maxFences, maxScrSess)
	cmdctx.fences["snmp"] = cmdctx.snmpSwitch
	cmdctx.fences["tasmota"] = cmdctx.tasmotaSwitch
	cmdctx.fences["beaker"] = cmdctx.beakerSwitch
	initEsc()
	m:= MonCtx{
		monitorConfig:  monConfig,
		prompt:         []byte(prompt),
		commands:       cmdctx,
		monitorIn:      monitorIn,
		monitorOut:     monitorOut,
		router:         r,
	}
	m.HandleCharInit()
	debugPrint(log.Printf, levelCrazy, "Created object MonCtx = %+v", m)
	cmdctx.monitor = &m
	return &m
}
func (m *MonCtx) doMonitor() {
	var line []byte
	debugPrint(log.Printf, levelDebug, "Starting Operations\n")
	out := m.prompt
	for {
		for _, c := range out {
			m.monitorOut <- c
		}
		b := <- m.monitorIn
		out = m.HandleChar[b](b, &line)
	}
	debugPrint(log.Printf, levelWarning, "Terminating Operations\n")
}

func (m *MonCtx) ChEnter(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelCrazy, "ChEnter %x '%s'\n", b,string(*line))
	out := "\n\r"
	cmd := strings.Split(string(*line), " ")
	key := cmd[0]
	args := strings.Join(cmd[1:]," ")
	debugPrint(log.Printf, levelCrazy, "key=%s args='%s'", key, args)
	if key!="" {
		if _, ok := m.commands.commands[key]; ok {
			out = out + m.commands.commands[key].Handler(args)
		} else {
			out = out + "Error!\r\n"
		}
	}
	*line = []byte{}
	return []byte(out + string(m.prompt))

}

func (m *MonCtx) ChNormal(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelCrazy, "ChNormal %x '%s'\n", b,string(*line))

	out := []byte{b}
	*line = append(*line,b)
	if Escape > 0 {
		Escape --
		if Escape == 0 {
			key := string((*line)[len(*line)-2:])
			if _, ok := EscapeFunc[key]; ok {
				debugPrint(log.Printf, levelDebug, "Escape sequence '%s'", key)
				EscapeFunc[key](line)
			}
			*line = (*line)[:len(*line)-2]
		}
		return nil

	}
        return out
}

func (m *MonCtx) ChBackspace(b byte, line *[]byte) []byte{
	var ret []byte

	debugPrint(log.Printf, levelCrazy, "ChBackspace %x '%s'\n", b,string(*line))
	oldLine := *line
	if len(oldLine) <= 0 {
		ret = nil
	} else {
		newLine := oldLine[:len(oldLine)-1]
		*line = newLine
		ret = []byte{8,32,8}
	}
	return ret
}

func (m *MonCtx) ChDiscard(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelWarning, "ChDiscard %x '%s'\n", b,string(*line))
        return nil
}

func (m *MonCtx) ChEscape(b byte, line *[]byte) []byte {
	debugPrint(log.Printf, levelInfo, "Escape enabled'\n")
	Escape = 2
	return nil
}

func (m *MonCtx) commandNames() []string {
	names := make([]string, 0, len(m.commands.commands))
	for name := range m.commands.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func longestCommonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[:i]
}

func (m *MonCtx) ChCtrlC(b byte, line *[]byte) []byte {
	debugPrint(log.Printf, levelDebug, "Drop the line'\n")
	*line = []byte{}
	return []byte("\r\n" + string(m.prompt))

}

func (m *MonCtx) ChTab(b byte, line *[]byte) []byte {
	lineStr := strings.TrimSpace(string(*line))
	prefix := lineStr
	rest := ""
	if idx := strings.Index(lineStr, " "); idx >= 0 {
		prefix = lineStr[:idx]
		rest = lineStr[idx:]
	}
	all := m.commandNames()
	var matches []string
	for _, name := range all {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	if len(matches) == 1 {
		completion := matches[0]
		if prefix == completion && (rest == "" || rest == " ") {
			return nil
		}
		if rest == "" {
			completion = completion + " "
		}
		*line = []byte(completion)
		return []byte(completion[len(prefix):])
	}
	lcp := matches[0]
	for i := 1; i < len(matches); i++ {
		lcp = longestCommonPrefix(lcp, matches[i])
	}
	if len(prefix) < len(lcp) {
		*line = []byte(lcp)
		return []byte(lcp[len(prefix):])
	}
	out := "\r\n"
	for _, name := range matches {
		out += name + " "
	}
	out += "\r\n" + string(m.prompt) + string(*line)
	return []byte(out)
}
