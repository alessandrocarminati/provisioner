package main

import (
	"os"
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strings"
	"strconv"
	"errors"
	"time"
)

type InstructionType int

const (
	LoadRegister InstructionType = iota
	Move
	Clear
	Fetch
	Compare
	Regex
	Append
	Output
	JE
	JNE
	JMP
	End
	Suffix
	NumInstructions
)

var InstructionMnemonics = [...]string{
	LoadRegister: "loadreg",
	Move:         "move",
	Clear:        "clear",
	Fetch:        "fetch",
	Compare:      "compare",
	Regex:        "regex",
	Append:       "append",
	Output:       "output",
	JE:           "je",
	JNE:          "jne",
	JMP:          "jmp",
	End:          "end",
	Suffix:       "suffix",
}

type Instruction struct {
        Type InstructionType
        Args []string
}


type Executor struct {
	accumulator   string
	registers     [16]string
	flag          bool
	pc            int
	instructions  []Instruction
	fetcher       func(chan byte) byte
	fetcherArg    chan byte
	putter        func(string)
	executed      int
}

func (e *Executor) setFetcher(f func(chan byte) byte ) {
	debugPrint(log.Printf, levelDebug, "Set new fetcher function")
	e.fetcher =f
}

func (e *Executor) setFetcherArg(a chan byte) {
	debugPrint(log.Printf, levelDebug, "Set new fetcher function argument")
        e.fetcherArg =a
}

func (e *Executor) fetch() byte {
	debugPrint(log.Printf, levelCrazy, "executing fetcher function")
	return e.fetcher(e.fetcherArg)
}

func (e *Executor) setPutter(f func(string) ) {
	debugPrint(log.Printf, levelDebug, "Set new putter function")
	e.putter =f
}

func (e *Executor) put(s string) {
	debugPrint(log.Printf, levelCrazy, "executing putter function")
	e.putter(s+"\r\n")
	return
}

func (e *Executor) Parse(input []string) (error) {
	debugPrint(log.Printf, levelInfo, "Parsing new program")
	instructions := make([]Instruction, 0)
	labels := make(map[string]int)

	debugFmtParsedLine:="line %d, Adding new %s %s"

	for linen, line := range input {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		labelRegex := regexp.MustCompile(`^(\w+):$`)
		labelMatches := labelRegex.FindStringSubmatch(line)
		if len(labelMatches) > 1 {
			debugPrint(log.Printf, levelDebug, "line %d: Adding new label %s at instr[%d]", linen, labelMatches[1], len(instructions))
			labels[labelMatches[1]] = len(instructions)
			continue
		}

		parts := splitCmdArg(line)

		var instrType InstructionType
//		debugPrint(log.Printf, levelWarning, "%d ===> %s %s <-- %s", linen, parts[0], parts[1], InstructionMnemonics[LoadRegister])

		switch strings.ToLower(parts[0]) {
		case InstructionMnemonics[LoadRegister]:
			instrn := InstructionMnemonics[Move]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = LoadRegister
		case InstructionMnemonics[Move]:
			instrn := InstructionMnemonics[Move]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Move
		case InstructionMnemonics[Clear]:
			instrn := InstructionMnemonics[Clear]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Clear
		case InstructionMnemonics[Fetch]:
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, InstructionMnemonics[Fetch], parts[1])
			instrType = Fetch
		case InstructionMnemonics[Compare]:
			instrn := InstructionMnemonics[Compare]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Compare
		case InstructionMnemonics[Suffix]:
			instrn := InstructionMnemonics[Suffix]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Suffix
		case InstructionMnemonics[Regex]:
			instrn := InstructionMnemonics[Regex]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Regex
		case InstructionMnemonics[Append]:
			instrn := InstructionMnemonics[Append]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Append
		case InstructionMnemonics[Output]:
			instrn := InstructionMnemonics[Output]
			_, err := getRegisterIndex(parts[1])
			if err != nil {
				s:=fmt.Sprintf("line %d, %s: %s", linen, instrn, err.Error())
				debugPrint(log.Printf, levelError, s)
				return errors.New(s)
			}
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = Output
		case InstructionMnemonics[JE]:
			instrn := InstructionMnemonics[JE]
			instrType = JE
			labelsIndex, ok := labels[parts[1]]
			if !ok {
				return errors.New(fmt.Sprintf("Unknown label %s at %d", parts[1], linen))
			}
			parts[1] = fmt.Sprint(labelsIndex)
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
		case InstructionMnemonics[JNE]:
			instrn := InstructionMnemonics[JNE]
			instrType = JNE
			labelsIndex, ok := labels[parts[1]]
			if !ok {
				return errors.New(fmt.Sprintf("Unknown label %s at %d", parts[1], linen))
			}
			parts[1] = fmt.Sprint(labelsIndex)
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
		case InstructionMnemonics[JMP]:
			instrn := InstructionMnemonics[JMP]
			instrType = JMP
			labelsIndex, ok := labels[parts[1]]
			if !ok {
				return errors.New(fmt.Sprintf("Unknown label %s at %d", parts[1], linen))
			}
			parts[1] = fmt.Sprint(labelsIndex)
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
		case InstructionMnemonics[End]:
			instrn := InstructionMnemonics[End]
			debugPrint(log.Printf, levelDebug, debugFmtParsedLine, linen, instrn, parts[1])
			instrType = End
		default:
			s:=fmt.Sprintf("unknown instruction at %d: %s", linen+1, line)
			debugPrint(log.Printf, levelError, s)
			return errors.New(s)
		}
		instructions = append(instructions, Instruction{Type: instrType, Args: parts[1:]})
	}
	debugPrint(log.Printf, levelInfo, "Program parsed, no errors")
	e.instructions = instructions
	return nil
}

