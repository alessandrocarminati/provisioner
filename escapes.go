package main

import (
	"log"
)

type EscFunc func(line *[]byte)

var Escape = 0

var EscapeFunc = make(map[string]EscFunc)

func initEsc() {
	EscapeFunc["[A"] = arrowUp
	EscapeFunc["[B"] = arrowDown
	EscapeFunc["[C"] = arrowRight
	EscapeFunc["[D"] = arrowLeft
}

func arrowUp(line *[]byte) {
	debugPrint(log.Printf, levelDebug, "escape char Arrow up")
}
func arrowDown(line *[]byte) {
	debugPrint(log.Printf, levelDebug, "escape char Arrow Down")
}
func arrowLeft(line *[]byte) {
	debugPrint(log.Printf, levelDebug, "escape char Arrow Left")
}
func arrowRight(line *[]byte) {
	debugPrint(log.Printf, levelDebug, "escape char Arrow Right")
}
