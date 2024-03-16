package main
import(
        "log"
	)
type EscFunc func(line *[]byte)

var Escape = 0

var EscapeFunc = make(map[string] EscFunc)

func initEsc(){
	EscapeFunc["[A"] = arrowUp
	EscapeFunc["[B"] = arrowDown
	EscapeFunc["[C"] = arrowRight
	EscapeFunc["[D"] = arrowLeft
}

func arrowUp(line *[]byte){
	log.Println("Arrow up")
}
func arrowDown(line *[]byte){
	log.Println("Arrow down")
}
func arrowLeft(line *[]byte){
	log.Println("Arrow Left")
}
func arrowRight(line *[]byte){
	log.Println("Arrow Right")
}

