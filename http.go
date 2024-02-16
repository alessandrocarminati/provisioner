package main
import (
	"net/http"
	"log"
	"fmt"
)



// HTTPHandler serves files via HTTP and exports APIs.
func HTTPHandler(rootDir string, httpPort string) {
    http.Handle("/", http.FileServer(http.Dir(rootDir)))

    // Define API handlers
    http.HandleFunc("/api/example", func(w http.ResponseWriter, r *http.Request) {
        // Your API logic here
        fmt.Fprintf(w, "This is an example API response.")
    })

    // Start the HTTP server
    err := http.ListenAndServe(":"+httpPort, nil)
    if err != nil {
        log.Fatal("Error starting HTTP server:", err)
    }
}

