package main

import (
	"fmt"
	"net/http"
)

func main() {
	hub := NewHub()
	go hub.Run()

	serverMux := http.NewServeMux()

	// web page
	serverMux.Handle("/", http.FileServer(http.Dir("public")))

	// server websocked
	serverMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWS(hub, w, r)
	})

	fmt.Println("running server :8080")
	fmt.Println(http.ListenAndServe(":8080", serverMux))
}
