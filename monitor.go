package main

import (
        "strings"
	"log"
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

	for i:=0;i<=31;i++ {
		HandleChar[i]=m.ChDiscard
	}
	for i:=32;i<=127;i++ {
		HandleChar[i]=m.ChNormal
	}
	for i:=128;i<=255;i++ {
		HandleChar[i]=m.ChDiscard
	}
	HandleChar[0x0d]=m.ChEnter
	HandleChar[0x7f]=m.ChBackspace
	HandleChar[0x1b]=m.ChEscape
	m.HandleChar=HandleChar
}

func MonitorInit(monitorIn <-chan byte, monitorOut chan<- byte, monConfig map[string] string, r *Router, prompt string, maxFences int) (*MonCtx) {
	debugPrint(log.Printf, levelDebug, "Monitor initialization\n")
	fences:=make(map[string]FenceFuncs, maxFences)
	cmdctx := command_init(nil, fences)
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

func (m *MonCtx) ChEscape(b byte, line *[]byte) []byte{
        debugPrint(log.Printf, levelInfo, "Escape enabled'\n")
	Escape = 2
        return nil
}
