package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}

	mux.Handle("/", http.FileServer(http.Dir(".")))

	fmt.Printf("Server starting on http://%s\n", server.Addr)
	server.ListenAndServe()
}
