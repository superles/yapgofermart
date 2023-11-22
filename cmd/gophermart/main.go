package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	log.Println("start app")
	helloHandler := func(w http.ResponseWriter, req *http.Request) { io.WriteString(w, "Hello, world!\n") }
	http.HandleFunc("/hello", helloHandler)
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Println(err.Error())
	}
}
