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
	//	TestQ                    = Sequence("Bella")
	CursorPositionF          = Sequence("") // The intent here is to prevent client to answer late, because latency introduced by the tunnel
	DeviceSpecsF             = Sequence("")
	PrimaryDeviceAttributesF = Sequence("")
	//	TestF                    = Sequence("bella")
	CursorPositionA          = Sequence("\x1b[?25;80R") // provisional answer, 80x25
	DeviceSpecsA             = Sequence("\x1b[?64;1;2;4;6;15;22c")
	PrimaryDeviceAttributesA = Sequence("\x1b[?1;2c")

// TestA                    = Sequence("Dylan")
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
		[]Sequence{CursorPositionQ, DeviceSpecsQ, PrimaryDeviceAttributesQ},
		[]Sequence{CursorPositionF, DeviceSpecsF, PrimaryDeviceAttributesF},
		[]Sequence{CursorPositionA, DeviceSpecsA, PrimaryDeviceAttributesA},
		//		[]Sequence{TestQ},
		//		[]Sequence{TestF},
		//		[]Sequence{TestA},
	}
)

func (sf *StreamFilter) Feed(b byte) (toBroadcast []byte, injectToBoard []byte) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	debugPrint(log.Printf, levelCrazy, "current buffer='%s', received='%s'\n", sf.buf, string(b))
	sf.buf = append(sf.buf, b)
	PrefixesFound := 0
	for i, seq := range sf.rules.Received {
		debugPrint(log.Printf, levelCrazy, "Check against '%s'\n", string(seq))
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
	debugPrint(log.Printf, levelCrazy, "no match return buffered buffer='%s'\n", sf.buf)
	return tmp, nil //no match return buffered
}
