package main

import (
	"sync"
        "strings"
	"log"
)

type CharFunc func(b byte, line *[]byte) []byte
var monitorConfig map[string] string
var prompt []byte = []byte("> ")
var HandleChar [256] CharFunc


func HandleCharInit(){
	for i:=0;i<=31;i++ {
		HandleChar[i]=ChDiscard
	}
	for i:=32;i<=127;i++ {
		HandleChar[i]=ChNormal
	}
	for i:=128;i<=255;i++ {
		HandleChar[i]=ChDiscard
	}
	HandleChar[0x0d]=ChEnter
	HandleChar[0x7f]=ChBackspace
	HandleChar[0x1b]=ChEscape
}

func Monitor(monitorIn <-chan byte, monitorOut chan<- byte, monConfig map[string] string) {
	var wg sync.WaitGroup
	var line []byte

	fences=make(map[string]FenceFuncs, 20)

	monitorConfig=monConfig
	command_init()
	fences["snmp"] = snmpSwitch
	fences["tasmota"] = tasmotaSwitch
	fences["beaker"] = beakerSwitch
	initEsc()
	HandleCharInit()
	out := prompt

	wg.Add(1)
	go func() {
		for {
			for _, c := range out {
				monitorOut <- c
			}
			b := <- monitorIn
			out = HandleChar[b](b, &line)
		}
	}()
	wg.Wait()
}

func ChEnter(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelDebug, "ChEnter %x '%s'\n", b,string(*line))
	out := "\n\r"
	cmd := strings.Split(string(*line), " ")
	key := cmd[0]
	args := strings.Join(cmd[1:]," ")
	debugPrint(log.Printf, levelDebug, "key=%s args='%s'", key, args)
	if key!="" {
		if _, ok := commands[key]; ok {
			out = out + commands[key].Handler(args)
		} else {
			out = out + "Error!\r\n"
		}
	}
	*line = []byte{}
	return []byte(out + string(prompt))

}

func ChNormal(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelDebug, "ChNormal %x '%s'\n", b,string(*line))

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

func ChBackspace(b byte, line *[]byte) []byte{
	var ret []byte

	debugPrint(log.Printf, levelDebug, "ChBackspace %x '%s'\n", b,string(*line))
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

func ChDiscard(b byte, line *[]byte) []byte{
	debugPrint(log.Printf, levelDebug, "ChDiscard %x '%s'\n", b,string(*line))
        return nil
}

func ChEscape(b byte, line *[]byte) []byte{
        debugPrint(log.Printf, levelDebug, "Escape enabled'\n")
	Escape = 2
        return nil
}
