package main
import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
 	"log"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type ExecutionState int
type TermType int

const (
	Undefined ExecutionState = iota
	Running
	TerminatedWithError
	TerminatedOk
)
const (
	UndefinedTerm TermType = iota
	CharOriented
	LineOriented
)

type ScriptGwData struct {
	state         ExecutionState
	TType         TermType
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

func generateRandomToken(length int) (string, error) {
    randomBytes := make([]byte, length)
    _, err := rand.Read(randomBytes)
    if err != nil {
        return "", err
    }

    token := hex.EncodeToString(randomBytes)

    return token, nil
}

func (gw *ScriptGwData) charOrientedWrite(stdinPipe io.WriteCloser, jisatsu *bool) {
	t, _ := generateRandomToken(5)
	debugPrint(log.Printf, levelDebug, "Starting id:%s'", t)
	defer stdinPipe.Close()
	for b := range gw.In {
		if !(*jisatsu) {
			break
		}
		debugPrint(log.Printf, levelDebug, "received %d",b)
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
	debugPrint(log.Printf, levelDebug, "shutting down id:%s'", t)
}

func (gw *ScriptGwData) lineOrientedWrite(stdinPipe io.WriteCloser, jisatsu *bool) {
	var tosend []byte

	t, _ := generateRandomToken(5)
	debugPrint(log.Printf, levelDebug, "Starting id:%s'", t)
	defer stdinPipe.Close()
	for b := range gw.In {
		if !(*jisatsu) {
			break
		}
		if b==13 {
			tosend = append(tosend, 10)
			tosend = append(tosend, 13)
			_, err := stdinPipe.Write(tosend)
			if err != nil {
				debugPrint(log.Printf, levelError, "Error writing to stdin: %s", err.Error())
				return
			}
			tosend = []byte{}
		} else  {
		tosend = append(tosend, b)
		}
		debugPrint(log.Printf, levelCrazy, "(%s)received %02x current='%s'", t, b, string(tosend))
	}
	debugPrint(log.Printf, levelDebug, "shutting down id:%s'", t)
}

func (gw *ScriptGwData) ScriptGwExec() {
	debugPrint(log.Printf, levelInfo, "Script '%s' launched", gw.cmdFn)
	gw.updateState(Running)
	jisatsu := true

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
	debugPrint(log.Printf, levelDebug, "Script Started!")

	switch gw.TType {
		case CharOriented:
			debugPrint(log.Printf, levelDebug, "Starting chars based stdin manager")
			go gw.charOrientedWrite(stdinPipe, &jisatsu)
		case LineOriented:
			debugPrint(log.Printf, levelDebug, "Starting lines based stdin manager")
			go gw.lineOrientedWrite(stdinPipe, &jisatsu)
		default:
			gw.updateState(TerminatedWithError)
			return
	}
	go func() {
		debugPrint(log.Printf, levelDebug, "stdout is starting!")
		defer stdoutPipe.Close()
		reader := bufio.NewReader(stdoutPipe)
		for jisatsu {
//			debugPrint(log.Printf, levelDebug, "stdout is reading!")
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
		debugPrint(log.Printf, levelDebug, "stdout is shutting down!")
	}()

	go func() {
		debugPrint(log.Printf, levelDebug, "stderr is starting!")
		defer stderrPipe.Close()
		reader := bufio.NewReader(stderrPipe)
		for jisatsu {
//			debugPrint(log.Printf, levelDebug, "stderr is reading!")
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
		debugPrint(log.Printf, levelDebug, "stderr is shutting down!")
	}()

	err = cmd.Wait()
	jisatsu = false
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

func ScriptGwInit(cmdFn string, ttype TermType, In <-chan byte, Out chan<- byte) *ScriptGwData {
	return  &ScriptGwData{
		state:        Undefined,
		cmdFn:        cmdFn,
		TType:        ttype,
		In:           In,
		Out:          Out,
	}

}
