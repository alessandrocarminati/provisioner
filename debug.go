package main

import (
	"runtime"
	"fmt"
	"os"
)

type DebugLevels struct {
	Value	int
	Label	string
}
var (
	levelPanic	= DebugLevels{0, "Panic  "}
	levelError	= DebugLevels{1, "Error  "}
	levelWarning	= DebugLevels{2, "Warning"}
	levelNotice	= DebugLevels{3, "Notice "}
	levelInfo	= DebugLevels{4, "Info   "}
	levelDebug	= DebugLevels{5, "Debug  "}
)

var DebugLevel int = 2

type PrintFunc func(format string, a ...interface{})

func debugPrint(printFunc PrintFunc, level DebugLevels,  format string, a ...interface{}) {
	var s string

	if level.Value<=DebugLevel {
		pc, _, _, ok := runtime.Caller(1)
		s = "?"
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				s = fn.Name()
			}
		}
		newformat := fmt.Sprintf("(%s)[" + s + "] ", level.Label) + format
		printFunc(newformat,  a...)
		if level.Value == 0 {
			os.Exit(-1)
		}
	}
}
