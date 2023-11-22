package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	log.Println("start app")
	helloHandler := func(w http.ResponseWriter, req *http.Request) { io.WriteString(w, "Hello, world!\n") }
	http.ListenAndServe("localhost:8080", helloHandler)
}
