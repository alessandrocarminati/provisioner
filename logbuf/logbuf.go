package logbuf

import (
	"runtime"
	"fmt"
//	"os"
)

type DebugLevels struct {
	Value	int
	Label	string
}

var (
	LevelPanic	= DebugLevels{0, "Panic  "}
	LevelError	= DebugLevels{1, "Error  "}
	LevelWarning	= DebugLevels{2, "Warning"}
	LevelNotice	= DebugLevels{3, "Notice "}
	LevelInfo	= DebugLevels{4, "Info   "}
	LevelDebug	= DebugLevels{5, "Debug  "}
)

func LogSprintf(level DebugLevels,  format string, a ...interface{}) string {
	var s string

	pc, _, _, ok := runtime.Caller(1)
	s = "?"
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			s = fn.Name()
		}
	}
	newformat := fmt.Sprintf("<%d>(%s)[" + s + "] ", level.Value, level.Label) + format
	return fmt.Sprintf(newformat,  a...)
}