func instr2str(i Instruction) string {
	mnemonic := InstructionMnemonics[i.Type]

	arg1 := ""
	if i.Args[0] != "" {
		arg1 = " " + i.Args[0]
	}

	arg2 := ""
	if i.Args[1] != "" {
		arg2 = "," + i.Args[1]
	}

	return fmt.Sprintf("%s%s%s", mnemonic, arg1, arg2)
}


func (e *Executor) DumpCPU(instructions []Instruction) {

	s:= fmt.Sprintf("[%03d]: %s\n", e.pc, instr2str(instructions[e.pc]))
	s = s+ fmt.Sprintf("Accumulator='%s'\n", e.accumulator)
	for i, r := range e.registers {
		s = s + fmt.Sprintf("R[%d]='%s' ", i, r)
	}
	s = s + fmt.Sprintf("\nFlags=%t, executed: %d\n", e.flag, e.executed)
	debugPrint(log.Printf, levelDebug, s)
}

func (e *Executor) Execute(limit int) error {

	if (e.fetcher==nil) || (e.fetcherArg == nil) {
		s:="Executor not initialyzed"
		debugPrint(log.Printf, levelError, s)
		return errors.New(s)
	}

	e.executed=0
	for e.pc < len(e.instructions) {
		instr := e.instructions[e.pc]
		if e.executed > limit {
			s:=fmt.Sprintf("Limit %d instrs reached!", limit)
			debugPrint(log.Printf, levelError, s)
			return errors.New(s)
		}
		e.DumpCPU(e.instructions)
		e.pc++
		switch instr.Type {
		case LoadRegister:
			reg, _ := getRegisterIndex(instr.Args[0])
			e.registers[reg] = instr.Args[1]
		case Move:
			reg, _ := getRegisterIndex(instr.Args[0])
			e.registers[reg] = e.accumulator
		case Clear:
			e.accumulator = ""
		case Fetch:
			e.accumulator = e.accumulator +  string(e.fetch())
		case Compare:
			reg, _ := getRegisterIndex(instr.Args[0])
			if e.accumulator == e.registers[reg] {
				e.flag = true
			} else {
				e.flag = false
			}
		case Suffix:
			reg, _ := getRegisterIndex(instr.Args[0])
			if strings.HasSuffix(e.accumulator, e.registers[reg]) {
				e.flag = true
			} else {
				e.flag = false
			}
		case Regex:
			reg, _ := getRegisterIndex(instr.Args[0])
			Regex := regexp.MustCompile(e.registers[reg])
			Matches := Regex.FindStringSubmatch(e.accumulator)
			if len(Matches) > 1 {
				e.flag = true
			} else {
				e.flag = false
			}
		case Append:
			reg, _ := getRegisterIndex(instr.Args[0])
			e.registers[reg] += e.accumulator
		case Output:
			reg, _ := getRegisterIndex(instr.Args[0])
			e.put(e.registers[reg])
		case JE:
			labelIdx, _ := strconv.Atoi(instr.Args[0])
			if e.flag {
				e.pc = labelIdx
			}
		case JNE:
			labelIdx, _ := strconv.Atoi(instr.Args[0])
			if !e.flag {
				e.pc = labelIdx
			}
		case JMP:
			labelIdx, _ := strconv.Atoi(instr.Args[0])
			e.pc = labelIdx
		case End:
			return nil
		}
		e.executed++
	}
	return nil
}

