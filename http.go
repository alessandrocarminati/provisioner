package main
import (
	"net/http"
	"log"
)

func HTTPHandler(rootDir string, httpPort string) {
	debugPrint(log.Printf, levelWarning, "Starting http service on port: "+ httpPort)
	http.Handle("/", http.FileServer(http.Dir(rootDir)))

	http.HandleFunc("/api/example", func(w http.ResponseWriter, r *http.Request) {
		debugPrint(log.Printf, levelWarning, "This is an example API response.")
	})

	bind:=":"+httpPort
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		debugPrint(log.Printf, levelError, "Error starting HTTP server: %s", err.Error())
	} else {
	debugPrint(log.Printf, levelWarning, "http service is active on %s", bind)
	}
}

