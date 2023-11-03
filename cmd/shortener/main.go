package main

import (
	"net/http"
)

var storage = make(Storage)

func main() {
	if err := http.ListenAndServe(`:8080`, ShortenURLRouter()); err != nil {
		panic(err)
	}
}