func getRegisterIndex(arg string) (int, error) {
	var reg = regexp.MustCompile(`^[rR][0-9]{1,2}$`)
	err:=fmt.Errorf("invalid register: %s", arg)

	debugPrint(log.Printf, levelDebug, "Applying regex at '%s'", arg)
	if !reg.MatchString(arg) {
		return -1, err
	}

	debugPrint(log.Printf, levelDebug, "Convert number '%s'",arg[1:] )
	num, err := strconv.Atoi(arg[1:])
	if err != nil {
		debugPrint(log.Printf, levelError, "Convert error: %s",  err.Error())
		return -1, err
	}
	if num >= 16 {
		debugPrint(log.Printf, levelError, "Num too large: %d",  num)
		return -1, err
	}
	debugPrint(log.Printf, levelDebug, "Return %d", num )
	return num, nil
}

func TextRead(fn string) ([]string, error) {
	var lines []string

	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
func einit(fn string, In <-chan byte, Out chan<- byte) (Executor, error){
        input:=make(chan byte, 4096)

        executor := Executor{}
        executor.setFetcher(peek)
        executor.setFetcherArg(input)
        executor.setPutter(func(s string) {
				for i:=0;i<len(s);i++ {
					Out <- byte(s[i])
				}
				for len(Out) > 0 { // warning! out channel can be used for other actors
					time.Sleep(50 * time.Millisecond)
				}
			})
	lines, err :=TextRead(fn)
	if err!= nil {
		debugPrint(log.Printf, levelError, "Error in reading assm file: %s", err.Error())
		return Executor{}, err
	}
	err = executor.Parse(lines)
	if err!= nil {
		debugPrint(log.Printf, levelError, "Error in parsing assm file: %s", err.Error())
		return Executor{}, err
	}

	go func() {
		for {
			data := <-In
			input <- data
		}
	}()
	return executor, nil
}

func peek(i chan byte) byte {
	b:= <- i
	return b
}

func splitCmdArg(input string) []string {
	debugPrint(log.Printf, levelDebug, "Operate on '%s'", input)
	input = strings.TrimSpace(input)

	debugPrint(log.Printf, levelDebug, "Removing leading space -> '%s'", input)
	parts := strings.SplitN(input, " ", 2)

	debugPrint(log.Printf, levelDebug, "Spilitting")

	switch len(parts) {
	case 1:
		debugPrint(log.Printf, levelDebug, "Command only: '%s'", parts[0])
		return []string{parts[0], "", ""}
	case 2:
		debugPrint(log.Printf, levelDebug, "Command and arg: '%s' '%s'", parts[0], parts[1])
		if strings.Contains(parts[1], ",") {
			innerParts := strings.SplitN(parts[1], ",", 2)
			debugPrint(log.Printf, levelDebug, "Spilitting arg: '%s' '%s', '%s'", parts[0], innerParts[0], innerParts[1])
			return []string{parts[0], innerParts[0], innerParts[1]}
		}
		return []string{parts[0], parts[1], ""}
	default:
		return []string{"", "", ""}
	}
}
