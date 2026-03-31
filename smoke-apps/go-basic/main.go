package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "go-basic smoke app")
	})

	fmt.Println("go-basic listening on", port)
	_ = http.ListenAndServe(":"+port, nil)
}
