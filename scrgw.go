package main
import (
	"bufio"
//	"fmt"
	"log"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type ExecutionState int

const (
	Undefined ExecutionState = iota
	Running
	TerminatedWithError
	TerminatedOk
)

type ScriptGwData struct {
	state         ExecutionState
	mu            sync.Mutex
	cmdFn         string
	In            <-chan byte
	Out           chan<- byte
}

func (gw *ScriptGwData) GetState() string {
	switch gw.state {
	case Undefined: return "Undefined"
	case Running: return "Running"
	case TerminatedWithError: return "TerminatedWithError"
	case TerminatedOk: return "TerminatedOk"
	}
	return "Inconsistent"
}

func (gw *ScriptGwData) ScriptGwExec() {
	debugPrint(log.Printf, levelInfo, "Script '%s' launched", gw.cmdFn)
	gw.updateState(Running)

	cmdParts := splitCmdString(gw.cmdFn)

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		debugPrint(log.Printf, levelError, "Error creating stdin pipe: %s", err.Error())
		gw.updateState(TerminatedWithError)
		return
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		debugPrint(log.Printf, levelError, "Error creating stdout pipe: %s", err.Error())
		gw.updateState(TerminatedWithError)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		debugPrint(log.Printf, levelError, "Error creating stderr pipe: %s", err.Error())
		gw.updateState(TerminatedWithError)
		return
	}

	if err := cmd.Start(); err != nil {
		debugPrint(log.Printf, levelError, "Error starting command: %s", err.Error())
		gw.updateState(TerminatedWithError)
		return
	}

	go func() {
		defer stdinPipe.Close()
		for b := range gw.In {
			debugPrint(log.Printf, levelCrazy, "received %d",b)
			tosend := []byte{b}
			if b==13 {
				tosend = []byte{10, b}
			}
			_, err := stdinPipe.Write(tosend)
			if err != nil {
				debugPrint(log.Printf, levelError, "Error writing to stdin: %s", err.Error())
				return
			}
		}
	}()

	go func() {
		defer stdoutPipe.Close()
		reader := bufio.NewReader(stdoutPipe)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				debugPrint(log.Printf, levelError, "Error reading from stdout: %s", err.Error())
				gw.updateState(TerminatedWithError)
				return
			}
			debugPrint(log.Printf, levelCrazy, "reading from stdout: %s", line)
			for _, b := range []byte(line) {
				if b==10 {
					gw.Out <- 13
				}
				gw.Out <- b
			}
		}
	}()

	go func() {
		defer stderrPipe.Close()
		reader := bufio.NewReader(stderrPipe)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				debugPrint(log.Printf, levelError, "Error reading from stderr: %s", err.Error())
				gw.updateState(TerminatedWithError)
				return
			}
			debugPrint(log.Printf, levelInfo, "SCRIPT: %s", line)
		}
	}()

	err = cmd.Wait()
	if err != nil {
		debugPrint(log.Printf, levelError, "Command execution error: %s", err.Error())
		gw.updateState(TerminatedWithError)
		return
	}

	gw.updateState(TerminatedOk)
}

func splitCmdString(cmdFn string) []string {
	return strings.Fields(cmdFn)
}

func (gw *ScriptGwData) updateState(newState ExecutionState) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.state = newState
}

func ScriptGwInit(cmdFn string, In <-chan byte, Out chan<- byte) *ScriptGwData {
	return  &ScriptGwData{
		state:        Undefined,
		cmdFn:        cmdFn,
		In:           In,
		Out:          Out,
	}

}
