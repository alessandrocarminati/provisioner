package main

import (
	"bytes"
	"log"
	"sync"
)

var (
	CursorPositionQ          = Sequence("\x1b[6n")
	DeviceSpecsQ             = Sequence("\x1b[c")
	PrimaryDeviceAttributesQ = Sequence("\x1b[0c")
	TestQ                    = Sequence("Hello")
	CursorPositionF          = Sequence("\x1b[6n")
	DeviceSpecsF             = Sequence("\x1b[c")
	PrimaryDeviceAttributesF = Sequence("\x1b[0c")
	TestF                    = Sequence("hello")
	CursorPositionA          = Sequence("\x1b[?25;80R")
	DeviceSpecsA             = Sequence("\x1b[?64;1;2;4;6;15;22c")
	PrimaryDeviceAttributesA = Sequence("\x1b[?1;2c")
	TestA                    = Sequence("stocazzo")
)

type Sequence []byte

type StreamFilter struct {
	mu    sync.Mutex
	buf   []byte
	rules FilterRule
}

type FilterRule struct {
	Received  []Sequence
	Forwarded []Sequence
	Answered  []Sequence
}

var (
	defaultFilterRule = FilterRule{
		//		[]Sequence{CursorPositionQ, DeviceSpecsQ, PrimaryDeviceAttributesQ, TestQ},
		//		[]Sequence{CursorPositionF, DeviceSpecsF, PrimaryDeviceAttributesF, TestF},
		//		[]Sequence{CursorPositionA, DeviceSpecsA, PrimaryDeviceAttributesA, TestA},
		[]Sequence{TestQ},
		[]Sequence{TestF},
		[]Sequence{TestA},
	}
)

func (sf *StreamFilter) Feed(b byte) (toBroadcast []byte, injectToBoard []byte) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	debugPrint(log.Printf, levelDebug, "current buffer='%s', received='%s'\n", sf.buf, string(b))
	sf.buf = append(sf.buf, b)
	PrefixesFound := 0
	for i, seq := range sf.rules.Received {
		debugPrint(log.Printf, levelDebug, "Check against '%s'\n", string(seq))
		if bytes.HasPrefix(seq, sf.buf) {
			PrefixesFound++
			if bytes.Equal(sf.buf, seq) {
				toBroadcast = sf.rules.Forwarded[i]
				injectToBoard = sf.rules.Answered[i]
				sf.buf = []byte{}
				debugPrint(log.Printf, levelDebug, "match return intended toBroadcast='%s', injectToBoard='%s'\n", string(toBroadcast), string(injectToBoard))
				return // match return intended
			}
		}
	}
	if PrefixesFound > 0 {
		debugPrint(log.Printf, levelDebug, "partial match (%d)wait to see what happen\n", PrefixesFound)
		return nil, nil //partial match wait to see what happen
	}
	tmp := sf.buf
	sf.buf = []byte{}
	debugPrint(log.Printf, levelDebug, "no match return buffered buffer='%s'\n", sf.buf)
	return tmp, nil //no match return buffered
}
