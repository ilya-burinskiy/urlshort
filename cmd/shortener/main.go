package main

import "net/http"

func main() {
	parseFlags()

	if err := http.ListenAndServe(config.serverAddress, ShortenURLRouter()); err != nil {
		panic(err)
	}
}
