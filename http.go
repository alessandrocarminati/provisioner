package main
import (
	"net/http"
	"log"
	"fmt"
)

func HTTPHandler(rootDir string, httpPort string) {
	http.Handle("/", http.FileServer(http.Dir(rootDir)))

	http.HandleFunc("/api/example", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "This is an example API response.")
	})

	err := http.ListenAndServe(":"+httpPort, nil)
	if err != nil {
		log.Fatal("Error starting HTTP server:", err)
	}
}

