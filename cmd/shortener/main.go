package main

import "net/http"

var storage = make(Storage)

func main() {
	parseFlags()

	if err := http.ListenAndServe(config.serverAddress, ShortenURLRouter()); err != nil {
		panic(err)
	}
}
